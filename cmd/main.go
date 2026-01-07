package main

import (
	"fmt"
	"net"
)

func main() {
	nf, err := net.Interfaces()
	if err != nil {
		fmt.Println(err)
	}

	for _, n := range nf {
		ipNet, err := n.Addrs()
		if err != nil {
			fmt.Println(err)
		}

		for _, addr := range ipNet {
			fmt.Println(addr.String())
		}
	}

}
