package memory

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/internal/collector/smbios"
	"github.com/zenithax-cc/baize/pkg/utils"
)

func (m *Memory) collectFromSMBIOS(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	memoryTables, err := smbios.GetTypeData[*smbios.Type17MemoryDevice](17)
	if err != nil {
		return err
	}

	bitWidthStr := func(v uint16) string {
		if v == 0 || v == 0xFFFF {
			return "Unknown"
		}
		return fmt.Sprintf("%d bits", v)
	}

	speedStr := func(v uint16) string {
		if v == 0 || v == 0xFFFF {
			return "Unknown"
		}
		return fmt.Sprintf("%d MT/s", v)
	}

	voltageStr := func(v uint16) string {
		switch {
		case v == 0:
			return "Unknown"
		case v%100 == 0:
			return fmt.Sprintf("%.1f V", float32(v)/1000.0)
		default:
			return fmt.Sprintf("%g V", float32(v)/1000.0)
		}
	}

	m.Maxslots = strconv.Itoa(len(memoryTables))
	var totalSize int

	for _, t := range memoryTables {
		speed := speedStr(t.Speed)
		if speed == "Unknown" {
			continue
		}

		m.PhysicalMemoryEntries = append(m.PhysicalMemoryEntries, &SmbiosMemoryEntry{
			Size:              t.GetSizeString(),
			SerialNumber:      t.SerialNumber,
			Manufacturer:      t.Manufacturer,
			TotalWidth:        bitWidthStr(t.TotalWidth),
			DataWidth:         bitWidthStr(t.DataWidth),
			FormFactor:        t.FormFactor.String(),
			DeviceLocator:     t.DeviceLocator,
			BankLocator:       t.BankLocator,
			Type:              t.Type.String(),
			TypeDetail:        t.TypeDetail.String(),
			Speed:             speed,
			PartNumber:        t.PartNumber,
			Rank:              t.GetRankString(),
			ConfiguredSpeed:   speedStr(t.ConfiguredSpeed),
			ConfiguredVoltage: voltageStr(t.ConfiguredVoltage),
			Technology:        t.Technology.String(),
		})

		if size, err := toBytes(t.GetSizeString()); err == nil {
			totalSize += size
		}
	}

	m.PhysicalMemorySize = utils.KGMT(float64(totalSize), true)

	return nil
}

func toBytes(s string) (int, error) {
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid size string: %s", s)
	}

	unit := strings.ToLower(parts[1])
	res, err := strconv.Atoi(parts[0])

	switch unit {
	case "b":
		return res, err
	case "kb":
		return res * 1024, err
	case "mb":
		return res * 1024 * 1024, err
	case "gb":
		return res * 1024 * 1024 * 1024, err
	case "tb":
		return res * 1024 * 1024 * 1024 * 1024, err
	}

	return res, err
}
