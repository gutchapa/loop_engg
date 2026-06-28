// Package metric handles parsing structured METRIC lines from command output.
package metric

import (
	"fmt"
	"regexp"
	"strconv"
)

var metricLine = regexp.MustCompile(`^METRIC\s+(\w+)=(\S+)$`)

type Metric struct {
	Name  string
	Value string
}

// ParseLine parses a single "METRIC name=value" line.
// Returns nil if the line doesn't match.
func ParseLine(line string) *Metric {
	m := metricLine.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	return &Metric{Name: m[1], Value: m[2]}
}

// ParseFloat converts the metric value to float64.
func (m Metric) ParseFloat() (float64, error) {
	return strconv.ParseFloat(m.Value, 64)
}

// ParseInt converts the metric value to int.
func (m Metric) ParseInt() (int, error) {
	return strconv.Atoi(m.Value)
}

// ParseBool converts the metric value (0/1) to bool.
func (m Metric) ParseBool() (bool, error) {
	switch m.Value {
	case "1":
		return true, nil
	case "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool metric: %s", m.Value)
	}
}

// ParseAll extracts all METRIC lines from output text.
func ParseAll(output string) []Metric {
	var metrics []Metric
	for _, line := range regexp.MustCompile(`\r?\n`).Split(output, -1) {
		if m := ParseLine(line); m != nil {
			metrics = append(metrics, *m)
		}
	}
	return metrics
}
