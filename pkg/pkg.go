package pkg

import (
	"encoding/json"
	"fmt"
)

func printJson(mode string, c any) {
	if c == nil {
		fmt.Printf("%s information is not collected yet\n", mode)
		return
	}
	js, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		fmt.Printf("Failed to marshal %s information to JSON: %v\n", mode, err)
		return
	}

	fmt.Println(string(js))
}
