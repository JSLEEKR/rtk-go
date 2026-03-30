// Package config provides YAML-based configuration for rtk-go.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all rtk-go configuration.
type Config struct {
	// MaxLines is the global maximum output lines before truncation.
	MaxLines int `yaml:"max_lines"`

	// Filters configures per-filter behavior.
	Filters FilterConfig `yaml:"filters"`

	// Disabled lists filter names that should be bypassed.
	Disabled []string `yaml:"disabled"`
}

// FilterConfig holds per-filter configuration.
type FilterConfig struct {
	GrepMaxResults int `yaml:"grep_max_results"`
	GrepMaxPerFile int `yaml:"grep_max_per_file"`
	GitStatusMax   int `yaml:"git_status_max"`
	GitDiffMaxLines int `yaml:"git_diff_max_lines"`
	GitLogMaxCommits int `yaml:"git_log_max_commits"`
	FindMaxResults  int `yaml:"find_max_results"`
	TestMaxFailures int `yaml:"test_max_failures"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxLines: 300,
		Filters: FilterConfig{
			GrepMaxResults:   200,
			GrepMaxPerFile:   25,
			GitStatusMax:     15,
			GitDiffMaxLines:  100,
			GitLogMaxCommits: 10,
			FindMaxResults:   100,
			TestMaxFailures:  10,
		},
	}
}

// ConfigPath returns the default config file path.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "rtk-go", "config.yaml"), nil
}

// Load reads configuration from the default path.
// Returns DefaultConfig if the file doesn't exist.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return DefaultConfig(), nil
	}
	return LoadFrom(path)
}

// LoadFrom reads configuration from a specific path.
// Returns DefaultConfig if the file doesn't exist.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := parseYAML(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}

// IsDisabled checks if a filter name is in the disabled list.
func (c *Config) IsDisabled(filterName string) bool {
	for _, d := range c.Disabled {
		if strings.EqualFold(d, filterName) {
			return true
		}
	}
	return false
}

// Save writes configuration to the default path.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes configuration to a specific path.
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data := marshalYAML(c)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// parseYAML is a minimal YAML parser for flat/simple YAML.
// We implement this to avoid external dependencies.
func parseYAML(data []byte, cfg *Config) error {
	lines := strings.Split(string(data), "\n")
	section := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// L4 fix: Check for section headers (removed space check to allow headers with spaces)
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, ": ") {
				section = strings.TrimSuffix(trimmed, ":")
				continue
			}
		}

		// H4 fix: Handle YAML list items in one place only (disabled section)
		if section == "disabled" && strings.HasPrefix(trimmed, "- ") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if item != "" {
				cfg.Disabled = append(cfg.Disabled, item)
			}
			continue
		}

		// Parse key: value
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch section {
		case "":
			switch key {
			case "max_lines":
				if v, err := strconv.Atoi(value); err == nil {
					cfg.MaxLines = v
				}
			}
		case "filters":
			switch key {
			case "grep_max_results":
				if v, err := strconv.Atoi(value); err == nil {
					cfg.Filters.GrepMaxResults = v
				}
			case "grep_max_per_file":
				if v, err := strconv.Atoi(value); err == nil {
					cfg.Filters.GrepMaxPerFile = v
				}
			case "git_status_max":
				if v, err := strconv.Atoi(value); err == nil {
					cfg.Filters.GitStatusMax = v
				}
			case "git_diff_max_lines":
				if v, err := strconv.Atoi(value); err == nil {
					cfg.Filters.GitDiffMaxLines = v
				}
			case "git_log_max_commits":
				if v, err := strconv.Atoi(value); err == nil {
					cfg.Filters.GitLogMaxCommits = v
				}
			case "find_max_results":
				if v, err := strconv.Atoi(value); err == nil {
					cfg.Filters.FindMaxResults = v
				}
			case "test_max_failures":
				if v, err := strconv.Atoi(value); err == nil {
					cfg.Filters.TestMaxFailures = v
				}
			}
		}
	}

	return nil
}

// marshalYAML serializes config to YAML format.
func marshalYAML(cfg *Config) []byte {
	var b strings.Builder
	b.WriteString("# rtk-go configuration\n")
	b.WriteString(fmt.Sprintf("max_lines: %d\n", cfg.MaxLines))
	b.WriteString("\nfilters:\n")
	b.WriteString(fmt.Sprintf("  grep_max_results: %d\n", cfg.Filters.GrepMaxResults))
	b.WriteString(fmt.Sprintf("  grep_max_per_file: %d\n", cfg.Filters.GrepMaxPerFile))
	b.WriteString(fmt.Sprintf("  git_status_max: %d\n", cfg.Filters.GitStatusMax))
	b.WriteString(fmt.Sprintf("  git_diff_max_lines: %d\n", cfg.Filters.GitDiffMaxLines))
	b.WriteString(fmt.Sprintf("  git_log_max_commits: %d\n", cfg.Filters.GitLogMaxCommits))
	b.WriteString(fmt.Sprintf("  find_max_results: %d\n", cfg.Filters.FindMaxResults))
	b.WriteString(fmt.Sprintf("  test_max_failures: %d\n", cfg.Filters.TestMaxFailures))

	if len(cfg.Disabled) > 0 {
		b.WriteString("\ndisabled:\n")
		for _, d := range cfg.Disabled {
			b.WriteString(fmt.Sprintf("  - %s\n", d))
		}
	}

	return []byte(b.String())
}
