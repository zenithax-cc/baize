package gpu

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/pci"
	"golang.org/x/sync/errgroup"
)

type GPU struct {
	GraphicsCards []*GraphicsCard `json:"graphics_cards,omitempty"`
}

type GraphicsCard struct {
	IsOnBoard bool      `json:"is_onboard,omitempty"`
	PCIe      *pci.PCIe `json:"pcie,omitempty"`
}

const (
	dirDrm         = "/sys/class/drm"
	defaultCap     = 8
	collectTimeout = 10 * time.Second
	maxConcurrency = 4
)

var (
	errNoGPUFound = errors.New("no GPU device found")
	onBoardMap    = map[string]bool{
		"102b:0522": true,
		"102b:0533": true,
		"102b:0534": true,
		"102b:0536": true,
		"102b:0538": true,
		"19e5:1711": true,
		"1a03:2000": true,
	}
)

func New() *GPU {
	return &GPU{
		GraphicsCards: make([]*GraphicsCard, 0, 8),
	}
}

func (g *GPU) Collect(ctx context.Context) error {

	if err := fromDrm(ctx, g); err == nil {
		return nil
	}

	if err := fromLspci(ctx, g); err != nil {
		return fmt.Errorf("failed to collect GPU information from both drm and lspci: %w", err)
	}

	if len(g.GraphicsCards) == 0 {
		return errNoGPUFound
	}

	return nil
}

func fromDrm(ctx context.Context, g *GPU) error {
	dirEntries, err := os.ReadDir(dirDrm)
	if err != nil {
		return fmt.Errorf("failed to read drm directory: %w", err)
	}

	var validCards []string

	for _, entry := range dirEntries {
		dirName := entry.Name()
		if !strings.HasPrefix(dirName, "card") || strings.ContainsRune(dirName, '-') {
			continue
		}
		devicePath := filepath.Join(dirDrm, dirName, "device")
		if !utils.PathExists(devicePath) {
			continue
		}
		pciBus, err := os.Readlink(devicePath)
		if err != nil {
			slog.Debug("failed to read DRM device link")
			continue
		}
		bus := filepath.Base(pciBus)
		validCards = append(validCards, bus)
	}

	if len(validCards) == 0 {
		return fmt.Errorf("no valid GPU found from DRM")
	}

	return collectGraphicsCards(ctx, g, validCards)

}

func fromLspci(ctx context.Context, g *GPU) error {
	buses, err := pci.GetDisplayControllerPCIBus()
	if err != nil {
		return fmt.Errorf("failed to get display controller PCI bus: %w", err)
	}

	if len(buses) == 0 {
		return fmt.Errorf("no display controller PCI bus found")
	}

	return collectGraphicsCards(ctx, g, buses)
}

func collectGraphicsCards(ctx context.Context, g *GPU, items []string) error {
	eg, egCtx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	sem := make(chan struct{}, maxConcurrency)

	for _, bus := range items {
		pciBus := bus
		eg.Go(func() error {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-egCtx.Done():
				return egCtx.Err()
			}
			p := pci.New(pciBus)
			if err := p.Collect(); err != nil {
				slog.Error("failed to collect PCI information for GPU", "pci_bus", pciBus, "error", err)
				return nil
			}
			card := &GraphicsCard{
				IsOnBoard: onBoardMap[p.VendorID+":"+p.DeviceID],
				PCIe:      p,
			}
			mu.Lock()
			g.GraphicsCards = append(g.GraphicsCards, card)
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("error during GPU collection: %w", err)
	}

	if len(g.GraphicsCards) == 0 {
		return errNoGPUFound
	}

	return nil
}
