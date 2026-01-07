package network

import (
	"bytes"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const lldpctl = "/usr/sbin/lldpctl"

func lldpNeighbors(nic string) LLDP {
	output := execute.Command(lldpctl, nic, "-f", "keyvalue")
	if output.AsError() != nil {
		return LLDP{}
	}

	prefix := "lldp." + nic + "."
	scanner := bufio.Newscanner(bytes.NewReader(output.Stdout))
	res := LLDP{}
	for scanner.Scan() {
		line := scanner.Text()
	}
}
