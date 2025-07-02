package main

import (
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/pkg"
)

type componentGetter interface {
	Collect() error
	PrintJson()
	PrintBrief()
	PrintDetail()
}

type outputType int

const (
	Json outputType = iota
	Brief
	Detail
)

type componentType string

const (
	All     componentType = "all"
	Product componentType = "product"
	CPU     componentType = "cpu"
	Memory  componentType = "memory"
	RAID    componentType = "raid"
	Network componentType = "network"
	Bond    componentType = "bond"
	GPU     componentType = "gpu"
	Health  componentType = "health"
)

var validComponents = []componentType{All, Product, CPU, Memory, RAID, Network, Bond, GPU, Health}

func componentGetterFactory(c componentType) (componentGetter, error) {
	switch c {
	case Product:
		return pkg.NewProduct(), nil
	case CPU:
		return pkg.NewCPU(), nil
	case Memory:
		return pkg.NewMemory(), nil
	case RAID:
		return pkg.NewRAID(), nil
	case GPU:
		return pkg.NewGPU(), nil
	case Network:
		return pkg.NewNetwork(), nil
	case Bond:
		return pkg.NewBond(), nil
	case Health:
		return pkg.NewHealth(), nil
	default:
		return nil, fmt.Errorf("invalid component type: %s", c)
	}
}

func getAllComponents() []string {
	res := make([]string, len(validComponents))
	for i, v := range validComponents {
		res[i] = string(v)
	}
	return res
}

func getComponentInstance(c componentType) ([]componentGetter, error) {
	if c == All {
		var res []componentGetter
		var errs []error
		for _, v := range validComponents[1:] {
			g, err := componentGetterFactory(v)
			if err != nil {
				errs = append(errs, err)
			}
			res = append(res, g)
		}
		return res, utils.CombineErrors(errs)
	}

	g, err := componentGetterFactory(c)
	if err != nil {
		return nil, err
	}
	return []componentGetter{g}, nil
}

func processComponent(cgs []componentGetter, ot outputType) {
	for _, cg := range cgs {
		if err := cg.Collect(); err != nil {
			fmt.Printf("Failed to collect %s information: %v \n", cg, err)
			continue
		}

		switch ot {
		case Json:
			cg.PrintJson()
		case Brief:
			cg.PrintBrief()
		case Detail:
			cg.PrintDetail()
		}
	}
}

func main() {
	mode := flag.String("m", "all", fmt.Sprintf("Query mode infomation,surpported value: %v", getAllComponents()))
	detail := flag.Bool("d", false, "Show detail information.")
	js := flag.Bool("j", false, "Output json format.")
	flag.Parse()
	component := componentType(*mode)
	if !slices.Contains(validComponents, component) {
		fmt.Fprintf(os.Stderr, "Unsupported mode:%s\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
	cgs, err := getComponentInstance(component)
	if err != nil || len(cgs) == 0 {
		fmt.Fprintf(os.Stderr, "Failed to get component instance: %v\n", err)
		os.Exit(1)
	}
	var ot outputType
	switch {
	case *js:
		ot = Json
	case *detail:
		ot = Detail
	default:
		ot = Brief
	}
	processComponent(cgs, ot)
}
