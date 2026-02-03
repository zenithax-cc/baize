package raid

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zenithax-cc/baize/pkg/utils"
)

func (n *nvme) collect() error {
	busPath := filepath.Join(sysfsDevicesPath, n.PCIe.PCIAddr, "nvme")
	dirs, err := os.ReadDir(busPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", busPath, err)
	}

	if len(dirs) != 1 {
		return fmt.Errorf("excepted 1 directories in %s,got %d", busPath, len(dirs))
	}

	var errs []error
	dirName := dirs[0].Name()
	n.physicalDrive.MappingFile = "/dev/" + dirName
	err = n.physicalDrive.collectSMARTData(SMARTConfig{Option: "nvme"})
	if err != nil {
		errs = append(errs, err)
	}

	namespacePath := filepath.Join(busPath, dirName)
	namespaceDirs, err := os.ReadDir(namespacePath)
	if err != nil {
		errs = append(errs, fmt.Errorf("read %s: %w", namespacePath, err))
		return utils.CombineErrors(errs)
	}

	for _, dir := range namespaceDirs {
		n.Namespaces = append(n.Namespaces, "/dev"+dir.Name())
	}

	return utils.CombineErrors(errs)
}
