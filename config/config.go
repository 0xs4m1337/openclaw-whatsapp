// Package config handles loading and managing application configuration
// from YAML files and environment variable overrides.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// WebhookFilters controls which messages are forwarded to the webhook.
type WebhookFilters struct {
	DMOnly       bool     `yaml:"dm_only"`
	IgnoreGroups []string `yaml:"ignore_groups"`
}

// Config holds all application configuration values.
type Config struct {
	Port               int            `yaml:"port"`
	DataDir            string         `yaml:"data_dir"`
	WebhookURL         string         `yaml:"webhook_url"`
	WebhookFilters     WebhookFilters `yaml:"webhook_filters"`
	AutoReconnect      bool           `yaml:"auto_reconnect"`
	ReconnectInterval  Duration       `yaml:"reconnect_interval"`
	LogLevel           string         `yaml:"log_level"`
}

// Duration is a wrapper around time.Duration that supports YAML unmarshalling
// from human-readable strings like "30s", "5m", "1h".
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for Duration.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface for Duration.
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

// defaults returns a Config populated with sensible default values.
func defaults() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return &Config{
		Port:              8555,
		DataDir:           filepath.Join(homeDir, ".openclaw-whatsapp"),
		WebhookURL:        "",
		WebhookFilters:    WebhookFilters{},
		AutoReconnect:     true,
		ReconnectInterval: Duration{30 * time.Second},
		LogLevel:          "info",
	}
}

// Load reads configuration from the YAML file at path, falling back to
// defaults if the file does not exist. Environment variables with the
// OC_WA_ prefix override any file or default values.
func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		// File doesn't exist — proceed with defaults.
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

// applyEnvOverrides applies OC_WA_* environment variable overrides to cfg.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("OC_WA_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Port = p
		}
	}
	if v := os.Getenv("OC_WA_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("OC_WA_WEBHOOK_URL"); v != "" {
		cfg.WebhookURL = v
	}
	if v := os.Getenv("OC_WA_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("OC_WA_RECONNECT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ReconnectInterval = Duration{d}
		}
	}
	if v := os.Getenv("OC_WA_AUTO_RECONNECT"); v != "" {
		switch strings.ToLower(v) {
		case "true", "1", "yes":
			cfg.AutoReconnect = true
		case "false", "0", "no":
			cfg.AutoReconnect = false
		}
	}
}

// EnsureDataDir creates the DataDir and its media subdirectory if they
// do not already exist.
func (c *Config) EnsureDataDir() error {
	if err := os.MkdirAll(c.DataDir, 0o755); err != nil {
		return fmt.Errorf("creating data dir %s: %w", c.DataDir, err)
	}
	mediaDir := filepath.Join(c.DataDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return fmt.Errorf("creating media dir %s: %w", mediaDir, err)
	}
	return nil
}
