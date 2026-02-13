package logger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// LogOutput 定义日志输出目标
type LogOutput int

const (
	OutputNone     LogOutput                     = 0         // 不输出
	OutputFile     LogOutput                     = 1 << iota // 仅文件 (1)
	OutputTerminal                                           // 仅终端 (2)
	OutputBoth     = OutputFile | OutputTerminal             // 文件和终端 (3)
)

type LogConfig struct {
	// 输出控制
	Output LogOutput // 输出目标：OutputFile, OutputTerminal, OutputBoth

	// 文件相关配置
	Dir            string // 日志目录
	FilenamePrefix string // 文件名前缀
	RetainDays     int    // 保留天数

	// 通用配置
	Format    LogFormat  // 日志格式：text, json
	Level     slog.Level // 日志级别
	AddSource bool       // 是否添加源码位置
}

var (
	loggerOnce      sync.Once
	onceLogger      *slog.Logger
	onceFileHandler *DailyFileHandler
	initErr         error
)

// InitLogger 初始化日志系统
func InitLogger(cfg *LogConfig) (*slog.Logger, error) {
	loggerOnce.Do(func() {
		if err := cfg.validate(); err != nil {
			initErr = fmt.Errorf("invalid config: %w", err)
			return
		}

		var handlers []slog.Handler

		// 创建文件 handler
		if cfg.Output&OutputFile != 0 {
			fileHandler, err := NewFileHandler(cfg)
			if err != nil {
				initErr = fmt.Errorf("create file handler failed: %w", err)
				return
			}
			onceFileHandler = fileHandler
			handlers = append(handlers, fileHandler)
		}

		// 创建终端 handler
		if cfg.Output&OutputTerminal != 0 {
			termHandler := NewTerminalHandler(os.Stderr, cfg)
			handlers = append(handlers, termHandler)
		}

		if len(handlers) == 0 {
			initErr = fmt.Errorf("no output target specified")
			return
		}

		// 创建多路 handler
		var finalHandler slog.Handler
		if len(handlers) == 1 {
			finalHandler = handlers[0]
		} else {
			finalHandler = NewMultiHandler(handlers...)
		}

		onceLogger = slog.New(finalHandler)
		slog.SetDefault(onceLogger)
	})

	if initErr != nil {
		return nil, initErr
	}

	return onceLogger, nil
}

// GetLogger 获取日志实例
func GetLogger() *slog.Logger {
	if onceLogger != nil {
		return onceLogger
	}

	// 如果未初始化，使用默认配置（仅终端输出）
	_, _ = InitLogger(&LogConfig{
		Output: OutputTerminal,
	})

	if onceLogger != nil {
		return onceLogger
	}

	return slog.Default()
}

// Close 关闭日志系统，释放资源
func Close() error {
	if onceFileHandler != nil {
		return onceFileHandler.Close()
	}
	return nil
}

// validate 验证并设置默认值
func (cfg *LogConfig) validate() error {
	// 验证输出目标
	if cfg.Output == OutputNone {
		return fmt.Errorf("output target cannot be OutputNone")
	}

	// 文件相关配置
	if cfg.Output&OutputFile != 0 {
		if cfg.Dir == "" {
			cfg.Dir = "./logs"
		}
		if cfg.FilenamePrefix == "" {
			cfg.FilenamePrefix = "app"
		}
		if cfg.RetainDays <= 0 {
			cfg.RetainDays = 30
		}
	}

	if cfg.Format == "" {
		cfg.Format = LogFormatText
	}

	if cfg.Format != LogFormatText && cfg.Format != LogFormatJSON {
		return fmt.Errorf("unsupported log format: %s", cfg.Format)
	}

	return nil
}

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &MultiHandler{handlers: handlers}
}

func (mh *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range mh.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (mh *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error

	for _, h := range mh.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}

		if err := h.Handle(ctx, r); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (mh *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(mh.handlers))
	for i, h := range mh.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: handlers}
}

func (mh *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(mh.handlers))
	for i, h := range mh.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: handlers}
}

type DailyFileHandler struct {
	mu          sync.Mutex
	cfg         *LogConfig
	curDate     string
	curFile     *os.File
	curInner    slog.Handler
	cleanTicker *time.Ticker
	cleanDone   chan struct{}
	closeOnce   sync.Once
}

func NewFileHandler(cfg *LogConfig) (*DailyFileHandler, error) {
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory failed: %w", err)
	}

	handler := &DailyFileHandler{
		cfg:       cfg,
		cleanDone: make(chan struct{}),
	}

	if err := handler.rotateIfNeeded(time.Now()); err != nil {
		return nil, err
	}

	// 启动定时清理任务
	handler.cleanTicker = time.NewTicker(24 * time.Hour)
	go handler.cleanOldLogsLoop()

	return handler, nil
}

func (h *DailyFileHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.cfg.Level <= level
}

func (h *DailyFileHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.rotateIfNeeded(r.Time); err != nil {
		return err
	}

	if h.curInner == nil {
		return fmt.Errorf("file handler not initialized")
	}

	return h.curInner.Handle(ctx, r)
}

func (h *DailyFileHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.curInner == nil {
		return h
	}

	return &fileHandlerWrapper{
		original: h,
		inner:    h.curInner.WithAttrs(attrs),
	}
}

