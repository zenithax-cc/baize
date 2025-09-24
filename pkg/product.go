package pkg

import (
	"strings"

	"github.com/zenithax-cc/baize/internal/collector/product"
)

type Product struct {
	*product.Product
}

func NewProduct() *Product {
	return &Product{
		Product: product.New(),
	}
}

func (p *Product) PrintJSON() {
	printJson("Product", p.Product)
}

func (p *Product) PrintBrief() {
	var sb strings.Builder
	sb.Grow(1000)
	sb.WriteString("[PRODUCT INFO]\n")

	chassisFields := []string{"AssetTag", "Type", "SN", "Height"}
	systemFields := []string{"ProductName", "Manufacturer", "UUID"}
	osFields := []string{"Hostname", "KernelName", "MinorVersion", "KernelRelease"}

	sb.WriteString(selectFields(p.Product.System, systemFields, 1, nil).String())
	sb.WriteString(selectFields(p.Product.Chassis, chassisFields, 1, nil).String())
	sb.WriteString(selectFields(p.Product.OS, osFields, 1, nil).String())

	println(sb.String())
}

func (p *Product) PrintDetail() {}

func (p *Product) Name() string {
	return "Product"
}
