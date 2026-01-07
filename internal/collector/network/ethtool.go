package network

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
	"github.com/zenithax-cc/baize/pkg/utils"
)

const ethtool = "/usr/sbin/ethtool"

func (nf *NetInterface) collectEthtoolSetting(eth string) error {
	output := execute.Command(ethtool, eth)
	if output.AsError() != nil {
		return output.Err
	}

	ethMap := utils.ParseKeyValue(string(output.Stdout), ":")
	fiedlMap := map[string]*string{
		"Speed":         &nf.Speed,
		"Duplex":        &nf.Duplex,
		"Link detected": &nf.LinkDetected,
		"Port":          &nf.Port,
	}

	for k, v := range fiedlMap {
		if *v != "" {
			continue
		}

		if val, ok := ethMap[k]; ok {
			*v = val
		}
	}

	return nil
}

func (nf *NetInterface) collectEthtoolDriver(eth string) error {
	output := execute.Command(ethtool, "-i", eth)
	if output.AsError() != nil {
		return output.Err
	}

	ethMap := utils.ParseKeyValue(string(output.Stdout), ":")
	fiedlMap := map[string]*string{
		"driver":           &nf.Driver,
		"version":          &nf.DriverVersion,
		"firmware-version": &nf.FirmwareVersion,
	}

	for k, v := range fiedlMap {
		if *v != "" {
			continue
		}

		if val, ok := ethMap[k]; ok {
			*v = val
		}
	}

	return nil
}

func collectEthtoolRingBuffer(nic string) RingBuffer {
	output := execute.Command(ethtool, "-g", nic)
	if output.AsError() != nil {
		return RingBuffer{}
	}

	scanner := bufio.NewScanner(bytes.NewReader(output.Stdout))
	res := RingBuffer{}
	flag := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Current") {
			flag = true
			continue
		}

		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			if flag {
				switch key {
				case "RX":
					res.CurrentRX = value
				case "TX":
					res.CurrentTX = value
				}
			}

			switch key {
			case "RX":
				res.MaxRX = value
			case "TX":
				res.MaxTX = value
			}
		}

	}

	return res
}

func collectEthtoolChannel(nic string) Channel {
	output := execute.Command(ethtool, "-l", nic)
	if output.AsError() != nil {
		return Channel{}
	}

	scanner := bufio.NewScanner(bytes.NewReader(output.Stdout))
	res := Channel{}
	flag := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Current") {
			flag = true
			continue
		}

		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			if flag {
				switch key {
				case "RX":
					res.CurrentRX = value
				case "TX":
					res.CurrentTX = value
				case "Combined":
					res.CurrentCombined = value
				}
			}
			switch key {
			case "RX":
				res.MaxRX = value
			case "TX":
				res.MaxTX = value
			case "Combined":
				res.MaxCombined = value
			}
		}
	}

	return res
}
