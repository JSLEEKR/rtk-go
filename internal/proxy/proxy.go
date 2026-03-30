// Package proxy executes commands and pipes their output through filters.
// Unlike rtk which uses shell interpolation (sh -c), rtk-go uses exec.Command
// with explicit argument arrays to prevent shell injection vulnerabilities.
package proxy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/JSLEEKR/rtk-go/internal/config"
	"github.com/JSLEEKR/rtk-go/internal/filter"
	"github.com/JSLEEKR/rtk-go/internal/report"
	"github.com/JSLEEKR/rtk-go/internal/token"
)

// Proxy intercepts command execution and applies filters to compress output.
type Proxy struct {
	Registry *filter.Registry
	Config   *config.Config
	Reporter *report.Reporter
	Stdout   io.Writer
	Stderr   io.Writer
	// Passthrough disables filtering (for --raw flag).
	Passthrough bool
	// ShowReport enables the savings line on stderr (M1/M2 fix).
	ShowReport bool
}

// New creates a Proxy with default settings.
func New() *Proxy {
	cfg, _ := config.Load()
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &Proxy{
		Registry: filter.NewRegistry(),
		Config:   cfg,
		Reporter: report.New(),
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
	}
}

// Result holds the outcome of a proxied command execution.
type Result struct {
	// ExitCode from the underlying command.
	ExitCode int
	// RawOutput is the unfiltered stdout output.
	RawOutput string
	// RawStderr is the unfiltered stderr output.
	RawStderr string
	// FilteredOutput is the compressed output (stdout only).
	FilteredOutput string
	// FilterName is the name of the filter that was applied.
	FilterName string
	// Stats holds token counting statistics.
	Stats token.Stats
}

// Run executes a command, filters its output, and writes to stdout.
// It returns the exit code of the underlying command.
func (p *Proxy) Run(cmdName string, args []string) (int, error) {
	result, err := p.Execute(cmdName, args)
	if err != nil {
		return 1, err
	}

	// Write filtered stdout
	if result.FilteredOutput != "" {
		fmt.Fprint(p.Stdout, result.FilteredOutput)
		if !strings.HasSuffix(result.FilteredOutput, "\n") {
			fmt.Fprintln(p.Stdout)
		}
	}

	// H5 fix: write stderr separately (unfiltered)
	if result.RawStderr != "" {
		fmt.Fprint(p.Stderr, result.RawStderr)
		if !strings.HasSuffix(result.RawStderr, "\n") {
			fmt.Fprintln(p.Stderr)
		}
	}

	// M2 fix: Only print savings line when --report flag is set
	if p.ShowReport && result.Stats.Saved() > 0 {
		fmt.Fprintf(p.Stderr, "\n--- rtk-go: %s | %d->%d tokens (%.0f%% saved)\n",
			result.FilterName,
			result.Stats.InputTokens,
			result.Stats.OutputTokens,
			result.Stats.SavingsPercent(),
		)
	}

	return result.ExitCode, nil
}

// Execute runs a command and returns the full result without writing to stdout.
// This is useful for testing and programmatic use.
func (p *Proxy) Execute(cmdName string, args []string) (*Result, error) {
	// SECURITY: Use exec.Command with explicit args — NO shell interpolation.
	// This prevents the shell injection vulnerability found in rtk (issue #640).
	cmd := exec.Command(cmdName, args...)
	cmd.Stdin = os.Stdin

	// H5 fix: capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("executing %s: %w", cmdName, err)
		}
	}

	rawOutput := stdout.String()
	rawStderr := stderr.String()

	// Passthrough mode or empty output
	if p.Passthrough || rawOutput == "" {
		// M1 fix: in passthrough, combine for display but note mode
		combined := rawOutput
		return &Result{
			ExitCode:       exitCode,
			RawOutput:      rawOutput,
			RawStderr:      rawStderr,
			FilteredOutput: combined,
			FilterName:     "passthrough",
			Stats: token.Stats{
				InputTokens:  token.Count(rawOutput),
				OutputTokens: token.Count(rawOutput),
			},
		}, nil
	}

	// Find matching filter
	f := p.Registry.Lookup(cmdName, args)

	// Check if filter is disabled
	if p.Config.IsDisabled(f.Name()) {
		f = &filter.GenericFilter{}
	}

	// Apply filter with config (C1 fix: pass config to filters)
	filtered := applyFilterSafe(f, rawOutput, exitCode, &p.Config.Filters)

	inputTokens := token.Count(rawOutput)
	outputTokens := token.Count(filtered)

	// Record stats
	p.Reporter.Record(f.Name(), inputTokens, outputTokens)

	return &Result{
		ExitCode:       exitCode,
		RawOutput:      rawOutput,
		RawStderr:      rawStderr,
		FilteredOutput: filtered,
		FilterName:     f.Name(),
		Stats: token.Stats{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		},
	}, nil
}

// applyFilterSafe applies a filter with panic recovery.
// On any failure, it returns the raw output unchanged.
func applyFilterSafe(f filter.Filter, output string, exitCode int, cfg *config.FilterConfig) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = output // fail-safe: return raw output
		}
	}()
	return f.Apply(output, exitCode, cfg)
}
