package main

import (
	"context"
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/health"
	"github.com/zenithax-cc/baize/pkg/utils"
)

func main() {

	p := health.New()

	err := p.Collect(context.Background())
	if err != nil {
		fmt.Println("Error collecting product information:", err)
	}

	newPrint := utils.NewStructPrinter()
	newPrint.Print(p)
	// js, err := json.MarshalIndent(p, "", "  ")
	// if err != nil {
	// 	fmt.Println("Error marshalling product information:", err)
	// }

	// fmt.Println(string(js))
}
