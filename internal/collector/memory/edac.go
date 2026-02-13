package memory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/utils"
)

const edacPath = "/sys/devices/system/edac/mc/"

func (m *Memory) collectEdacMemory(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := os.Stat(edacPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	dimmDirs, err := filepath.Glob(filepath.Join(edacPath, "mc*", "dimm*"))
	if err != nil {
		return err
	}

	m.EdacSlots = strconv.Itoa(len(dimmDirs))
	errs := make([]error, 0, len(dimmDirs))
	for _, dimmDir := range dimmDirs {
		dimm, err := parseDimmDir(dimmDir)
		if err != nil {
			errs = append(errs, err)
		}

		m.EdacMemoryEntries = append(m.EdacMemoryEntries, dimm)
	}

	return errors.Join(errs...)
}

func parseDimmDir(dimmDir string) (*EdacMemoryEntry, error) {
	dimm := &EdacMemoryEntry{
		DIMMID: filepath.Base(dimmDir),
	}

	fields := []struct {
		name  string
		value *string
	}{
		{name: "dimm_location", value: &dimm.MemoryLocation},
		{name: "dimm_mem_type", value: &dimm.MemoryType},
		{name: "dimm_edac_mode", value: &dimm.EdacMode},
		{name: "dimm_ue_count", value: &dimm.UncorrectableErrors},
		{name: "dimm_ce_count", value: &dimm.CorrectableErrors},
		{name: "dimm_dev_type", value: &dimm.DeviceType},
		{name: "size", value: &dimm.Size},
	}

	for _, field := range fields {
		filePath := filepath.Join(dimmDir, field.name)
		if content, err := os.ReadFile(filePath); err == nil {
			utils.FillField(strings.TrimSpace(string(content)), field.value)
		}
	}

	if content, err := os.ReadFile(filepath.Join(dimmDir, "dimm_label")); err == nil {
		parseDimmLabel(dimm, strings.TrimSpace(string(content)))
	}

	return dimm, nil
}

func parseDimmLabel(dimm *EdacMemoryEntry, content string) {
	parts := strings.Split(content, "_")
	if len(parts) < 4 {
		return
	}

	for _, part := range parts {
		key, value, ok := strings.Cut(part, "#")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "SrcID":
			utils.FillField(value, &dimm.SocketID)
		case "Ha", "MC":
			utils.FillField(value, &dimm.MemoryControllerID)
		case "Chan":
			utils.FillField(value, &dimm.ChannelID)
		case "DIMM":
			utils.FillField(value, &dimm.DIMMID)
		}
	}
}
