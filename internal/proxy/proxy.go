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
	// RawOutput is the unfiltered output.
	RawOutput string
	// FilteredOutput is the compressed output.
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

	// Write output
	if result.FilteredOutput != "" {
		fmt.Fprint(p.Stdout, result.FilteredOutput)
		if !strings.HasSuffix(result.FilteredOutput, "\n") {
			fmt.Fprintln(p.Stdout)
		}
	}

	// Show savings report
	if result.Stats.Saved() > 0 {
		fmt.Fprintf(p.Stderr, "\n--- rtk-go: %s | %d→%d tokens (%.0f%% saved)\n",
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
	if stderr.Len() > 0 {
		if rawOutput != "" && !strings.HasSuffix(rawOutput, "\n") {
			rawOutput += "\n"
		}
		rawOutput += stderr.String()
	}

	// Passthrough mode or empty output
	if p.Passthrough || rawOutput == "" {
		return &Result{
			ExitCode:       exitCode,
			RawOutput:      rawOutput,
			FilteredOutput: rawOutput,
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

	// Apply filter (fail-safe: return raw on panic/error)
	filtered := applyFilterSafe(f, rawOutput, exitCode)

	inputTokens := token.Count(rawOutput)
	outputTokens := token.Count(filtered)

	// Record stats
	p.Reporter.Record(f.Name(), inputTokens, outputTokens)

	return &Result{
		ExitCode:       exitCode,
		RawOutput:      rawOutput,
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
func applyFilterSafe(f filter.Filter, output string, exitCode int) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = output // fail-safe: return raw output
		}
	}()
	return f.Apply(output, exitCode)
}
