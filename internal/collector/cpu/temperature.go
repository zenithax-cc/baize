package cpu

import (
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const avtcmd = "/usr/local/beidou/tool/AVT/AVTCMD"

func collectAMDTemperature() (int, error) {
	output := execute.Command(avtcmd, `-module thermal "getDieTemp()" | awk '{print $4}'`)
	if output.Err != nil {
		return 0, output.Err
	}

	return strconv.Atoi(strings.TrimSpace(string(output.Stdout)))
}


