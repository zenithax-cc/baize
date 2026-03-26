// Package main is the entry point for the baize terminal CLI tool.
// It parses command-line flags and delegates to the collector manager.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"time"

	"github.com/zenithax-cc/baize/pkg/collector"
	"github.com/zenithax-cc/baize/pkg/utils"
)

// cliCfg holds parsed command-line configuration options.
type cliCfg struct {
	module string // target module name, e.g., "cpu", "memory", "all"
	json   bool   // when true, output results as JSON
	detail bool   // when true, print detailed view instead of brief summary
}

// newCliCfg registers CLI flags and parses them, returning a populated cliCfg.
func newCliCfg() *cliCfg {
	res := &cliCfg{}
	flag.StringVar(&res.module, "m", "all", "module name")
	flag.BoolVar(&res.json, "j", false, "output json")
	flag.BoolVar(&res.detail, "d", false, "output detail")

	flag.Parse()

	return res
}

// printBanner prints the application header when in terminal (non-JSON) mode.
// func printBanner() {
// 	fmt.Printf("\n%s╔══════════════════════════════════════════════════╗%s\n", utils.ColorCyan, utils.ColorReset)
// 	fmt.Printf("%s║%s  %s白泽 (Baize) — Hardware Information Collector%s    %s║%s\n",
// 		utils.ColorCyan, utils.ColorReset, utils.ColorBold, utils.ColorReset, utils.ColorCyan, utils.ColorReset)
// 	fmt.Printf("%s╚══════════════════════════════════════════════════╝%s\n", utils.ColorCyan, utils.ColorReset)
// }

func main() {
	cfg := newCliCfg()

	// Build the collector manager with the parsed CLI settings.
	m := collector.Manager{
		Module: cfg.module,
		Detail: cfg.detail,
		Json:   cfg.json,
		Log:    slog.Default(),
	}

	start := time.Now()

	if err := collector.NewManager(&m); err != nil {
		if !cfg.json {
			fmt.Printf("\n%s⚠ collection warning: %v%s\n", utils.Yellow, err, utils.Reset)
		}
	}

	// Print elapsed time for terminal modes.
	if !cfg.json {
		elapsed := time.Since(start)
		fmt.Printf("\n%s── Collection completed in %.2fs ──%s\n\n",
			utils.Green, elapsed.Seconds(), utils.Reset)
	}
}
