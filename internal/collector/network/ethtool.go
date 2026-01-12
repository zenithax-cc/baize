package network

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
	"github.com/zenithax-cc/baize/pkg/utils"
)

const ethtool = "/usr/sbin/ethtool"

func applyFieldMapping(source map[string]string, target map[string]*string) {
	for key, ptr := range target {
		if *ptr == "" {
			if val, ok := source[key]; ok {
				*ptr = val
			}
		}
	}
}

func (nf *NetInterface) collectEthtoolSetting(eth string) error {
	output := execute.Command(ethtool, eth)
	if output.AsError() != nil {
		return output.Err
	}

	applyFieldMapping(utils.ParseKeyValue(string(output.Stdout), ":"),
		map[string]*string{
			"Speed":         &nf.Speed,
			"Duplex":        &nf.Duplex,
			"Link detected": &nf.LinkDetected,
			"Port":          &nf.Port,
		})

	return nil
}

func (nf *NetInterface) collectEthtoolDriver(eth string) error {
	output := execute.Command(ethtool, "-i", eth)
	if output.AsError() != nil {
		return output.Err
	}

	applyFieldMapping(utils.ParseKeyValue(string(output.Stdout), ":"),
		map[string]*string{
			"driver":           &nf.Driver,
			"version":          &nf.DriverVersion,
			"firmware-version": &nf.FirmwareVersion,
		})

	return nil
}

type parseSection int

const (
	sectionPreset parseSection = iota
	sectionCurrent
)

type sectionFieldSetter struct {
	maxSetter     func(value string)
	currentSetter func(value string)
}

func applySectionFieldMapping(data []byte, source map[string]sectionFieldSetter) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	section := sectionPreset
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "Pre-set maximums"):
			section = sectionPreset
			continue
		case strings.HasPrefix(line, "Current hardware settings"):
			section = sectionCurrent
			continue
		}

		key, value, ok := utils.ParseLineKeyValue(line, ":")
		if !ok {
			continue
		}

		if setter, ok := source[key]; ok {
			switch section {
			case sectionPreset:
				setter.maxSetter(value)
			case sectionCurrent:
				setter.currentSetter(value)
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

	applySectionFieldMapping(output.Stdout, map[string]sectionFieldSetter{
		"RX": {
			maxSetter:     func(value string) { res.MaxRX = value },
			currentSetter: func(value string) { res.CurrentRX = value },
		},
		"TX": {
			maxSetter:     func(value string) { res.MaxTX = value },
			currentSetter: func(value string) { res.CurrentTX = value },
		},
	})

	return res
}

func collectEthtoolChannel(nic string) Channel {
	output := execute.Command(ethtool, "-l", nic)
	if output.AsError() != nil {
		return Channel{}
	}

	var res Channel
	applySectionFieldMapping(
		output.Stdout,
		map[string]sectionFieldSetter{
			"Rx": {
				maxSetter:     func(v string) { res.MaxRX = v },
				currentSetter: func(v string) { res.CurrentRX = v },
			},
			"Tx": {
				maxSetter:     func(v string) { res.MaxTX = v },
				currentSetter: func(v string) { res.CurrentTX = v },
			},
			"Combined": {
				maxSetter:     func(v string) { res.MaxCombined = v },
				currentSetter: func(v string) { res.CurrentCombined = v },
			},
		},
	)

	return res
}
