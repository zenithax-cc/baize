package network

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const lldpctl = "/usr/sbin/lldpctl"

var lldpFields = map[string]func(*LLDP, string){
	"chassis.mac":      func(l *LLDP, v string) { l.ToRAddress = v },
	"chassis.name":     func(l *LLDP, v string) { l.ToRName = v },
	"chassis.mgmt-ip":  func(l *LLDP, v string) { l.ManagementIP = v },
	"port.ifname":      func(l *LLDP, v string) { l.Interface = v },
	"port.aggregation": func(l *LLDP, v string) { l.PortAggregation = v },
	"vlan.vlan-id":     func(l *LLDP, v string) { l.VLAN = v },
	"vlan.pvid":        func(l *LLDP, v string) { l.PPVID = v },
	"ppvid.support":    func(l *LLDP, v string) { l.PPVIDSupport = v },
	"ppvid.enabled":    func(l *LLDP, v string) { l.PPVIDEnabled = v },
}

func lldpNeighbors(nic string) (LLDP, error) {
	output := execute.Command(lldpctl, nic, "-f", "keyvalue")
	if output.AsError() != nil {
		return LLDP{}, output.Err
	}

	prefix := "lldp." + nic + "."
	scanner := bufio.NewScanner(bytes.NewReader(output.Stdout))
	res := LLDP{}
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "=") {
			res.ToRDesc = line
			continue
		}

		line = strings.TrimPrefix(line, prefix)
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			if f, ok := lldpFields[key]; ok {
				f(&res, value)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return res, err
	}

	return res, nil
}
