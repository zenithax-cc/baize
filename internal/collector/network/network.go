package network

import (
	"context"

	"github.com/zenithax-cc/baize/pkg/utils"
)

func New() *Network {
	return &Network{
		PhyInterfaces:  make([]PhyInterface, 0, 8),
		BondInterfaces: make([]BondInterface, 0, 2),
		NetInterfaces:  make([]NetInterface, 0, 16),
	}
}

func (n *Network) Collect(ctx context.Context) error {
	var errs []error
	phys, err := collectNic()
	if err != nil {
		errs = append(errs, err)
	}
	n.PhyInterfaces = phys

	nets, err := CollectNetInterfaces()
	if err != nil {
		errs = append(errs, err)
	}
	n.NetInterfaces = nets

	bonds, err := collectBondInterfaces()
	if err != nil {
		errs = append(errs, err)
	}
	n.BondInterfaces = bonds

	return utils.CombineErrors(errs)
}

func (n *Network) JSON() error {
	return utils.JSONPrintln(n)
}
