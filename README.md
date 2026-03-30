# rtk-go

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-156-success?style=for-the-badge)](https://github.com/JSLEEKR/rtk-go)
[![Zero Deps](https://img.shields.io/badge/Dependencies-0_external-brightgreen?style=for-the-badge)](go.mod)

**CLI proxy that reduces LLM token consumption by filtering command output.**

Inspired by [rtk](https://github.com/rtk-ai/rtk) (14.6K stars, Rust). Reimplemented from scratch in Go with a unified filter interface, no shell injection vulnerability, and zero external dependencies.

---

## Why This Exists

LLM-powered coding tools (Claude Code, GitHub Copilot, Cursor) read CLI output to understand your project. But most CLI tools produce verbose, human-optimized output: ANSI colors, progress bars, passing test output, compilation status lines, git hints. LLMs don't need any of that noise.

**rtk-go sits between your commands and your LLM**, compressing output by 60-99% while preserving the information that actually matters for decision-making.

```
Before:  git status → 200 tokens (headers, hints, whitespace)
After:   rtk-go git status → 10 tokens ("[main] 3 modified, 1 untracked")
Savings: 95%
```

### Why Not Use the Original rtk?

| Issue | rtk (Rust) | rtk-go |
|-------|-----------|--------|
| Architecture | 72 separate modules, no shared abstraction | Unified `Filter` interface, 11 implementations |
| Shell Safety | `sh -c` with user input ([#640](https://github.com/rtk-ai/rtk/issues/640)) | `exec.Command` with explicit args — zero injection vectors |
| Dependencies | 22 crates | Zero external deps (Go stdlib only) |
| Streaming | Buffers all output ([#222](https://github.com/rtk-ai/rtk/issues/222)) | Buffered with fail-safe recovery |
| Telemetry | Opt-out device tracking | None |
| Build | Requires Rust toolchain | Single `go build` |

---

## Installation

### From Source

```bash
go install github.com/JSLEEKR/rtk-go/cmd/rtk-go@latest
```

### Build from Repository

```bash
git clone https://github.com/JSLEEKR/rtk-go.git
cd rtk-go
go build -o rtk-go ./cmd/rtk-go/
```

---

## Quick Start

```bash
# Instead of running commands directly, prefix with rtk-go:
rtk-go git status
rtk-go git diff
rtk-go git log
rtk-go grep -r "TODO" .
rtk-go find . -name "*.go"
rtk-go go test ./...
rtk-go go build ./...

# Pass through without filtering:
rtk-go --raw git log

# Show savings report:
rtk-go --report git diff
```

---

## Supported Filters

rtk-go includes 11 filters covering the most common CLI tools used in development:

### Git Filters

#### `git status` — 90-99% reduction

Parses verbose git status into a compact summary with file counts by change type.

```
Input (200 tokens):
  On branch main
  Your branch is up to date with 'origin/main'.

  Changes to be committed:
    (use "git restore --staged <file>..." to unstage)
          new file:   README.md

  Changes not staged for commit:
    (use "git add <file>..." to update what will be committed)
          modified:   src/main.go
          modified:   src/util.go

  Untracked files:
    (use "git add <file>..." to include in what will be committed)
          newfile.txt

Output (15 tokens):
  [main] 1 staged, 2 modified, 1 untracked
  Staged:
    README.md
  Modified:
    src/main.go
    src/util.go
  Untracked:
    newfile.txt
```

#### `git diff` — 85-95% reduction

Limits diff hunks to 100 lines per file with summary statistics and recovery hints.

```
Output:
  diff --git a/main.go b/main.go
  @@ -1,5 +1,6 @@
  [first 100 lines of changes]
  ... [truncated, 100+ lines in main.go]

  --- 3 file(s) changed, +45 -12 (1 file(s) truncated at 100 lines)
```

#### `git log` — 80-90% reduction

Caps at 10 commits, strips trailers (Signed-off-by, Co-authored-by, etc.).

```
Output:
  commit abc123...
  Author: Developer <dev@example.com>

      Add new feature

  ... [showing 10 of 85 commits]
```

### Search Filters

#### `grep` / `rg` — 70-85% reduction

Groups results by file with per-file limits (25) and total limits (200).

```
Output:
  ## src/main.go (3 matches)
    10: func main() {
    25: func helper() {
    40: func process() {

  ## src/util.go (1 matches)
    5: func utility() {

  --- 4 matches in 2 files
```

#### `find` / `fd` — 60-80% reduction

Groups by parent directory with extension summary and budget-aware limiting.

```
Output:
  src/
    main.go
    util.go
  tests/
    main_test.go

  --- 3F 2D: .go(3)
```

#### `ls` — 50-70% reduction

Filters noise directories (node_modules, .git, __pycache__, etc.) and provides item counts.

```
Output:
  src
  tests
  go.mod
  README.md
  --- 4 items (3 noise dirs hidden)
```

### Test Runner Filters

#### `go test` — 90%+ reduction

Parses both JSON (`go test -json`) and verbose output. Hides passing tests, shows only failures.

```
Input (500 tokens):
  === RUN   TestA
  --- PASS: TestA (0.01s)
  === RUN   TestB
  --- PASS: TestB (0.02s)
  === RUN   TestC
      main_test.go:15: expected 1 got 2
  --- FAIL: TestC (0.03s)
  FAIL    myapp   0.5s

Output (30 tokens):
  FAIL: 1/3 tests failed
  --- FAIL: TestC (0.03s)
      main_test.go:15: expected 1 got 2
```

#### `pytest` — 90%+ reduction

State machine parser that extracts failure blocks and summary lines. Hides all passing tests.

```
Output:
  ========================= 2 passed, 1 failed in 0.5s =========================

  1 failure(s):
  _________________________________ test_bad _________________________________
      def test_bad():
  >       assert 1 == 2
  E       assert 1 == 2
```

#### `npm test` / `jest` / `vitest` — 90%+ reduction

Extracts test summary lines and failure blocks from JavaScript test runners.

### Build Filters

#### `go build` / `cargo build` / `make` — 60-80% reduction

Strips compilation progress lines ("Compiling...", "Building..."), keeps only errors and warnings.

```
Input (100 tokens):
  Compiling serde v1.0.0
  Compiling tokio v1.0.0
  Compiling myapp v0.1.0
  error[E0308]: mismatched types
    --> src/main.rs:10:5

Output (20 tokens):
  1 error(s):
  error[E0308]: mismatched types
    --> src/main.rs:10:5
  (3 progress lines hidden)
```

#### Generic Fallback

For unrecognized commands: strips ANSI escape codes, collapses blank lines, smart truncation (preserves head/tail).

---

## Architecture

### Unified Filter Interface

The core design advantage over rtk. Every filter implements the same interface:

```go
type Filter interface {
    Name() string
    Match(cmd string, args []string) bool
    Apply(output string, exitCode int) string
}
```

This replaces rtk's 72 ad-hoc Rust modules with a clean, testable abstraction.

### Data Flow

```
CLI Input (e.g., "rtk-go git diff")
  │
  ▼
Command Parser ─── identify command + args
  │
  ▼
Filter Registry ─── lookup matching filter by command name
  │
  ▼
exec.Command ─── run actual command, capture stdout/stderr
  │                (NO shell interpolation — prevents injection)
  ▼
Filter.Apply() ─── compress output using domain-specific heuristics
  │
  ▼
Token Counter ─── track input/output token counts (chars/4)
  │
  ▼
Compressed Output ─── to stdout (savings report to stderr)
```

### Fail-Safe Design

If any filter panics or errors, rtk-go returns the raw unfiltered output. You never lose data:

```go
func applyFilterSafe(f Filter, output string, exitCode int) (result string) {
    defer func() {
        if r := recover(); r != nil {
            result = output // fail-safe: return raw output
        }
    }()
    return f.Apply(output, exitCode)
}
```

### Security: No Shell Injection

rtk (Rust) has a critical vulnerability ([#640](https://github.com/rtk-ai/rtk/issues/640)): user input flows through `sh -c` enabling shell injection. rtk-go uses `exec.Command` with explicit argument arrays:

```go
// SECURITY: exec.Command with explicit args — NO shell interpolation
cmd := exec.Command(cmdName, args...)
```

### Exit Code Preservation

The underlying command's exit code is always preserved and propagated. This is critical for CI/CD pipelines.

---

## Configuration

rtk-go reads configuration from `~/.config/rtk-go/config.yaml`:

```yaml
# rtk-go configuration
max_lines: 300

filters:
  grep_max_results: 200
  grep_max_per_file: 25
  git_status_max: 15
  git_diff_max_lines: 100
  git_log_max_commits: 10
  find_max_results: 100
  test_max_failures: 10

# Disable specific filters (uses generic fallback instead)
disabled:
  - build
```

### Configuration Options

| Key | Default | Description |
|-----|---------|-------------|
| `max_lines` | 300 | Global max output lines before truncation |
| `grep_max_results` | 200 | Max total grep matches shown |
| `grep_max_per_file` | 25 | Max grep matches per file |
| `git_status_max` | 15 | Max staged/modified files shown |
| `git_diff_max_lines` | 100 | Max diff lines per file section |
| `git_log_max_commits` | 10 | Max commits shown in git log |
| `find_max_results` | 100 | Max files shown in find output |
| `test_max_failures` | 10 | Max test failures shown in detail |

---

## Token Counting

rtk-go uses the `chars / 4` heuristic for token estimation:

```
tokens = ceil(characters / 4)
savings = (input_tokens - output_tokens) / input_tokens × 100%
```

This approximation is accurate to within ~20% for English text and trades precision for zero-overhead measurement. **Note:** Savings percentages shown are estimates based on this heuristic, not exact tokenizer counts. Actual token savings may vary by model and encoding. The savings report is displayed on stderr when `--report` is used:

```
--- rtk-go: git-status | 250→12 tokens (95% saved)
```

Use `--report` for a session summary:

```
=== rtk-go Token Savings Report ===

Commands:      5
Input tokens:  3250
Output tokens: 450
Saved:         2800 (86.2%)

By filter:
  git-status        2 commands,   1800 tokens saved
  git-diff          1 commands,    600 tokens saved
  grep              1 commands,    300 tokens saved
  go-test           1 commands,    100 tokens saved
```

---

## Project Structure

```
rtk-go/
├── cmd/rtk-go/
│   └── main.go              # CLI entry point (stdlib flag)
├── internal/
│   ├── config/
│   │   ├── config.go         # YAML config loading (stdlib only)
│   │   └── config_test.go
│   ├── filter/
│   │   ├── filter.go         # Filter interface + Registry
│   │   ├── filter_test.go    # Registry lookup tests
│   │   ├── git.go            # Git filters (status, diff, log)
│   │   ├── git_test.go
│   │   ├── grep.go           # Grep, find, ls filters
│   │   ├── grep_test.go
│   │   ├── test.go           # Test runner filters (go, pytest, npm)
│   │   ├── test_test.go
│   │   ├── build.go          # Build output filters
│   │   ├── build_test.go
│   │   ├── generic.go        # Fallback filter
│   │   └── generic_test.go
│   ├── proxy/
│   │   ├── proxy.go          # Command execution + filter pipeline
│   │   └── proxy_test.go
│   ├── report/
│   │   ├── report.go         # Token savings reporting
│   │   └── report_test.go
│   ├── token/
│   │   ├── counter.go        # Token counting heuristic
│   │   └── counter_test.go
│   └── truncate/
│       ├── truncate.go       # Smart truncation (head/tail/middle)
│       └── truncate_test.go
├── go.mod
└── README.md
```

---

## Comparison with rtk

| Metric | rtk (Rust) | rtk-go |
|--------|-----------|--------|
| Language | Rust | Go |
| Source files | 72 modules | 11 filter files |
| External deps | 22 crates | 0 |
| Binary size | ~10MB | ~5MB |
| Filter architecture | Ad-hoc per module | Unified interface |
| Shell safety | `sh -c` (injection risk) | `exec.Command` (safe) |
| Telemetry | Device tracking (opt-out) | None |
| Test count | Unknown | 156 |
| Streaming support | Buffered only | Buffered with fail-safe |
| Config format | TOML | YAML (stdlib parser) |
| Windows | Incomplete hooks | First-class support |

### Key Improvements

1. **Unified Filter Interface** — Not 72 separate modules. One `Filter` interface, clean registry lookup. Easy to add new filters.

2. **No Shell Injection** — rtk passes user input through `sh -c`, enabling command injection. rtk-go uses `exec.Command` with explicit argument arrays. No shell involved.

3. **Zero External Dependencies** — Everything uses Go stdlib. The YAML parser is built-in (simple key-value format). No supply chain risk.

4. **Fail-Safe Design** — If any filter panics, raw output is returned unchanged. You never lose data.

5. **Simpler Codebase** — ~2100 lines of source code (excluding tests) vs thousands of lines of Rust across 72 files. Easier to audit, maintain, and contribute to.

---

## Development

```bash
# Run tests
go test ./... -v

# Build
go build -o rtk-go ./cmd/rtk-go/

# Vet
go vet ./...

# Run with verbose output
./rtk-go --report git status
```

---

## License

MIT

---

## Credits

- Inspired by [rtk](https://github.com/rtk-ai/rtk) by Patrick Szymkowiak
- Reimplemented from scratch in Go by [JSLEEKR](https://github.com/JSLEEKR)
- Core insight: "CLI output is a compressible signal. LLMs need structure, not formatting."
