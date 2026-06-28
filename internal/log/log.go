// Package log handles structured experiment logging (autoresearch.jsonl).
package log

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const DefaultFileName = "autoresearch.jsonl"

// Experiment represents one logged experiment run.
type Experiment struct {
	Type        string            `json:"type,omitempty"`
	Run         int               `json:"run,omitempty"`
	Commit      string            `json:"commit,omitempty"`
	Metric      float64           `json:"metric,omitempty"`
	Metrics     map[string]any   `json:"metrics,omitempty"`
	Status      string            `json:"status,omitempty"`
	Description string            `json:"description,omitempty"`
	Timestamp   int64             `json:"timestamp,omitempty"`
	Segment     int               `json:"segment,omitempty"`
	Confidence  *float64          `json:"confidence,omitempty"`
	ASI         map[string]any   `json:"asi,omitempty"`
}

// ConfigHeader represents the initial config entry in the JSONL file.
type ConfigHeader struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	MetricName   string `json:"metricName"`
	MetricUnit   string `json:"metricUnit"`
	BestDirection string `json:"bestDirection"`
}

// Logger manages appending to the experiment log file.
type Logger struct {
	path string
	file *os.File
}

func NewLogger(path string) (*Logger, error) {
	if path == "" {
		path = DefaultFileName
	}
	l := &Logger{path: path}
	return l, nil
}

func (l *Logger) close() {
	if l.file != nil {
		l.file.Close()
	}
}

// writeLine appends one JSON line to the log.
func (l *Logger) writeLine(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if _, err := f.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	return nil
}

// AppendExperiment writes an experiment result to the log.
func (l *Logger) AppendExperiment(exp Experiment) error {
	exp.Timestamp = time.Now().UnixMilli()
	return l.writeLine(exp)
}

// AppendConfig writes the initial config header to the log.
func (l *Logger) AppendConfig(cfg ConfigHeader) error {
	return l.writeLine(cfg)
}

// ReadAll reads all experiment entries from the log file.
func ReadAll(path string) ([]Experiment, error) {
	if path == "" {
		path = DefaultFileName
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read log: %w", err)
	}
	var exps []Experiment
	for _, line := range splitLines(string(data)) {
		if line == "" {
			continue
		}
		var exp Experiment
		if err := json.Unmarshal([]byte(line), &exp); err != nil {
			continue // skip unparseable lines (config headers, etc.)
		}
		exps = append(exps, exp)
	}
	return exps, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// LastRun returns the most recent run number from the log, or 0.
func LastRun(path string) (int, error) {
	exps, err := ReadAll(path)
	if err != nil {
		return 0, err
	}
	maxRun := 0
	for _, e := range exps {
		if e.Run > maxRun {
			maxRun = e.Run
		}
	}
	return maxRun, nil
}
