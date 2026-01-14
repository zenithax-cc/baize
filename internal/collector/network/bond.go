package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenithax-cc/baize/pkg/utils"
)

const procNetBonding = "/proc/net/bonding"

func collectBondInterfaces() ([]BondInterface, error) {
	bonds, err := os.ReadDir(procNetBonding)
	if err != nil {
		return nil, fmt.Errorf("read directory %s : %w", procNetBonding, err)
	}

	res := make([]BondInterface, 0, len(bonds))
	for _, bond := range bonds {
		name := bond.Name()

		res = append(res, parseBondFile(name))
	}

	return res, nil
}

type bondFunc func(b *BondInterface, val string)

var slaveFlag bool
var fieldMap = map[string]bondFunc{
	"Bonding Mode":              func(b *BondInterface, val string) { b.BondMode = val },
	"Transmit Hash Policy":      func(b *BondInterface, val string) { b.TransmitHashPolicy = val },
	"MII Polling Interval (ms)": func(b *BondInterface, val string) { b.MIIPollingInterval = val },
	"LACP Rate":                 func(b *BondInterface, val string) { b.LACPRate = val },
	"Aggregator ID": func(b *BondInterface, val string) {
		if !slaveFlag {
			b.AggregatorID = val
		}
	},
	"Number of ports": func(b *BondInterface, val string) { b.NumberOfPorts = val },
	"Slave Interface": func(b *BondInterface, val string) {
		b.SlaveInterfaces = append(b.SlaveInterfaces, parseSlaveInterface(val))
	},
}

func parseBondFile(name string) BondInterface {
	res := BondInterface{
		BondName: name,
	}

	file := filepath.Join(procNetBonding, name)
	content, err := utils.ReadLines(file)
	if err != nil {
		return res
	}

	for _, line := range content {
		if strings.TrimSpace(line) == "" {
			continue
		}

		key, value, ok := utils.ParseLineKeyValue(line, ":")
		if !ok {
			continue
		}

		if f, exists := fieldMap[key]; exists {
			f(&res, value)
		}
	}

	return res
}

type slaveFunc func(s *SlaveInterface, val string)

var slaveFieldMap = map[string]slaveFunc{
	"link_failure_count": func(s *SlaveInterface, val string) { s.LinkFailureCount = val },
	"ad_aggregator_id":   func(s *SlaveInterface, val string) { s.AggregatorID = val },
	"queue_id":           func(s *SlaveInterface, val string) { s.QueueID = val },
	"mii_status":         func(s *SlaveInterface, val string) { s.MIIStatus = val },
	"state":              func(s *SlaveInterface, val string) { s.State = val },
}

func parseSlaveInterface(slave string) SlaveInterface {
	res := SlaveInterface{
		SlaveName: slave,
	}
	dirPath := filepath.Join(sysfsNet, slave, "bonding_slave")
	entries, err := os.ReadDir(dirPath)
	if err != err {
		return res
	}

	for _, entry := range entries {
		name := entry.Name()
		if f, exists := slaveFieldMap[name]; exists {
			file := filepath.Join(dirPath, name)
			val, err := utils.ReadOneLineFile(file)
			if err != nil {
				continue
			}
			f(&res, val)
		}
	}
	return res
}
