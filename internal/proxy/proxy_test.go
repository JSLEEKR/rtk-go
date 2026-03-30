package proxy

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/JSLEEKR/rtk-go/internal/config"
	"github.com/JSLEEKR/rtk-go/internal/filter"
	"github.com/JSLEEKR/rtk-go/internal/report"
)

func TestNewProxy(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.Registry == nil {
		t.Error("Registry should not be nil")
	}
	if p.Config == nil {
		t.Error("Config should not be nil")
	}
	if p.Reporter == nil {
		t.Error("Reporter should not be nil")
	}
}

func echoCmd() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", "echo", "hello world"}
	}
	return "echo", []string{"hello world"}
}

func TestExecuteSimpleCommand(t *testing.T) {
	p := New()
	cmd, args := echoCmd()
	result, err := p.Execute(cmd, args)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.RawOutput, "hello") {
		t.Errorf("RawOutput should contain 'hello', got: %q", result.RawOutput)
	}
}

func TestExecutePassthrough(t *testing.T) {
	p := New()
	p.Passthrough = true
	cmd, args := echoCmd()
	result, err := p.Execute(cmd, args)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.FilterName != "passthrough" {
		t.Errorf("FilterName = %q, want passthrough", result.FilterName)
	}
	if result.RawOutput != result.FilteredOutput {
		t.Error("Passthrough should not modify output")
	}
}

func TestExecuteNonExistentCommand(t *testing.T) {
	p := New()
	_, err := p.Execute("nonexistent_command_xyz_123", nil)
	if err == nil {
		t.Error("Non-existent command should return error")
	}
}

func TestExecuteExitCode(t *testing.T) {
	p := New()
	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "exit", "42"}
	} else {
		cmd = "bash"
		args = []string{"-c", "exit 42"}
	}
	result, err := p.Execute(cmd, args)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}
}

func TestExecuteRecordsStats(t *testing.T) {
	p := New()
	cmd, args := echoCmd()
	result, err := p.Execute(cmd, args)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Stats.InputTokens == 0 {
		t.Error("InputTokens should be > 0")
	}
}

func TestRunWritesToStdout(t *testing.T) {
	p := New()
	var buf bytes.Buffer
	p.Stdout = &buf
	p.Stderr = &bytes.Buffer{}

	cmd, args := echoCmd()
	exitCode, err := p.Run(cmd, args)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", exitCode)
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("stdout should contain 'hello', got: %q", buf.String())
	}
}

func TestExecuteDisabledFilter(t *testing.T) {
	p := New()
	p.Config = config.DefaultConfig()
	p.Config.Disabled = []string{"generic"}

	cmd, args := echoCmd()
	result, err := p.Execute(cmd, args)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// Even with "generic" disabled, it should still use generic as the lookup itself returns generic
	// and the disabled check replaces it with generic. This tests the path.
	_ = result
}

func TestApplyFilterSafePanic(t *testing.T) {
	// Create a filter that panics
	panicFilter := &panicingFilter{}
	output := "raw output"
	result := applyFilterSafe(panicFilter, output, 0, nil)
	if result != output {
		t.Errorf("Panic should return raw output, got: %q", result)
	}
}

type panicingFilter struct{}

func (f *panicingFilter) Name() string                                               { return "panic" }
func (f *panicingFilter) Match(cmd string, args []string) bool                       { return true }
func (f *panicingFilter) Apply(output string, exitCode int, cfg *config.FilterConfig) string {
	panic("intentional panic")
}

func TestExecuteFilterSelection(t *testing.T) {
	p := New()
	p.Registry = filter.NewRegistry()

	// Test that git status uses git-status filter
	f := p.Registry.Lookup("git", []string{"status"})
	if f.Name() != "git-status" {
		t.Errorf("Expected git-status, got: %q", f.Name())
	}
}

func TestResultStats(t *testing.T) {
	result := &Result{
		Stats: struct {
			InputTokens  int
			OutputTokens int
		}{1000, 100},
	}
	if result.Stats.InputTokens != 1000 {
		t.Error("InputTokens wrong")
	}
	if result.Stats.OutputTokens != 100 {
		t.Error("OutputTokens wrong")
	}
}

func TestProxyReporter(t *testing.T) {
	p := New()
	p.Reporter = report.New()

	cmd, args := echoCmd()
	_, err := p.Execute(cmd, args)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	entries := p.Reporter.Entries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}

// M2 fix: Test that savings line is not printed without --report
func TestRunNoSavingsWithoutReport(t *testing.T) {
	p := New()
	var stdout, stderr bytes.Buffer
	p.Stdout = &stdout
	p.Stderr = &stderr
	p.ShowReport = false

	cmd, args := echoCmd()
	_, err := p.Run(cmd, args)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if strings.Contains(stderr.String(), "rtk-go:") {
		t.Error("Savings line should not appear without --report")
	}
}

// H5 fix: Test that stderr is captured separately
func TestExecuteSeparateStderr(t *testing.T) {
	p := New()
	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "echo stdout_msg && echo stderr_msg 1>&2"}
	} else {
		cmd = "bash"
		args = []string{"-c", "echo stdout_msg; echo stderr_msg >&2"}
	}
	result, err := p.Execute(cmd, args)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !strings.Contains(result.RawOutput, "stdout_msg") {
		t.Errorf("RawOutput should contain stdout, got: %q", result.RawOutput)
	}
	if !strings.Contains(result.RawStderr, "stderr_msg") {
		t.Errorf("RawStderr should contain stderr, got: %q", result.RawStderr)
	}
}
