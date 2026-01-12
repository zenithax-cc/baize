package main

import (
	"encoding/json"
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/network"
)

func main() {
	n, err := network.CollectNetInterfaces()
	if err != nil {
		fmt.Printf("network: %v", err)
	}

	js, err := json.MarshalIndent(n, "", " ")
	if err != nil {
		fmt.Printf("marshl json: %v", err)
	}

	fmt.Println(string(js))
}
