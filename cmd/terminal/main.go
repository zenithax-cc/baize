package main

import (
	"context"
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/network"
)

// import (
// 	"flag"
// 	"fmt"
// 	"os"

// 	"github.com/zenithax-cc/baize/pkg/collector"
// )

// type cliCfg struct {
// 	module string
// 	json   bool
// 	detail bool
// }

// func newCliCfg() *cliCfg {
// 	res := &cliCfg{}
// 	flag.StringVar(&res.module, "m", "all", "module name")
// 	flag.BoolVar(&res.json, "j", false, "output json")
// 	flag.BoolVar(&res.detail, "d", false, "output detail")

// 	flag.Parse()

// 	return res
// }

// var supportedModules = map[string]struct{}{
// 	"all":     {},
// 	"product": {},
// 	"cpu":     {},
// 	"memory":  {},
// 	"raid":    {},
// 	"network": {},
// 	"bond":    {},
// 	"gpu":     {},
// 	"health":  {},
// }

// func isSurpportModule(module string) bool {
// 	_, exists := supportedModules[module]
// 	return exists
// }

// func runCollection() error {
// 	cfg := newCliCfg()

// 	if !isSurpportModule(cfg.module) {
// 		flag.Usage()
// 		return fmt.Errorf("module %s not surpport", cfg.module)
// 	}

// 	err := collector.NewManager()

// 	return err
// }

// func main() {
// 	if err := runCollection(); err != nil {
// 		fmt.Printf("collection error: %v", err)
// 		os.Exit(1)
// 	}
// }

func main() {
	c := network.New()

	if err := c.Collect(context.Background()); err != nil {
		fmt.Printf("collection error: %v", err)
	}

	c.JSON()
}
