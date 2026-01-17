package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenithax-cc/baize/pkg/utils"
)

const procNetBonding = "/proc/net/bonding"

type bondParser struct {
	bond           *BondInterface
	inSlaveSection bool
}

func (p *bondParser) setBondMode(val string)           { p.bond.BondMode = val }
func (p *bondParser) setTransmitHashPolicy(val string) { p.bond.TransmitHashPolicy = val }
func (p *bondParser) setMIIPollingInterval(val string) { p.bond.MIIPollingInterval = val }
func (p *bondParser) setLACPRate(val string)           { p.bond.LACPRate = val }
func (p *bondParser) setAggregatorID(val string) {
	if !p.inSlaveSection {
		p.bond.AggregatorID = val
	}
}
func (p *bondParser) setNumberOfPorts(val string) { p.bond.NumberOfPorts = val }
func (p *bondParser) setSlaveInterface(val string) {
	p.inSlaveSection = true
	p.bond.SlaveInterfaces = append(p.bond.SlaveInterfaces, parseSlaveInterface(val))
}

func collectBondInterfaces() ([]BondInterface, error) {
	bonds, err := os.ReadDir(procNetBonding)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read directory %s : %w", procNetBonding, err)
	}

	res := make([]BondInterface, 0, len(bonds))
	for _, bond := range bonds {
		if !bond.Type().IsRegular() {
			continue
		}

		bondInterface, err := parseBondFile(bond.Name())
		if err != nil {
			continue
		}

		res = append(res, bondInterface)
	}

	return res, nil
}

func parseBondFile(name string) (BondInterface, error) {
	res := BondInterface{
		BondName:        name,
		SlaveInterfaces: make([]SlaveInterface, 0, 2),
	}

	file := filepath.Join(procNetBonding, name)
	content, err := utils.ReadLines(file)
	if err != nil {
		return res, err
	}

	bp := &bondParser{
		bond:           &res,
		inSlaveSection: false,
	}

	for _, line := range content {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "Bonding Mode":
			bp.setBondMode(value)
		case "Transmit Hash Policy":
			bp.setTransmitHashPolicy(value)
		case "MII Polling Interval (ms)":
			bp.setMIIPollingInterval(value)
		case "LACP Rate":
			bp.setLACPRate(value)
		case "Aggregator ID":
			bp.setAggregatorID(value)
		case "Number of ports":
			bp.setNumberOfPorts(value)
		case "Slave Interface":
			bp.setSlaveInterface(value)
		}
	}

	return res, nil
}

var slaveFieldMap = []struct {
	name   string
	setter func(*SlaveInterface, string)
}{
	{name: "link_failure_count", setter: func(s *SlaveInterface, val string) { s.LinkFailureCount = val }},
	{name: "ad_aggregator_id", setter: func(s *SlaveInterface, val string) { s.AggregatorID = val }},
	{name: "queue_id", setter: func(s *SlaveInterface, val string) { s.QueueID = val }},
	{name: "mii_status", setter: func(s *SlaveInterface, val string) { s.MIIStatus = val }},
	{name: "state", setter: func(s *SlaveInterface, val string) { s.State = val }},
}

func parseSlaveInterface(slave string) SlaveInterface {
	res := SlaveInterface{
		SlaveName: slave,
	}

	dirPath := filepath.Join(sysfsNet, slave, "bonding_slave")

	for _, field := range slaveFieldMap {
		filePath := filepath.Join(dirPath, field.name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		field.setter(&res, strings.TrimSpace(string(content)))
	}

	return res
}
