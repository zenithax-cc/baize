package network

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
	"golang.org/x/sync/errgroup"
)

const (
	procBonding = "/proc/net/bonding"
)

func (n *Network) collectBondInterfaces(ctx context.Context) error {
	dirEntries, err := utils.ReadDir(procBonding)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, 2)
	physicalMap := n.buildPhysicalMap()

	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		port := entry.Name()
		eg.Go(func() error {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return ctx.Err()
			}
			return n.proccessBondInterface(ctx, port, physicalMap)
		})
	}

	return eg.Wait()
}

func (n *Network) proccessBondInterface(ctx context.Context, port string, physicalMap map[string]physicalInterface) error {

	bond := bondInterface{
		SlaveInterfaces: make([]physicalInterface, 0, 2),
	}

	if netIntf, err := n.getBaiscInfo(ctx, port); err != nil {
		bond.Name = port
		return err
	} else {
		bond.networkInterface = netIntf
	}

	filePath := filepath.Join(procBonding, port)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	isSlave := false
	fieldMap := map[string]func(val string){
		"Bonding Mode":              func(val string) { bond.Mode = val },
		"Transmit Hash Policy":      func(val string) { bond.HashPolicy = val },
		"MII Polling Interval (ms)": func(val string) { bond.PollingInterval = val },
		"LACP Rate":                 func(val string) { bond.LACPRate = val },
		"LACP Active":               func(val string) { bond.LACPActive = val },
		"Aggregator ID": func(val string) {
			if !isSlave {
				bond.AggregatorID = val
			}
		},
		"Number of ports": func(val string) { bond.NumberOfPorts = val },
		"Slave Interface": func(val string) {
			if slave, ok := physicalMap[val]; ok {
				bond.SlaveInterfaces = append(bond.SlaveInterfaces, slave)
			}
		},
	}
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.ContainsAny(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "Slave Interface") {
			isSlave = true
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if fn, ok := fieldMap[key]; ok {
			fn(value)
		}
	}

	bond.diagnose()

	n.mutex.Lock()
	n.BondInterfaces = append(n.BondInterfaces, bond)
	n.mutex.Unlock()
	return nil
}

func (b *bondInterface) diagnose() {
	b.Diagnose = "Healthy"
	var sb strings.Builder

	if len(b.SlaveInterfaces) != 2 {
		sb.WriteString("the amount of slave interfaces should be 2; ")
	}

	if b.Status == "down" {
		sb.WriteString("the bond interface is down; ")
	}

	for _, slave := range b.SlaveInterfaces {
		if b.AggregatorID != slave.AggregatorID {
			sb.WriteString("the aggregator id of slave interfaces is not the same; ")
		}
	}

	if sb.Len() > 0 {
		b.Diagnose = "Unhealthy"
		b.DiagnoseDetail = sb.String()
	}
}
