// Package config loads poundai's YAML configuration.
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration to support YAML strings like "500ms".
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(dur)
	return nil
}

// Std returns the standard library duration, or def when unset.
func (d Duration) Std(def time.Duration) time.Duration {
	if d == 0 {
		return def
	}
	return time.Duration(d)
}

// Service configures a single provider profile.
type Service struct {
	Provider     string            `yaml:"provider"` // openai | ollama | gemini | anthropic | bedrock
	BaseURL      string            `yaml:"base_url,omitempty"`
	APIKey       string            `yaml:"api_key,omitempty"`
	APIKeyEnv    string            `yaml:"api_key_env,omitempty"`
	Model        string            `yaml:"model"`
	Temperature  *float64          `yaml:"temperature,omitempty"`
	MaxTokens    int               `yaml:"max_tokens,omitempty"`
	Timeout      Duration          `yaml:"timeout,omitempty"`
	Organization string            `yaml:"organization,omitempty"` // openai
	Region       string            `yaml:"region,omitempty"`       // bedrock
	ExtraHeaders map[string]string `yaml:"extra_headers,omitempty"`
}

// ResolveAPIKey returns the API key, preferring the literal value and
// falling back to the configured environment variable.
func (s Service) ResolveAPIKey() string {
	if s.APIKey != "" {
		return s.APIKey
	}
	if s.APIKeyEnv != "" {
		return os.Getenv(s.APIKeyEnv)
	}
	return ""
}

// ContextCommand is a user-defined command whose output is injected into the
// prompt as additional context.
type ContextCommand struct {
	Name     string   `yaml:"name"`
	Command  string   `yaml:"command"`
	Timeout  Duration `yaml:"timeout"`
	MaxBytes int      `yaml:"max_bytes"`
}

// Context configures what extra context gets injected into the prompt.
type Context struct {
	Cwd      bool             `yaml:"cwd,omitempty"`
	OS       bool             `yaml:"os,omitempty"`
	History  int              `yaml:"history,omitempty"` // last N history entries from $HISTFILE
	Env      []string         `yaml:"env,omitempty"`     // env var names to include
	Commands []ContextCommand `yaml:"commands,omitempty"`
}

// Prompt configures the system prompt.
type Prompt struct {
	System        string `yaml:"system,omitempty"`         // full override of the system prompt
	SystemExtra   string `yaml:"system_extra,omitempty"`   // appended to the base system prompt
	OptimizedFile string `yaml:"optimized_file,omitempty"` // prompt artifact: {system, demos} JSON (see prompt.example.json)
}

// Config is the root configuration.
type Config struct {
	Service  string             `yaml:"service"`
	Services map[string]Service `yaml:"services"`
	Prompt   Prompt             `yaml:"prompt,omitempty"`
	Context  Context            `yaml:"context,omitempty"`
}

// configRelPath is the config file path relative to XDG config directories.
const configRelPath = "poundai/config.yml"

// DefaultPath returns the default config file location using XDG base
// directories. It first searches XDG_CONFIG_HOME and XDG_CONFIG_DIRS for
// an existing file, then falls back to XDG_CONFIG_HOME/poundai/config.yml.
func DefaultPath() string {
	// Search existing config across all XDG config directories.
	if path, err := xdg.SearchConfigFile(configRelPath); err == nil {
		return path
	}
	return filepath.Join(xdg.ConfigHome, configRelPath)
}

func UserPath() string {
	return filepath.Join(xdg.ConfigHome, configRelPath)
}

// Load reads and validates the config at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s (see config.example.yaml for the expected format)", path)
		}
		return nil, err
	}
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(cfg.Services) == 0 {
		return nil, fmt.Errorf("%s: no services defined", path)
	}
	if cfg.Service == "" {
		if len(cfg.Services) == 1 {
			for name := range cfg.Services {
				cfg.Service = name
			}
		} else {
			return nil, fmt.Errorf("%s: multiple services defined but no top-level 'service' selector", path)
		}
	}
	if _, ok := cfg.Services[cfg.Service]; !ok {
		return nil, fmt.Errorf("%s: selected service %q not found in services", path, cfg.Service)
	}
	return &cfg, nil
}

// Active returns the selected service profile. name overrides the config's
// selector when non-empty.
func (c *Config) Active(name string) (Service, error) {
	if name == "" {
		name = c.Service
	}
	svc, ok := c.Services[name]
	if !ok {
		return Service{}, fmt.Errorf("service %q not found in config", name)
	}
	if svc.Provider == "" {
		return Service{}, fmt.Errorf("service %q: missing 'provider'", name)
	}
	if svc.Model == "" {
		return Service{}, fmt.Errorf("service %q: missing 'model'", name)
	}
	return svc, nil
}
