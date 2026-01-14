package network

import "github.com/zenithax-cc/baize/pkg/utils"

func New() *Network {
	return &Network{
		PhyInterfaces:  make([]PhyInterface, 8),
		BondInterfaces: make([]BondInterface, 2),
		NetInterfaces:  make([]NetInterface, 16),
	}
}

func (n *Network) Collect() error {
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
