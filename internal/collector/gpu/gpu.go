package gpu

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenithax-cc/baize/internal/collector/pci"
	"github.com/zenithax-cc/baize/pkg/utils"
	"golang.org/x/sync/errgroup"
)

type GPU struct {
	GraphicsCard []*GraphicsCard `json:"graphics_card,omitzero"`
}

type GraphicsCard struct {
	IsOnBoard bool     `json:"is_on_board,omitzero"`
	PCIe      *pci.PCI `json:"pcie,omitzero"`
}

const (
	drmDir         = "/sys/class/drm"
	defaultCap     = 9
	maxConcurrency = 4
)

var (
	errNotFound = errors.New("GPU device not found")

	onBoardSet = map[string]struct{}{
		"102b:0522": {},
		"102b:0533": {},
		"102b:0534": {},
		"102b:0536": {},
		"102b:0538": {},
		"19e5:1711": {},
		"1a03:2000": {},
	}
)

func New() *GPU {
	return &GPU{
		GraphicsCard: make([]*GraphicsCard, 0, defaultCap),
	}
}

func (g *GPU) Collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := g.fromDrm(ctx); err == nil {
		return nil
	}

	if err := g.fromLspci(ctx); err == nil {
		return nil
	}

	return errNotFound
}

func (g *GPU) fromDrm(ctx context.Context) error {
	dirEntries, err := os.ReadDir(drmDir)
	if err != nil {
		return fmt.Errorf("read %s: %w", drmDir, err)
	}

	cardsBus := make([]string, 0, len(dirEntries)/2)
	for _, entry := range dirEntries {
		dirName := entry.Name()

		if !strings.HasPrefix(dirName, "card") {
			continue
		}

		if strings.Contains(dirName, "-") {
			continue
		}

		devicePath := filepath.Join(drmDir, dirName, "device")
		if !utils.PathExists(devicePath) {
			continue
		}

		pciBus, err := os.Readlink(devicePath)
		if err != nil {
			continue
		}

		cardsBus = append(cardsBus, pciBus)
	}

	if len(cardsBus) == 0 {
		return errNotFound
	}

	return g.collectGraphicsCards(ctx, cardsBus)
}

func (g *GPU) fromLspci(ctx context.Context) error {
	cardsBus, err := pci.GetDisplayPCIBus()
	if err != nil {
		return err
	}

	if len(cardsBus) == 0 {
		return errNotFound
	}

	return g.collectGraphicsCards(ctx, cardsBus)
}

func (g *GPU) collectGraphicsCards(ctx context.Context, cardsBus []string) error {
	n := len(cardsBus)
	if n == 0 {
		return errNotFound
	}

	concurrency := maxConcurrency
	if n < concurrency {
		concurrency = n
	}

	res := make([]*GraphicsCard, n)
	eg, egCtx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, concurrency)

	for i, bus := range cardsBus {
		idx := i
		pciBus := bus

		eg.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-egCtx.Done():
				return egCtx.Err()
			}
			defer func() { <-sem }()

			p := pci.New(pciBus)
			if err := p.Collect(); err != nil {
				return nil
			}

			card := &GraphicsCard{
				IsOnBoard: isOnBoard(p.VendorID, p.DeviceID),
				PCIe:      p,
			}

			res[idx] = card
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	var gpuCount int
	for _, card := range res {
		if card != nil {
			gpuCount++
		}
	}

	if gpuCount == 0 {
		return errNotFound
	}

	g.GraphicsCard = make([]*GraphicsCard, 0, gpuCount)
	for _, card := range res {
		if card != nil {
			g.GraphicsCard = append(g.GraphicsCard, card)
		}
	}

	return nil
}

func isOnBoard(vendorID, deviceID string) bool {
	var sb strings.Builder
	sb.Grow(9)
	sb.WriteString(vendorID)
	sb.WriteByte(':')
	sb.WriteString(deviceID)

	_, exists := onBoardSet[sb.String()]

	return exists
}
