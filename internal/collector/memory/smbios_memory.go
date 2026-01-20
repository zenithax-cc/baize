package memory

import (
	"context"
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

func collectPhysicalMemory(ctx context.Context) ([]*SmbiosMemory, error) {
	d, err := smbios.New(ctx)
	if err != nil {
		return nil, err
	}

	memoryTables, err := smbios.GetTypeData[*smbios.Type17MemoryDevice](d, 17)
	if err != nil {
		return nil, err
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

	res := make([]*SmbiosMemory, 0, len(memoryTables))
	for _, t := range memoryTables {
		speed := speedStr(t.Speed)
		if speed == "Unknown" {
			continue
		}

		mem := &SmbiosMemory{
			BaseMemoryInfo: BaseMemoryInfo{
				Size:         t.GetSizeString(),
				SerialNumber: t.SerialNumber,
				Manufacturer: t.Manufacturer,
			},
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
		}

		res = append(res, mem)
	}

	return res, nil
}
