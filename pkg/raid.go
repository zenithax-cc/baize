package pkg

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
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

func (r *RAID) PrintJson() {
	printJson("RAID", r.Controllers)
}

func (r *RAID) PrintBrief() {
	if err := r.getSystemDiskRAID(); err != nil {
		fmt.Printf("%v \n", err)
	}

	println("[RAID INFO]")
	fmt.Fprintf(os.Stdout, "%s%-*s: %v\n\n", "    ", 36, "SystemDisk", r.SystemDisk)
	ctrFields := []string{"ProductName", "ID", "ControllerStatus", "LogicalDrives", "Location", "State", "Capacity"}
	var sb *strings.Builder
	for _, ctr := range r.Controllers.Controller {
		sb = utils.SelectFields(ctr, ctrFields, 1)
		for _, vd := range ctr.LogicalDrives {
			for _, pd := range vd.PhysicalDrives {
				pdDetail := fmt.Sprintf("%s %s %s %s", pd.Vendor, pd.Capacity, pd.FormFactor, pd.RotationRate)
				sb.WriteString("\n    	" + pdDetail)
			}
		}
	}
	println(sb.String())
}

func (r *RAID) PrintDetail() {}

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
