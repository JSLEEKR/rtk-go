package filter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// MaxFailures is the maximum number of test failures to display.
const MaxFailures = 10

// --- Go Test Filter ---

// GoTestFilter parses go test JSON output (go test -json) and
// standard verbose output, showing only failures and summary.
type GoTestFilter struct{}

func (f *GoTestFilter) Name() string { return "go-test" }

func (f *GoTestFilter) Match(cmd string, args []string) bool {
	if cmd != "go" || len(args) == 0 {
		return false
	}
	return args[0] == "test"
}

// goTestEvent represents a single go test JSON event.
type goTestEvent struct {
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
	Output  string `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

func (f *GoTestFilter) Apply(output string, exitCode int) string {
	if output == "" {
		return "no test output"
	}

	// Try JSON parsing first (go test -json)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "{") {
		return f.parseJSON(lines, exitCode)
	}

	return f.parseVerbose(lines, exitCode)
}

func (f *GoTestFilter) parseJSON(lines []string, exitCode int) string {
	passed := 0
	failed := 0
	skipped := 0
	var failures []string
	failureOutput := make(map[string][]string) // test name -> output lines

	for _, line := range lines {
		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		switch event.Action {
		case "pass":
			if event.Test != "" {
				passed++
			}
		case "fail":
			if event.Test != "" {
				failed++
				key := event.Package + "/" + event.Test
				failures = append(failures, key)
			}
		case "skip":
			if event.Test != "" {
				skipped++
			}
		case "output":
			if event.Test != "" {
				key := event.Package + "/" + event.Test
				failureOutput[key] = append(failureOutput[key], strings.TrimRight(event.Output, "\n"))
			}
		}
	}

	var b strings.Builder
	total := passed + failed + skipped

	if failed > 0 {
		b.WriteString(fmt.Sprintf("FAIL: %d/%d tests failed", failed, total))
		if skipped > 0 {
			b.WriteString(fmt.Sprintf(", %d skipped", skipped))
		}
		b.WriteByte('\n')

		shown := 0
		for _, name := range failures {
			if shown >= MaxFailures {
				b.WriteString(fmt.Sprintf("\n... +%d more failures", len(failures)-MaxFailures))
				break
			}
			b.WriteString(fmt.Sprintf("\n--- FAIL: %s\n", name))
			if output, ok := failureOutput[name]; ok {
				for _, line := range output {
					if line != "" {
						b.WriteString(fmt.Sprintf("    %s\n", line))
					}
				}
			}
			shown++
		}
	} else {
		b.WriteString(fmt.Sprintf("PASS: %d tests passed", passed))
		if skipped > 0 {
			b.WriteString(fmt.Sprintf(", %d skipped", skipped))
		}
	}

	return b.String()
}

func (f *GoTestFilter) parseVerbose(lines []string, exitCode int) string {
	passed := 0
	failed := 0
	var failures []string
	var currentFailure []string
	inFailure := false

	passRegex := regexp.MustCompile(`^--- PASS:`)
	failRegex := regexp.MustCompile(`^--- FAIL: (.+)`)
	okRegex := regexp.MustCompile(`^ok\s+`)
	failPkgRegex := regexp.MustCompile(`^FAIL\s+`)

	for _, line := range lines {
		if passRegex.MatchString(line) {
			passed++
			inFailure = false
			continue
		}

		if m := failRegex.FindStringSubmatch(line); m != nil {
			if inFailure && len(currentFailure) > 0 {
				failures = append(failures, strings.Join(currentFailure, "\n"))
			}
			failed++
			inFailure = true
			currentFailure = []string{line}
			continue
		}

		if inFailure {
			currentFailure = append(currentFailure, line)
		}

		if okRegex.MatchString(line) || failPkgRegex.MatchString(line) {
			if inFailure && len(currentFailure) > 0 {
				failures = append(failures, strings.Join(currentFailure, "\n"))
				inFailure = false
				currentFailure = nil
			}
		}
	}

	if inFailure && len(currentFailure) > 0 {
		failures = append(failures, strings.Join(currentFailure, "\n"))
	}

	var b strings.Builder
	total := passed + failed

	if failed > 0 {
		b.WriteString(fmt.Sprintf("FAIL: %d/%d tests failed\n", failed, total))
		for i, f := range failures {
			if i >= MaxFailures {
				b.WriteString(fmt.Sprintf("\n... +%d more failures", len(failures)-MaxFailures))
				break
			}
			b.WriteString(f)
			b.WriteByte('\n')
		}
	} else if exitCode != 0 {
		// Build error or compilation failure
		return output(lines)
	} else {
		b.WriteString(fmt.Sprintf("PASS: %d tests passed", passed))
	}

	return b.String()
}

func output(lines []string) string {
	return strings.Join(lines, "\n")
}

// --- Pytest Filter ---

// PytestFilter parses pytest output using a state machine,
// hiding passing tests and showing only failures with context.
type PytestFilter struct{}

func (f *PytestFilter) Name() string { return "pytest" }

func (f *PytestFilter) Match(cmd string, args []string) bool {
	if cmd == "pytest" || cmd == "python" || cmd == "python3" || cmd == "py" {
		for _, arg := range args {
			if strings.Contains(arg, "pytest") || arg == "-m" {
				return true
			}
		}
		if cmd == "pytest" {
			return true
		}
	}
	return false
}

func (f *PytestFilter) Apply(output string, exitCode int) string {
	if output == "" {
		return "no test output"
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	var failures []string
	var currentFailure []string
	var summaryLine string
	inFailure := false

	failSectionRegex := regexp.MustCompile(`^={3,} FAILURES ={3,}$`)
	testHeaderRegex := regexp.MustCompile(`^_{3,} (.+) _{3,}$`)
	summaryRegex := regexp.MustCompile(`^={3,} .*(passed|failed|error).* ={3,}$`)
	shortSummaryRegex := regexp.MustCompile(`^={3,} short test summary`)

	for _, line := range lines {
		if summaryRegex.MatchString(line) {
			summaryLine = line
			if inFailure && len(currentFailure) > 0 {
				failures = append(failures, strings.Join(currentFailure, "\n"))
			}
			inFailure = false
			continue
		}

		if shortSummaryRegex.MatchString(line) {
			if inFailure && len(currentFailure) > 0 {
				failures = append(failures, strings.Join(currentFailure, "\n"))
			}
			inFailure = false
			continue
		}

		if failSectionRegex.MatchString(line) {
			inFailure = true
			continue
		}

		if testHeaderRegex.MatchString(line) && inFailure {
			if len(currentFailure) > 0 {
				failures = append(failures, strings.Join(currentFailure, "\n"))
			}
			currentFailure = []string{line}
			continue
		}

		if inFailure {
			currentFailure = append(currentFailure, line)
		}
	}

	if inFailure && len(currentFailure) > 0 {
		failures = append(failures, strings.Join(currentFailure, "\n"))
	}

	var b strings.Builder

	if summaryLine != "" {
		b.WriteString(summaryLine)
		b.WriteByte('\n')
	}

	if len(failures) > 0 {
		b.WriteString(fmt.Sprintf("\n%d failure(s):\n", len(failures)))
		for i, f := range failures {
			if i >= MaxFailures {
				b.WriteString(fmt.Sprintf("\n... +%d more failures", len(failures)-MaxFailures))
				break
			}
			b.WriteString(f)
			b.WriteString("\n\n")
		}
	} else if exitCode == 0 {
		if summaryLine == "" {
			b.WriteString("all tests passed")
		}
	} else {
		// Error state, return more of the output
		return strings.Join(lines[max(0, len(lines)-20):], "\n")
	}

	return b.String()
}

// --- NPM Test Filter ---

// NPMTestFilter parses npm test / jest / vitest output.
type NPMTestFilter struct{}

func (f *NPMTestFilter) Name() string { return "npm-test" }

func (f *NPMTestFilter) Match(cmd string, args []string) bool {
	if cmd == "npm" || cmd == "npx" || cmd == "pnpm" || cmd == "yarn" {
		for _, arg := range args {
			if arg == "test" || arg == "vitest" || arg == "jest" {
				return true
			}
		}
	}
	if cmd == "jest" || cmd == "vitest" {
		return true
	}
	return false
}

var jestSummaryRegex = regexp.MustCompile(`(?i)(Tests?|Test Suites?):.*\d+\s+(passed|failed)`)
var jestFailRegex = regexp.MustCompile(`(?i)●|FAIL\s+`)

func (f *NPMTestFilter) Apply(output string, exitCode int) string {
	if output == "" {
		return "no test output"
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	var summaryLines []string
	var failLines []string
	inFail := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Capture summary lines
		if jestSummaryRegex.MatchString(trimmed) {
			summaryLines = append(summaryLines, trimmed)
			continue
		}

		// Capture failure blocks
		if jestFailRegex.MatchString(trimmed) {
			inFail = true
		}

		if inFail {
			failLines = append(failLines, line)
			if trimmed == "" && len(failLines) > 1 {
				inFail = false
			}
		}
	}

	var b strings.Builder

	if len(summaryLines) > 0 {
		for _, s := range summaryLines {
			b.WriteString(s)
			b.WriteByte('\n')
		}
	}

	if len(failLines) > 0 {
		b.WriteString("\nFailures:\n")
		limit := len(failLines)
		if limit > 50 {
			limit = 50
		}
		for i := 0; i < limit; i++ {
			b.WriteString(failLines[i])
			b.WriteByte('\n')
		}
		if len(failLines) > 50 {
			b.WriteString(fmt.Sprintf("... +%d more lines\n", len(failLines)-50))
		}
	}

	result := b.String()
	if result == "" {
		if exitCode == 0 {
			return "all tests passed"
		}
		// Return last 30 lines for error context
		start := len(lines) - 30
		if start < 0 {
			start = 0
		}
		return strings.Join(lines[start:], "\n")
	}

	return result
}
