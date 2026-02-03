package product

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/zenithax-cc/baize/pkg/utils"
)

const (
	hostNamePath      = "/proc/sys/kernel/hostname"
	ostypePath        = "/proc/sys/kernel/ostype"
	kernelReleasePath = "/proc/sys/kernel/osrelease"
	kernelVersionPath = "/proc/sys/kernel/version"
	osReleasePath     = "/etc/os-release"
	centosReleasePath = "/etc/centos-release"
	redhatReleasePath = "/etc/redhat-release"
	rockyReleasePath  = "/etc/rocky-release"
	debianVersionPath = "/etc/debian_version"
)

var (
	regexVersion = regexp.MustCompile(`[\( ]([\d\.]+)`) // Ubuntu, RHEL 通用
	regexCentos  = regexp.MustCompile(`^CentOS(?: Linux)? release ([\d\.]+)`)
	regexRocky   = regexp.MustCompile(`^Rocky Linux release ([\d\.]+)`)
	regexDebian  = regexp.MustCompile(`^([\d\.]+)`)

	osReleaseFieldMap = map[string]func(*OS, string){
		"PRETTY_NAME":      func(os *OS, v string) { os.PrettyName = v },
		"NAME":             func(os *OS, v string) { os.Distr = v },
		"VERSION_ID":       func(os *OS, v string) { os.DistrVersion = v },
		"VERSION_CODENAME": func(os *OS, v string) { os.CodeName = v },
		"ID_LIKE":          func(os *OS, v string) { os.IDLike = v },
	}
)

type distroMatcher struct {
	prefix   string
	filePath string
	regex    *regexp.Regexp
	submatch int
}

var distroMatchers = []distroMatcher{
	{prefix: "ubuntu", regex: regexVersion, submatch: 1},
	{prefix: "centos", filePath: centosReleasePath, regex: regexCentos, submatch: 1},
	{prefix: "rocky", filePath: rockyReleasePath, regex: regexRocky, submatch: 1},
	{prefix: "rhel", filePath: redhatReleasePath, regex: regexVersion, submatch: 1},
	{prefix: "red hat", filePath: redhatReleasePath, regex: regexVersion, submatch: 1},
	{prefix: "debian", filePath: debianVersionPath, regex: regexDebian, submatch: 1},
}

type kernelCfg struct {
	path   string
	target *string
}

func (p *Product) collectKernel() error {
	kernelCfgs := []kernelCfg{
		{path: ostypePath, target: &p.OS.KernelName},
		{path: kernelReleasePath, target: &p.OS.KernelRelease},
		{path: kernelVersionPath, target: &p.OS.KernelVersion},
		{path: hostNamePath, target: &p.OS.HostName},
	}

	errs := make([]error, 0, len(kernelCfgs))
	for _, cfg := range kernelCfgs {
		content, err := os.ReadFile(cfg.path)
		if err != nil {
			errs = append(errs, fmt.Errorf("read %s: %w", cfg.path, err))
			*cfg.target = "Unknown"
			continue
		}

		*cfg.target = strings.TrimSpace(string(content))
	}

	return errors.Join(errs...)
}

func (p *Product) collectDistribution() error {
	lines, err := utils.ReadLines(osReleasePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", osReleasePath, err)
	}

	if len(lines) == 0 {
		return fmt.Errorf("no information found in %s", osReleasePath)
	}

	for _, line := range lines {
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		if fn, ok := osReleaseFieldMap[key]; ok {
			fn(&p.OS, value)
		}
	}

	p.OS.MinorVersion = getMinorVersion(p.OS.Distr)

	return nil
}

func getMinorVersion(distr string) string {
	if distr == "" {
		return "Unknown"
	}

	lowerDistr := strings.ToLower(distr)

	for _, matcher := range distroMatchers {
		if !strings.Contains(lowerDistr, matcher.prefix) {
			continue
		}

		content := []byte(distr)
		if matcher.filePath != "" {
			var err error
			content, err = os.ReadFile(matcher.filePath)
			if err != nil {
				continue
			}
		}

		if matches := matcher.regex.FindSubmatch(content); len(matches) > matcher.submatch {
			return string(matches[matcher.submatch])
		}
	}

	return "Unknown"
}
