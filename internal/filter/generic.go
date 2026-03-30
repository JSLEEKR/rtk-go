package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JSLEEKR/rtk-go/internal/config"
)

// GenericFilter is the fallback filter applied when no command-specific filter matches.
// It strips ANSI escape codes, collapses blank lines, and applies smart truncation.
type GenericFilter struct{}

func (f *GenericFilter) Name() string { return "generic" }

// Match always returns true — this is the catch-all fallback.
func (f *GenericFilter) Match(cmd string, args []string) bool {
	return true
}

// MaxGenericLines is the max output lines for the generic filter.
const MaxGenericLines = 300

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func (f *GenericFilter) Apply(output string, exitCode int, cfg *config.FilterConfig) string {
	if output == "" {
		return ""
	}

	// Strip ANSI codes
	cleaned := StripANSI(output)

	// M3 fix: Collapse multiple blank lines (3+ -> 2, matching comment)
	lines := strings.Split(cleaned, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if trimmed == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, "")
			}
			continue
		}
		blankCount = 0
		result = append(result, trimmed)
	}

	// Trim trailing blank lines
	for len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}

	// Smart truncation: use cfg.MaxLines if set, otherwise default
	maxLines := MaxGenericLines
	if cfg != nil && cfg.MaxLines > 0 {
		maxLines = cfg.MaxLines
	}
	if len(result) > maxLines {
		keepHead := maxLines * 2 / 3
		keepTail := maxLines - keepHead
		omitted := len(result) - keepHead - keepTail

		truncated := make([]string, 0, maxLines+1)
		truncated = append(truncated, result[:keepHead]...)
		truncated = append(truncated, fmt.Sprintf("\n... [%d lines omitted] ...\n", omitted))
		truncated = append(truncated, result[len(result)-keepTail:]...)
		result = truncated
	}

	return strings.Join(result, "\n")
}
