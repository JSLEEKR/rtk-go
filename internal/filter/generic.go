package filter

import (
	"fmt"
	"regexp"
	"strings"
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

func (f *GenericFilter) Apply(output string, exitCode int) string {
	if output == "" {
		return ""
	}

	// Strip ANSI codes
	cleaned := StripANSI(output)

	// Collapse multiple blank lines (3+ -> 2)
	lines := strings.Split(cleaned, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if trimmed == "" {
			blankCount++
			if blankCount <= 1 {
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

	// Smart truncation
	if len(result) > MaxGenericLines {
		keepHead := MaxGenericLines * 2 / 3
		keepTail := MaxGenericLines - keepHead
		omitted := len(result) - keepHead - keepTail

		truncated := make([]string, 0, MaxGenericLines+1)
		truncated = append(truncated, result[:keepHead]...)
		truncated = append(truncated, fmt.Sprintf("\n... [%d lines omitted] ...\n", omitted))
		truncated = append(truncated, result[len(result)-keepTail:]...)
		result = truncated
	}

	return strings.Join(result, "\n")
}
