// Package config manages experiment configuration (autoresearch.config.json).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultFileName = "autoresearch.config.json"

type Config struct {
	MaxIterations int    `json:"maxIterations"`
	WorkingDir    string `json:"workingDir"`
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
