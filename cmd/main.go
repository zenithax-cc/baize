package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/cpu"
)

func main() {
	n := cpu.New()

	err := n.Collect(context.Background())
	if err != nil {
		fmt.Printf("cpu: %v", err)
	}

	js, err := json.MarshalIndent(n, "", " ")
	if err != nil {
		fmt.Printf("marshl json: %v", err)
	}

	fmt.Println(string(js))
}
