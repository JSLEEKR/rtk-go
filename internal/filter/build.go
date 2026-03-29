package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// BuildFilter compresses build output (go build, cargo build, make, etc.)
// by stripping compilation progress lines and keeping only errors/warnings.
type BuildFilter struct{}

func (f *BuildFilter) Name() string { return "build" }

func (f *BuildFilter) Match(cmd string, args []string) bool {
	// go build / go vet
	if cmd == "go" && len(args) > 0 {
		return args[0] == "build" || args[0] == "vet" || args[0] == "install"
	}
	// cargo build / cargo clippy
	if cmd == "cargo" && len(args) > 0 {
		return args[0] == "build" || args[0] == "clippy" || args[0] == "check"
	}
	// make
	if cmd == "make" || cmd == "cmake" {
		return true
	}
	// npm/pnpm build
	if (cmd == "npm" || cmd == "pnpm" || cmd == "yarn") && len(args) > 0 {
		return args[0] == "run" && len(args) > 1 && args[1] == "build"
	}
	// tsc
	if cmd == "tsc" {
		return true
	}
	if cmd == "npx" {
		for _, arg := range args {
			if arg == "tsc" {
				return true
			}
		}
	}
	return false
}

var (
	// Lines to skip (compilation progress, not useful for LLMs)
	compilingRegex = regexp.MustCompile(`(?i)^\s*(Compiling|Downloading|Downloaded|Building|Linking|Running|Finished|Updating|Packaging|Verifying|Archiving)\s`)
	makeEnterRegex = regexp.MustCompile(`^make\[\d+\]: (Entering|Leaving) directory`)
	npmTimingRegex = regexp.MustCompile(`^npm\s+(timing|http)\s`)

	// Lines to keep (errors, warnings)
	errorRegex   = regexp.MustCompile(`(?i)(error|^E\d{4}|panic|fatal|undefined|cannot find|not found|failed)`)
	warningRegex = regexp.MustCompile(`(?i)(warning|^W\d{4}|deprecated|unused)`)
)

func (f *BuildFilter) Apply(output string, exitCode int) string {
	if output == "" {
		if exitCode == 0 {
			return "build succeeded"
		}
		return "build failed (no output)"
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	var errors []string
	var warnings []string
	var other []string
	skippedLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Skip noise lines
		if compilingRegex.MatchString(trimmed) ||
			makeEnterRegex.MatchString(trimmed) ||
			npmTimingRegex.MatchString(trimmed) {
			skippedLines++
			continue
		}

		if errorRegex.MatchString(trimmed) {
			errors = append(errors, line)
		} else if warningRegex.MatchString(trimmed) {
			warnings = append(warnings, line)
		} else {
			other = append(other, line)
		}
	}

	var b strings.Builder

	if exitCode == 0 && len(errors) == 0 {
		b.WriteString("build succeeded")
		if len(warnings) > 0 {
			b.WriteString(fmt.Sprintf(" with %d warning(s)", len(warnings)))
			b.WriteByte('\n')
			limit := len(warnings)
			if limit > 20 {
				limit = 20
			}
			for i := 0; i < limit; i++ {
				b.WriteString(warnings[i])
				b.WriteByte('\n')
			}
			if len(warnings) > 20 {
				b.WriteString(fmt.Sprintf("... +%d more warnings\n", len(warnings)-20))
			}
		}
		if skippedLines > 0 {
			b.WriteString(fmt.Sprintf(" (%d progress lines hidden)", skippedLines))
		}
		return b.String()
	}

	// Build failed or has errors
	if len(errors) > 0 {
		b.WriteString(fmt.Sprintf("%d error(s):\n", len(errors)))
		limit := len(errors)
		if limit > 30 {
			limit = 30
		}
		for i := 0; i < limit; i++ {
			b.WriteString(errors[i])
			b.WriteByte('\n')
		}
		if len(errors) > 30 {
			b.WriteString(fmt.Sprintf("... +%d more errors\n", len(errors)-30))
		}
	}

	if len(warnings) > 0 {
		b.WriteString(fmt.Sprintf("\n%d warning(s):\n", len(warnings)))
		limit := len(warnings)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			b.WriteString(warnings[i])
			b.WriteByte('\n')
		}
	}

	// Include some context lines for errors
	if len(errors) == 0 && len(other) > 0 {
		// No clear errors found, show tail of output
		start := len(other) - 20
		if start < 0 {
			start = 0
		}
		for _, line := range other[start:] {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	if skippedLines > 0 {
		b.WriteString(fmt.Sprintf("(%d progress lines hidden)", skippedLines))
	}

	return b.String()
}
