package pkg

import (
	"strings"

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
	var sb strings.Builder
	sb.Grow(1000)
	sb.WriteString("[HEALTH INFO]\n")

	fields := []string{"GameInit", "Puppet", "Gpostd", "SSHPort"}
	hwFields := []string{"State", "Errors", "ErrDetail"}

	sb.WriteString(selectFields(h.Health, fields, 1, colorMap["Health"]).String() + "\n")
	sb.WriteString(printSeparator("HWhealth", "", true, 1))
	sb.WriteString(selectFields(h.HWhealth, hwFields, 2, colorMap["Health"]).String() + "\n")

	println(sb.String())
}

func (h *Health) PrintDetail() {}
