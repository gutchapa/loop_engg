// Package config manages experiment configuration (autoresearch.config.json).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultFileName = "autoresearch.config.json"

// TerminationCondition defines a single termination rule.
type TerminationCondition struct {
	Metric   string  `json:"metric"`             // metric name from METRIC lines
	Operator string  `json:"operator"`           // >=, <=, ==, >, <
	Value    float64 `json:"value"`              // threshold value
}

// TerminationConfig defines when the experiment loop should stop.
type TerminationConfig struct {
	MaxIterations int                   `json:"maxIterations,omitempty"`
	Conditions    []TerminationCondition `json:"conditions,omitempty"`
}

// AIProviderConfig configures an LLM provider for autonomous AI mode.
type AIProviderConfig struct {
	Provider string `json:"provider,omitempty"` // "openai", "grok", "deepseek", "ollama"
	Model    string `json:"model,omitempty"`    // model name
	Endpoint string `json:"endpoint,omitempty"` // API endpoint URL
	APIKey   string `json:"apiKey,omitempty"`   // API key (also read from LOOP_API_KEY env var)
}

// AIConfig configures the autonomous AI agent mode.
type AIConfig struct {
	MaxIterations int              `json:"maxIterations,omitempty"` // max AI loop iterations
	Provider      AIProviderConfig `json:"provider,omitempty"`
	FilesInScope  []string         `json:"filesInScope,omitempty"` // file patterns to include in context
}

// Config represents the full experiment configuration.
type Config struct {
	MetricName    string            `json:"metricName,omitempty"`
	MetricUnit    string            `json:"metricUnit,omitempty"`
	Direction     string            `json:"direction,omitempty"`   // "higher" or "lower"
	Command       string            `json:"command,omitempty"`     // default experiment command
	MaxIterations int               `json:"maxIterations"`
	WorkingDir    string            `json:"workingDir"`
	Termination   TerminationConfig `json:"termination,omitempty"`
	AI            *AIConfig         `json:"ai,omitempty"`          // AI agent config (optional)
}

func DefaultConfig() Config {
	wd, _ := os.Getwd()
	return Config{
		MaxIterations: 20,
		WorkingDir:    wd,
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultFileName
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func (c Config) Save(path string) error {
	if path == "" {
		path = DefaultFileName
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}
