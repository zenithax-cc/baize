package pkg

import "github.com/zenithax-cc/baize/internal/collector/product"

type Product struct {
	*product.Product
}

func NewProduct() *Product {
	return &Product{
		Product: product.New(),
	}
}

func (p *Product) PrintJson() {
	printJson("Product", p.Product)
}

func (p *Product) PrintBrief() {
	println("[PRODUCT INFO]")
}

func (p *Product) PrintDetail() {}
