// Package truncate provides smart truncation that preserves head and tail of output.
package truncate

import (
	"fmt"
	"strings"
)

// DefaultMaxLines is the default maximum number of lines to keep.
const DefaultMaxLines = 200

// Smart truncates output by keeping the first and last portions,
// removing the middle. This preserves the most useful context for LLMs
// (headers/setup at top, results/errors at bottom).
func Smart(input string, maxLines int) string {
	if maxLines <= 0 {
		maxLines = DefaultMaxLines
	}

	lines := strings.Split(input, "\n")
	if len(lines) <= maxLines {
		return input
	}

	keepHead := maxLines * 2 / 3 // 2/3 from top
	keepTail := maxLines - keepHead
	if keepTail < 1 {
		keepTail = 1
		keepHead = maxLines - 1
	}

	omitted := len(lines) - keepHead - keepTail

	var b strings.Builder
	for i := 0; i < keepHead; i++ {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	b.WriteString(fmt.Sprintf("\n... [%d lines omitted] ...\n\n", omitted))
	for i := len(lines) - keepTail; i < len(lines); i++ {
		b.WriteString(lines[i])
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// Head keeps only the first maxLines lines.
func Head(input string, maxLines int) string {
	if maxLines <= 0 {
		return input
	}
	lines := strings.Split(input, "\n")
	if len(lines) <= maxLines {
		return input
	}
	omitted := len(lines) - maxLines
	result := strings.Join(lines[:maxLines], "\n")
	return result + fmt.Sprintf("\n... [%d more lines]", omitted)
}

// Tail keeps only the last maxLines lines.
func Tail(input string, maxLines int) string {
	if maxLines <= 0 {
		return input
	}
	lines := strings.Split(input, "\n")
	if len(lines) <= maxLines {
		return input
	}
	omitted := len(lines) - maxLines
	result := strings.Join(lines[len(lines)-maxLines:], "\n")
	return fmt.Sprintf("[%d lines above] ...\n", omitted) + result
}
