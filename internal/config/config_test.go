package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.MaxLines != 300 {
		t.Errorf("MaxLines = %d, want 300", cfg.MaxLines)
	}
	if cfg.Filters.GrepMaxResults != 200 {
		t.Errorf("GrepMaxResults = %d, want 200", cfg.Filters.GrepMaxResults)
	}
	if cfg.Filters.GrepMaxPerFile != 25 {
		t.Errorf("GrepMaxPerFile = %d, want 25", cfg.Filters.GrepMaxPerFile)
	}
	if cfg.Filters.GitStatusMax != 15 {
		t.Errorf("GitStatusMax = %d, want 15", cfg.Filters.GitStatusMax)
	}
	if cfg.Filters.GitDiffMaxLines != 100 {
		t.Errorf("GitDiffMaxLines = %d, want 100", cfg.Filters.GitDiffMaxLines)
	}
	if cfg.Filters.GitLogMaxCommits != 10 {
		t.Errorf("GitLogMaxCommits = %d, want 10", cfg.Filters.GitLogMaxCommits)
	}
	if cfg.Filters.FindMaxResults != 100 {
		t.Errorf("FindMaxResults = %d, want 100", cfg.Filters.FindMaxResults)
	}
	if cfg.Filters.TestMaxFailures != 10 {
		t.Errorf("TestMaxFailures = %d, want 10", cfg.Filters.TestMaxFailures)
	}
}

func TestLoadFromNonExistent(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Non-existent file should return default, got error: %v", err)
	}
	if cfg.MaxLines != 300 {
		t.Error("Should return default config")
	}
}

func TestLoadFromValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `max_lines: 500

filters:
  grep_max_results: 100
  grep_max_per_file: 10
  git_status_max: 20
  git_diff_max_lines: 50
  git_log_max_commits: 5
  find_max_results: 50
  test_max_failures: 5
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom error: %v", err)
	}
	if cfg.MaxLines != 500 {
		t.Errorf("MaxLines = %d, want 500", cfg.MaxLines)
	}
	if cfg.Filters.GrepMaxResults != 100 {
		t.Errorf("GrepMaxResults = %d, want 100", cfg.Filters.GrepMaxResults)
	}
	if cfg.Filters.GitLogMaxCommits != 5 {
		t.Errorf("GitLogMaxCommits = %d, want 5", cfg.Filters.GitLogMaxCommits)
	}
}

func TestIsDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Disabled = []string{"grep", "build"}

	if !cfg.IsDisabled("grep") {
		t.Error("grep should be disabled")
	}
	if !cfg.IsDisabled("GREP") { // case insensitive
		t.Error("GREP should be disabled (case insensitive)")
	}
	if !cfg.IsDisabled("build") {
		t.Error("build should be disabled")
	}
	if cfg.IsDisabled("git-status") {
		t.Error("git-status should not be disabled")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yaml")

	cfg := DefaultConfig()
	cfg.MaxLines = 999
	cfg.Filters.GrepMaxResults = 50
	cfg.Disabled = []string{"build", "grep"}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo error: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom error: %v", err)
	}
	if loaded.MaxLines != 999 {
		t.Errorf("MaxLines = %d, want 999", loaded.MaxLines)
	}
	if loaded.Filters.GrepMaxResults != 50 {
		t.Errorf("GrepMaxResults = %d, want 50", loaded.Filters.GrepMaxResults)
	}
	if len(loaded.Disabled) != 2 {
		t.Fatalf("Disabled length = %d, want 2", len(loaded.Disabled))
	}
	if loaded.Disabled[0] != "build" {
		t.Errorf("Disabled[0] = %q, want %q", loaded.Disabled[0], "build")
	}
	if loaded.Disabled[1] != "grep" {
		t.Errorf("Disabled[1] = %q, want %q", loaded.Disabled[1], "grep")
	}
}

func TestParseYAMLDisabledList(t *testing.T) {
	cfg := DefaultConfig()
	data := []byte(`max_lines: 300

disabled:
  - build
  - grep
  - git-log
`)
	if err := parseYAML(data, cfg); err != nil {
		t.Fatalf("parseYAML error: %v", err)
	}
	if len(cfg.Disabled) != 3 {
		t.Fatalf("Disabled length = %d, want 3", len(cfg.Disabled))
	}
	if cfg.Disabled[0] != "build" {
		t.Errorf("Disabled[0] = %q, want %q", cfg.Disabled[0], "build")
	}
	if cfg.Disabled[1] != "grep" {
		t.Errorf("Disabled[1] = %q, want %q", cfg.Disabled[1], "grep")
	}
	if cfg.Disabled[2] != "git-log" {
		t.Errorf("Disabled[2] = %q, want %q", cfg.Disabled[2], "git-log")
	}
}

func TestMarshalYAML(t *testing.T) {
	cfg := DefaultConfig()
	data := marshalYAML(cfg)
	s := string(data)

	if !strings.Contains(s, "max_lines: 300") {
		t.Error("Should contain max_lines")
	}
	if !strings.Contains(s, "grep_max_results: 200") {
		t.Error("Should contain grep_max_results")
	}
	if !strings.Contains(s, "# rtk-go configuration") {
		t.Error("Should contain comment header")
	}
}

func TestParseYAMLComments(t *testing.T) {
	cfg := DefaultConfig()
	data := []byte(`# This is a comment
max_lines: 400
# Another comment
filters:
  grep_max_results: 150
`)
	if err := parseYAML(data, cfg); err != nil {
		t.Fatalf("parseYAML error: %v", err)
	}
	if cfg.MaxLines != 400 {
		t.Errorf("MaxLines = %d, want 400", cfg.MaxLines)
	}
	if cfg.Filters.GrepMaxResults != 150 {
		t.Errorf("GrepMaxResults = %d, want 150", cfg.Filters.GrepMaxResults)
	}
}

func TestParseYAMLEmptyFile(t *testing.T) {
	cfg := DefaultConfig()
	if err := parseYAML([]byte(""), cfg); err != nil {
		t.Fatalf("parseYAML error: %v", err)
	}
	// Should keep defaults
	if cfg.MaxLines != 300 {
		t.Errorf("MaxLines should stay default, got %d", cfg.MaxLines)
	}
}

func TestParseYAMLPartial(t *testing.T) {
	cfg := DefaultConfig()
	data := []byte("max_lines: 100\n")
	if err := parseYAML(data, cfg); err != nil {
		t.Fatalf("parseYAML error: %v", err)
	}
	if cfg.MaxLines != 100 {
		t.Errorf("MaxLines = %d, want 100", cfg.MaxLines)
	}
	// Other values should keep defaults
	if cfg.Filters.GrepMaxResults != 200 {
		t.Errorf("GrepMaxResults should stay default, got %d", cfg.Filters.GrepMaxResults)
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath error: %v", err)
	}
	if !strings.Contains(path, "rtk-go") {
		t.Errorf("Path should contain rtk-go, got: %q", path)
	}
	if !strings.HasSuffix(path, "config.yaml") {
		t.Errorf("Path should end with config.yaml, got: %q", path)
	}
}
