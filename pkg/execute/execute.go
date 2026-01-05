package execute

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

const (
	DefaultTimeout = 20 * time.Minute
	DefaultShell   = "/bin/sh"
)

var (
	ErrEmptyCommand = errors.New("empty command")
	ErrTimeout      = errors.New("command timed out")
	ErrCanceled     = errors.New("command canceled")
	ErrNilContext   = errors.New("context cannot be nil")
)

// ExecResult 命令执行结果
type ExecResult struct {
	Stdout   []byte // 标准输出
	Stderr   []byte // 标准错误
	ExitCode int    // 退出码，-1 表示进程未正常退出
	Err      error  // 执行错误
}

// Success 判断命令是否执行成功
func (r *ExecResult) Success() bool {
	return r.Err == nil && r.ExitCode == 0
}

// Combined 返回合并的 stdout 和 stderr
func (r *ExecResult) Combined() []byte {
	return append(r.Stdout, r.Stderr...)
}

// String 返回执行结果的字符串表示
func (r *ExecResult) String() string {
	return fmt.Sprintf("ExecResult{ExitCode: %d, Stdout: %d bytes, Stderr: %d bytes, Err: %v}",
		r.ExitCode, len(r.Stdout), len(r.Stderr), r.Err)
}

// Error 实现 error 接口，方便直接使用
func (r *ExecResult) Error() string {
	if r.Err != nil {
		return fmt.Sprintf("exit code %d: %v", r.ExitCode, r.Err)
	}
	if r.ExitCode != 0 {
		return fmt.Sprintf("exit code %d: %s", r.ExitCode, string(r.Stderr))
	}
	return ""
}

// AsError 当执行失败时返回 error，成功返回 nil
func (r *ExecResult) AsError() error {
	if r.Success() {
		return nil
	}
	return r
}

// Execute 执行命令，使用默认超时时间
func Command(name string, args ...string) *ExecResult {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	return CommandWithContext(ctx, name, args...)
}

// ExecuteWithTimeout 执行命令，自定义超时时间
func CommandWithTimeout(timeout time.Duration, name string, args ...string) *ExecResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return CommandWithContext(ctx, name, args...)
}

// ExecuteShell 执行 shell 命令
func ShellCommand(cmd string) *ExecResult {
	if cmd == "" {
		return &ExecResult{ExitCode: -1, Err: ErrEmptyCommand}
	}
	return Command(DefaultShell, "-c", cmd)
}

// ExecuteShellWithTimeout 执行 shell 命令，自定义超时时间
func ExecuteShellWithTimeout(timeout time.Duration, cmd string) *ExecResult {
	if cmd == "" {
		return &ExecResult{ExitCode: -1, Err: ErrEmptyCommand}
	}
	return CommandWithTimeout(timeout, DefaultShell, "-c", cmd)
}

// ExecuteWithContext 使用 context 执行命令
func CommandWithContext(ctx context.Context, name string, args ...string) *ExecResult {
	result := &ExecResult{ExitCode: -1}

	if name == "" {
		result.Err = ErrEmptyCommand
		return result
	}

	if ctx == nil {
		result.Err = ErrNilContext
		return result
	}

	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()
	result.ExitCode = extractExitCode(cmd, err)
	result.Err = wrapError(ctx, err)

	return result
}

// extractExitCode 提取命令退出码
func extractExitCode(cmd *exec.Cmd, err error) int {
	if err == nil {
		return 0
	}

	// 尝试从 ExitError 获取退出码
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	// 进程未正常退出
	return -1
}

// wrapError 包装错误信息
func wrapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	case errors.Is(ctx.Err(), context.Canceled):
		return fmt.Errorf("%w: %v", ErrCanceled, err)
	default:
		return err
	}
}
