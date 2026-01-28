package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/product"
)

func main() {

	p := product.New()

	err := p.Collect(context.Background())
	if err != nil {
		fmt.Println("Error collecting product information:", err)
	}

	js, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling product information:", err)
	}

	fmt.Println(string(js))
}
