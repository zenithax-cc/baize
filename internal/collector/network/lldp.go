package network

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const lldpctl = "/usr/sbin/lldpctl"

const (
	fieldChassisMac      = "chassis.mac"
	fieldChassisName     = "chassis.name"
	fieldChassisMgmtIP   = "chassis.mgmt-ip"
	fieldPortIfname      = "port.ifname"
	fieldPortAggregation = "port.aggregation"
	fieldVlanID          = "vlan.vlan-id"
	fieldVlanPvid        = "vlan.pvid"
	fieldPpvidSupport    = "ppvid.support"
	fieldPpvidEnabled    = "ppvid.enabled"
)

func lldpNeighbors(nic string) (LLDP, error) {
	output := execute.Command(lldpctl, nic, "-f", "keyvalue")
	if output.AsError() != nil {
		return LLDP{}, output.Err
	}

	var prefixBuilder strings.Builder
	prefixBuilder.Grow(7 + len(nic))
	prefixBuilder.WriteString("lldp.")
	prefixBuilder.WriteString(nic)
	prefixBuilder.WriteByte('.')
	prefix := prefixBuilder.String()
	prefixLen := len(prefix)

	scanner := bufio.NewScanner(bytes.NewReader(output.Stdout))
	var res LLDP

	for scanner.Scan() {
		line := scanner.Text()
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			res.ToRDesc = line
			continue
		}

		if len(key) > prefixLen && key[:prefixLen] == prefix {
			key = key[prefixLen:]
		}

		setLLDPField(&res, strings.TrimSpace(key), strings.TrimSpace(value))

	}

	if err := scanner.Err(); err != nil {
		return res, err
	}

	return res, nil
}

func setLLDPField(l *LLDP, key, value string) {
	switch key {
	case fieldChassisMac:
		l.ToRAddress = value
	case fieldChassisName:
		l.ToRName = value
	case fieldChassisMgmtIP:
		l.ManagementIP = value
	case fieldPortIfname:
		l.Interface = value
	case fieldPortAggregation:
		l.PortAggregation = value
	case fieldVlanID:
		l.VLAN = value
	case fieldVlanPvid:
		l.PPVID = value
	case fieldPpvidSupport:
		l.PPVIDSupport = value
	case fieldPpvidEnabled:
		l.PPVIDEnabled = value
	}
}
