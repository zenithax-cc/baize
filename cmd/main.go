package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/memory"
)

func main() {
	n := memory.New()

	err := n.Collect(context.Background())
	if err != nil {
		fmt.Printf("network: %v", err)
	}

	js, err := json.MarshalIndent(n, "", " ")
	if err != nil {
		fmt.Printf("marshl json: %v", err)
	}

	fmt.Println(string(js))
}