func (h *DailyFileHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.curInner == nil {
		return h
	}

	return &fileHandlerWrapper{
		original: h,
		inner:    h.curInner.WithGroup(name),
	}
}

func (h *DailyFileHandler) Close() error {
	var err error
	h.closeOnce.Do(func() {
		if h.cleanTicker != nil {
			h.cleanTicker.Stop()
		}
		if h.cleanDone != nil {
			close(h.cleanDone)
		}

		h.mu.Lock()
		defer h.mu.Unlock()

		if h.curFile != nil {
			err = h.curFile.Close()
			h.curFile = nil
			h.curInner = nil
		}
	})
	return err
}

func (h *DailyFileHandler) rotateIfNeeded(t time.Time) error {
	date := t.Format("2006-01-02")
	if date == h.curDate && h.curInner != nil {
		return nil
	}

	// 关闭旧文件
	if h.curFile != nil {
		if err := h.curFile.Close(); err != nil {
			slog.Warn("failed to close old log file", "error", err)
		}
		h.curFile = nil
		h.curInner = nil
	}

	// 创建新文件
	filename := fmt.Sprintf("%s-%s.log", h.cfg.FilenamePrefix, date)
	path := filepath.Join(h.cfg.Dir, filename)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}

	opts := &slog.HandlerOptions{
		Level:     h.cfg.Level,
		AddSource: h.cfg.AddSource,
	}

	var inner slog.Handler
	switch h.cfg.Format {
	case LogFormatJSON:
		inner = slog.NewJSONHandler(file, opts)
	default:
		inner = slog.NewTextHandler(file, opts)
	}

	h.curDate = date
	h.curFile = file
	h.curInner = inner

	return nil
}

func (h *DailyFileHandler) cleanOldLogsLoop() {
	// 立即执行一次清理
	h.cleanOldLogs()

	for {
		select {
		case <-h.cleanTicker.C:
			h.cleanOldLogs()
		case <-h.cleanDone:
			return
		}
	}
}

func (h *DailyFileHandler) cleanOldLogs() {
	if h.cfg.RetainDays <= 0 {
		return
	}

	entries, err := os.ReadDir(h.cfg.Dir)
	if err != nil {
		slog.Warn("failed to read log directory", "error", err, "dir", h.cfg.Dir)
		return
	}

	now := time.Now()
	prefix := h.cfg.FilenamePrefix + "-"
	suffix := ".log"
	cutoff := now.AddDate(0, 0, -h.cfg.RetainDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}

		// 提取日期部分
		dateStr := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if date.Before(cutoff) {
			path := filepath.Join(h.cfg.Dir, name)
			if err := os.Remove(path); err != nil {
				slog.Warn("failed to remove old log file",
					"file", path, "error", err)
			} else {
				slog.Info("removed old log file", "file", path)
			}
		}
	}
}

// fileHandlerWrapper 包装器，解决 WithAttrs/WithGroup 的资源共享问题
type fileHandlerWrapper struct {
	original *DailyFileHandler
	inner    slog.Handler
}

func (w *fileHandlerWrapper) Enabled(ctx context.Context, level slog.Level) bool {
	return w.original.Enabled(ctx, level)
}

func (w *fileHandlerWrapper) Handle(ctx context.Context, r slog.Record) error {
	w.original.mu.Lock()
	defer w.original.mu.Unlock()

	if err := w.original.rotateIfNeeded(r.Time); err != nil {
		return err
	}

	return w.inner.Handle(ctx, r)
}

func (w *fileHandlerWrapper) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &fileHandlerWrapper{
		original: w.original,
		inner:    w.inner.WithAttrs(attrs),
	}
}

func (w *fileHandlerWrapper) WithGroup(name string) slog.Handler {
	return &fileHandlerWrapper{
		original: w.original,
		inner:    w.inner.WithGroup(name),
	}
}

type TerminalHandler struct {
	out      *os.File
	inner    slog.Handler
	opts     *slog.HandlerOptions
	colorize bool
	bufPool  *sync.Pool
}

func NewTerminalHandler(out *os.File, cfg *LogConfig) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	return &TerminalHandler{
		out:      out,
		opts:     opts,
		colorize: true,
		bufPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

func (h *TerminalHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.opts.Level.Level() <= level
}

func (h *TerminalHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := h.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer h.bufPool.Put(buf)

	handler := slog.NewTextHandler(buf, h.opts)
	if err := handler.Handle(ctx, r); err != nil {
		return err
	}

	if h.colorize {
		var color string
		switch {
		case r.Level >= slog.LevelError:
			color = colorRed
		case r.Level == slog.LevelWarn:
			color = colorYellow
		case r.Level == slog.LevelInfo:
			color = ""
		default:
			color = colorGray
		}

		_, _ = h.out.Write([]byte(color))
		_, err := h.out.Write(buf.Bytes())
		_, _ = h.out.Write([]byte(colorReset))
		return err
	}

	_, err := h.out.Write(buf.Bytes())
	return err
}

func (h *TerminalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TerminalHandler{
		out:      h.out,
		opts:     h.opts,
		colorize: h.colorize,
		bufPool:  h.bufPool,
		inner:    h.inner,
	}
}

func (h *TerminalHandler) WithGroup(name string) slog.Handler {
	return &TerminalHandler{
		out:      h.out,
		opts:     h.opts,
		colorize: h.colorize,
		bufPool:  h.bufPool,
		inner:    h.inner,
	}
}
