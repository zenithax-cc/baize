// Package ipmi - sel.go collects and filters IPMI System Event Log (SEL) entries
// via `ipmitool sel elist`, retaining only error/warning-level events.
package ipmi

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

// criticalKeywords are matched (case-insensitive) against the SEL event text.
// Only entries containing at least one keyword are included in the output,
// reducing noise from routine informational events.
var criticalKeywords = []string{
	"fatal", "critical", "error", "err",
	"uncorrectable", "failed", "failure",
	"assert", "degraded", "fault",
}

// collectSEL runs `ipmitool sel elist` and parses SEL entries.
// Only events that match at least one critical keyword are stored.
// The SEL list is limited to the most-recent 200 entries to avoid
// excessive output on systems with large event logs.
//
// ipmitool sel elist output format (space/tab-separated):
//
//	<ID> | <Date> <Time> | <Sensor> | <Event> | <Direction>
func (m *IPMI) collectSEL(ctx context.Context) error {
	out := execute.CommandWithContext(ctx, ipmitool, "sel", "elist", "last", "200")
	if out.Err != nil {
		// Fall back to full list if "last N" is not supported by the BMC firmware.
		out = execute.CommandWithContext(ctx, ipmitool, "sel", "elist")
		if out.Err != nil {
			return out.Err
		}
	}

	m.SEL = make([]*SELEntry, 0, 16)

	scanner := bufio.NewScanner(bytes.NewReader(out.Stdout))
	for scanner.Scan() {
		line := scanner.Text()
		entry := parseSELLine(line)
		if entry == nil {
			continue
		}
		// Filter: keep only events matching critical keywords.
		if matchesCriticalKeyword(entry.Event + " " + entry.Sensor) {
			m.SEL = append(m.SEL, entry)
		}
	}

	return scanner.Err()
}

// parseSELLine parses a single ipmitool sel elist output line into a SELEntry.
// Returns nil if the line cannot be parsed.
//
// Expected format:
//
//	0001 | 01/01/2024 00:00:00 | Processor | IERR | Asserted
func parseSELLine(line string) *SELEntry {
	parts := strings.Split(line, "|")
	if len(parts) < 5 {
		return nil
	}

	entry := &SELEntry{
		ID:        strings.TrimSpace(parts[0]),
		Timestamp: strings.TrimSpace(parts[1]),
		Sensor:    strings.TrimSpace(parts[2]),
		Event:     strings.TrimSpace(parts[3]),
		Direction: strings.TrimSpace(parts[4]),
	}

	// Assign severity based on event and direction text.
	entry.Severity = classifySELSeverity(entry.Event, entry.Direction)

	return entry
}

// classifySELSeverity assigns a severity level based on the event description
// and assertion direction.
func classifySELSeverity(event, direction string) string {
	combined := strings.ToLower(event + " " + direction)
	switch {
	case strings.Contains(combined, "fatal") ||
		strings.Contains(combined, "critical") ||
		strings.Contains(combined, "uncorrectable") ||
		strings.Contains(combined, "ierr"):
		return "Critical"
	case strings.Contains(combined, "error") ||
		strings.Contains(combined, "err") ||
		strings.Contains(combined, "failed") ||
		strings.Contains(combined, "fault") ||
		strings.Contains(combined, "degraded"):
		return "Warning"
	default:
		return "Info"
	}
}

// matchesCriticalKeyword returns true if the text contains any of the
// predefined critical keywords (case-insensitive).
func matchesCriticalKeyword(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range criticalKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
