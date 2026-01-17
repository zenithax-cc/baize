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

	return utils.ParseKeyValueFromBytes(output.Stdout, ":", map[string]*string{
		"Speed":         &nf.Speed,
		"Duplex":        &nf.Duplex,
		"Link detected": &nf.LinkDetected,
		"Port":          &nf.Port,
	})
}

func (nf *NetInterface) collectEthtoolDriver(eth string) error {
	output := execute.Command(ethtool, "-i", eth)
	if output.AsError() != nil {
		return output.Err
	}

	return utils.ParseKeyValueFromBytes(output.Stdout, ":", map[string]*string{
		"driver":           &nf.Driver,
		"version":          &nf.DriverVersion,
		"firmware-version": &nf.FirmwareVersion,
	})
}

type parseSection int

const (
	sectionPreset parseSection = iota
	sectionCurrent
)

type sectionFieldSetter struct {
	key           string
	maxSetter     *string
	currentSetter *string
}

func applySectionFields(data []byte, source []sectionFieldSetter) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	section := sectionPreset

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 {
			switch line[0] {
			case 'P':
				section = sectionPreset
				continue
			case 'C':
				section = sectionCurrent
				continue
			}

			key, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}

			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			for _, field := range source {
				if field.key == key {
					switch section {
					case sectionPreset:
						if field.maxSetter != nil {
							*field.maxSetter = value
						}
					case sectionCurrent:
						if field.currentSetter != nil {
							*field.currentSetter = value
						}
					}
				}
			}
		}
	}
}

func collectEthtoolRingBuffer(nic string) RingBuffer {
	output := execute.Command(ethtool, "-g", nic)
	if output.AsError() != nil {
		return RingBuffer{}
	}

	var res RingBuffer

	applySectionFields(output.Stdout, []sectionFieldSetter{
		{key: "RX", maxSetter: &res.MaxRX, currentSetter: &res.CurrentRX},
		{key: "TX", maxSetter: &res.MaxTX, currentSetter: &res.CurrentTX},
	})

	return res
}

func collectEthtoolChannel(nic string) Channel {
	output := execute.Command(ethtool, "-l", nic)
	if output.AsError() != nil {
		return Channel{}
	}

	var res Channel
	applySectionFields(
		output.Stdout,
		[]sectionFieldSetter{
			{key: "Rx", maxSetter: &res.MaxRX, currentSetter: &res.CurrentRX},
			{key: "Tx", maxSetter: &res.MaxTX, currentSetter: &res.CurrentTX},
			{key: "Combined", maxSetter: &res.MaxCombined, currentSetter: &res.CurrentCombined},
		},
	)

	return res
}
