package pkg

import (
	"fmt"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/health"
)

type Health struct {
	*health.Health
}

func NewHealth() *Health {
	return &Health{
		Health: health.New(),
	}
}

func (h *Health) PrintJson() {
	printJson("Health", h.Health)
}

func (h *Health) PrintBrief() {
	println("[HEALTH INFO]")
	fields := []string{"GameInit", "Puppet", "Gpostd", "SSHPort"}
	sb := utils.SelectFields(h, fields, 1)
	fmt.Fprintf(sb, "%s%-*s: %v\n", "    ", 36, "HWhealth", h.HWhealth.State)
	if h.HWhealth.State == "error" {
		fmt.Fprintf(sb, "%s%-*s: %v\n", "    	", 32, "HWhealthErrors", h.HWhealth.Errors)
		fmt.Fprintf(sb, "%s%-*s: %v\n", "    	", 32, "HWhealthErrorDetail", h.HWhealth.ErrDetail)
	}
	fmt.Println(sb.String())
}

func (h *Health) PrintDetail() {}
