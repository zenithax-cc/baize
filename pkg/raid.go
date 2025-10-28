package pkg

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/zenithax-cc/baize/common/color"
	"github.com/zenithax-cc/baize/internal/collector/raid"
)

type RAID struct {
	SystemDisk string `json:"system_disk,omitempty"`
	*raid.Controllers
}

func NewRAID() *RAID {
	return &RAID{
		Controllers: raid.New(),
	}
}

func (r *RAID) PrintJSON() {
	printJson("RAID", r.Controllers)
}

func (r *RAID) PrintBrief() {
	var sb strings.Builder
	sb.Grow(r.estimatePrintSize())
	sb.WriteString("[RAID INFO]\n")

	if err := r.getSystemDiskRAID(); err != nil {
		fmt.Fprintf(&sb, "get system disk raid error: %v\n", err)
	} else {
		colorRaid := color.Green(r.SystemDisk)
		if r.SystemDisk != "RAID1" {
			colorRaid = color.Red(r.SystemDisk)
		}

		sb.WriteString(printSeparator("SystemDiskRAID", colorRaid, true, 1))
	}

	ctrFields := []string{"ProductName", "ID", "ControllerStatus"}
	ldFields := []string{"LogicalDrives", "Location", "State", "Capacity"}

	type otherPD struct {
		jbod  map[string]int
		ugood map[string]int
		other map[string]int
		nvme  map[string]int
	}

	otherPDs := otherPD{
		jbod:  map[string]int{},
		ugood: map[string]int{},
		other: map[string]int{},
		nvme:  map[string]int{},
	}

	for _, ctr := range r.Controllers.Controller {
		sb.WriteString(selectFields(ctr, ctrFields, 1, nil).String())

		for _, vd := range ctr.LogicalDrives {
			sb.WriteString("\n" + selectFields(vd, ldFields, 2, nil).String())
			pdMap := map[string]int{}
			for _, pd := range vd.PhysicalDrives {
				pdDetail := fmt.Sprintf("%s %s %s %s %s", pd.Vendor, pd.Capacity, pd.FormFactor, pd.ProtocolType, pd.RotationRate)
				pdMap[pdDetail]++
			}
			for k, v := range pdMap {
				sb.WriteString(printSeparator(k, fmt.Sprintf(" * %d", v), false, 3))
			}
		}

		for _, pd := range ctr.PhysicalDrives {
			if pd.State == "Onln" || pd.State == "OK" {
				continue
			}
			pdDetail := fmt.Sprintf("%s %s %s %s %s", pd.Vendor, pd.Capacity, pd.FormFactor, pd.ProtocolType, pd.RotationRate)
			if pd.State == "JBOD" {
				otherPDs.jbod[pdDetail]++
			} else if pd.State == "UGood" {
				otherPDs.ugood[pdDetail]++
			} else {
				otherPDs.other[pdDetail]++
			}
		}
	}

	if len(r.Controllers.NVMe) > 0 {
		for _, nvme := range r.Controllers.NVMe {
			nvmeDetail := fmt.Sprintf("%s %s %s %s", nvme.Vendor, nvme.Capacity, nvme.ProtocolType, nvme.RotationRate)
			otherPDs.nvme[nvmeDetail]++
		}
	}
	if len(otherPDs.jbod) > 0 {
		sb.WriteString("\n        JBOD Drives:\n")
		for k, v := range otherPDs.jbod {
			sb.WriteString(fmt.Sprintf("	%s * %d\n", k, v))
		}
	}

	if len(otherPDs.ugood) > 0 {
		sb.WriteString("\n        UGood Drives:\n")
		for k, v := range otherPDs.ugood {
			sb.WriteString(fmt.Sprintf("	%s * %d\n", k, v))
		}
	}

	if len(otherPDs.other) > 0 {
		sb.WriteString("\n        Other Drives:\n")
		for k, v := range otherPDs.other {
			sb.WriteString(fmt.Sprintf("%s * %d\n", k, v))
		}
	}

	if len(otherPDs.nvme) > 0 {
		sb.WriteString("\n        NVMe Drives:\n")
		for k, v := range otherPDs.nvme {
			sb.WriteString(fmt.Sprintf("	%s * %d\n", k, v))
		}
	}

	println(sb.String())
}

func (r *RAID) PrintDetail() {}

func (r *RAID) Name() string {
	return "RAID"
}

func (r *RAID) getSystemDiskRAID() error {
	systemDisk, err := findSystemDisk()
	if err != nil {
		return err
	}

	for _, controller := range r.Controllers.Controller {
		for _, vd := range controller.LogicalDrives {
			if vd.MappingFile == systemDisk {
				r.SystemDisk = vd.Type
				return nil
			}
		}

		for _, pd := range controller.PhysicalDrives {
			if pd.MappingFile == systemDisk {
				r.SystemDisk = pd.State
				return nil
			}
		}
	}

	return errors.New("not found system disk raid")
}

const procMount = "/proc/mounts"

func findSystemDisk() (string, error) {
	f, err := os.Open(procMount)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]

		if mountPoint == "/" {
			return extractMainDevice(device), nil
		}
	}

	return "", errors.New("not found system disk")
}

func extractMainDevice(device string) string {
	name := strings.TrimPrefix(device, "/dev/")

	pattens := []*regexp.Regexp{
		regexp.MustCompile(`^(nvme\d+n\d+)p\d+$`), // nvme
		regexp.MustCompile(`^(mmcblk\d+)p\d+$`),
		regexp.MustCompile(`^(vd[a-z]+)\d+$`),
		regexp.MustCompile(`^(xvd[a-z]+)\d+$`),
		regexp.MustCompile(`^([a-z]+)\d+$`),
	}

	handler := func(pattern *regexp.Regexp, s string) string {
		matchs := pattern.FindStringSubmatch(s)
		if len(matchs) > 1 {
			return "/dev/" + matchs[1]
		}
		return "/dev/" + s
	}

	for _, pattern := range pattens {
		if pattern.MatchString(name) {
			return handler(pattern, name)
		}
	}

	return "/dev/" + name
}

func (r *RAID) estimatePrintSize() int {
	baseSize := 100
	ctrlCount := len(r.Controllers.Controller)

	estimateSize := baseSize + (ctrlCount * 100)
	for _, ctrl := range r.Controllers.Controller {
		estimateSize += len(ctrl.LogicalDrives) * 50
	}

	return estimateSize
}
