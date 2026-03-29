# RTK (Rust Token Killer) - Deep Study Notes

## Repository

- **URL**: https://github.com/rtk-ai/rtk
- **Version**: 0.34.1
- **Author**: Patrick Szymkowiak
- **License**: MIT
- **Stars**: ~14,639
- **Language**: Rust (Edition 2021)

---

## Architecture

### Directory Structure

```
src/
  main.rs              # CLI parser (clap), command enum, routing
  filter.rs            # Language-aware code filtering (3 levels)
  tracking.rs          # SQLite-based token savings tracking
  config.rs            # TOML config system (~/.config/rtk/)
  runner.rs            # Command execution, output capture
  tee.rs               # Raw output persistence (failures/always/never)
  utils.rs             # Package manager detection, text processing
  tree.rs              # Tree command proxy with noise filtering
  ls.rs                # Compact directory listing
  read.rs              # Smart file reading with compression
  git.rs               # Git subcommand filtering (12 subcommands)
  grep_cmd.rs          # Grouped grep/rg results
  find_cmd.rs          # Compact find results with budget-aware output
  diff_cmd.rs          # Diff compaction with Jaccard similarity
  cargo_cmd.rs         # Cargo build/test/clippy filtering
  go_cmd.rs            # Go test/build/vet filtering (NDJSON)
  golangci_cmd.rs      # golangci-lint JSON grouping
  pytest_cmd.rs        # State machine test parsing
  ruff_cmd.rs          # Python linter (JSON/text dual mode)
  pip_cmd.rs           # pip JSON parsing with uv detection
  lint_cmd.rs          # Generic lint filtering
  npm_cmd.rs           # npm command filtering
  pnpm_cmd.rs          # pnpm command filtering
  vitest_cmd.rs        # Vitest test output filtering
  tsc_cmd.rs           # TypeScript compiler filtering
  next_cmd.rs          # Next.js build filtering
  prettier_cmd.rs      # Prettier output filtering
  playwright_cmd.rs    # Playwright test filtering
  prisma_cmd.rs        # Prisma ORM filtering
  container.rs         # Docker/kubectl filtering
  gh_cmd.rs            # GitHub CLI filtering
  aws_cmd.rs           # AWS CLI filtering
  psql_cmd.rs          # PostgreSQL filtering
  curl_cmd.rs          # curl output filtering
  wget_cmd.rs          # wget output filtering
  rspec_cmd.rs         # RSpec test filtering
  rubocop_cmd.rs       # RuboCop lint filtering
  rake_cmd.rs          # Rake/Rails test filtering
  dotnet_cmd.rs        # .NET build/test filtering
  mypy_cmd.rs          # mypy type checker filtering
  json_cmd.rs          # JSON structure extraction
  wc_cmd.rs            # Word count filtering
  env_cmd.rs           # Environment variable filtering
  format_cmd.rs        # Code formatter filtering
  rewrite_cmd.rs       # Hook rewriting logic
  hook_cmd.rs          # Hook installation/management
  hook_check.rs        # Hook health checking
  hook_audit_cmd.rs    # Hook audit trail
  init.rs              # rtk init for various AI tools
  gain.rs              # Token savings analytics
  session_cmd.rs       # Session adoption stats
  verify_cmd.rs        # Inline test verification
  trust.rs             # TOML filter trust system
  integrity.rs         # Runtime integrity verification
  permissions.rs       # Permission/deny/ask system
  telemetry.rs         # Usage telemetry
  summary.rs           # Output summarization
  display_helpers.rs   # Display formatting helpers
  local_llm.rs         # Local LLM integration
  cc_economics.rs      # Claude Code economics calculations
  ccusage.rs           # Claude Code usage tracking
  binlog.rs            # Binary log handling
  deps.rs              # Dependency tree filtering
  toml_filter.rs       # TOML-based custom filters
  log_cmd.rs           # Log file filtering
  gt_cmd.rs            # Graphite VCS filtering
  discover/            # Optimization opportunity discovery
  filters/             # Additional filter modules
  learn/               # Learning/training modules
  parser/              # Output parser modules
```

**Total: 72 modules** (45 command modules + 22 infrastructure + 5 subdirectories)

### Data Flow (6-Phase Execution)

```
Phase 1: PARSE    -> Clap extracts command + args + global flags
Phase 2: ROUTE    -> Match command enum variant to module
Phase 3: EXECUTE  -> std::process::Command runs actual CLI tool
Phase 4: FILTER   -> Module-specific filtering strategy applied
Phase 5: PRINT    -> Formatted, compressed output to stdout
Phase 6: TRACK    -> Record input/output tokens to SQLite
```

### Fallback Mechanism

If command parsing fails:
1. Try TOML filter lookup (project-local custom filters)
2. Raw passthrough with Stdio inheritance
3. Exit code preservation always

---

## Core Insight

**"This project's key innovation is treating CLI output as a compressible signal."**

Most CLI tools output verbose, human-readable text optimized for terminal reading. LLMs don't need ANSI colors, progress bars, compilation status lines, or passing test output. RTK applies domain-specific compression heuristics per command type, achieving 60-99% token reduction while preserving the information an LLM actually needs to make decisions.

The second key insight is the **hook-based transparent proxy**: the AI tool never knows RTK exists. A PreToolUse hook rewrites `git status` to `rtk git status` before execution, so the LLM sees compressed output without any prompt engineering.

---

## Key Algorithms

### 1. Language-Aware Code Filtering (filter.rs)

Three levels for `rtk read`:

| Level | What It Does | Reduction |
|-------|-------------|-----------|
| None | Pass through unchanged | 0% |
| Minimal | Strip comments, collapse blank lines (3+ -> 2), trim trailing whitespace | 20-40% |
| Aggressive | Keep only imports, signatures, type defs, struct fields. Replace bodies with `// ... implementation` | 60-90% |

Language detection via file extension. Supports: Rust, Python, JS, TS, Go, C, C++, Java.
Data formats (JSON, YAML, TOML, XML, CSV, Markdown) are **never** code-filtered.

Smart truncation (`smart_truncate()`): keeps first half of lines + important structural elements (imports, function signatures, type definitions).

### 2. Git Status Compaction (git.rs)

```
Input:  git status (raw ~200 tokens with hints, headers, whitespace)
Output: "3 modified, 1 untracked" (~10 tokens)
```

- Parses `git status --porcelain -b` for machine-readable format
- Groups by type: staged, modified, untracked, conflicts
- Caps: 15 staged/modified, 10 untracked files shown
- Shows branch with tracking relationship

### 3. Git Diff Compaction (git.rs)

- Limits hunk output to 100 lines per file section
- Tracks added/removed counts separately
- Truncates with recovery hint: `"full diff: rtk git diff --no-compact"`
- Git log: caps to 10 commits, filters trailers (Signed-off-by, Co-authored-by)

### 4. Grep Grouping (grep_cmd.rs)

- Uses `rg` (ripgrep) under the hood, not grep
- Groups results by file into `HashMap<String, Vec<(line_num, content)>>`
- Two limits: `max_results` (total) and `grep_max_per_file` (per file)
- Line cleaning: truncates long lines, context-only mode extracts surrounding pattern

### 5. Test Output Filtering (runner.rs, pytest_cmd.rs)

**State Machine Parsing** (pytest):
```
IDLE -> TEST_START -> PASSED/FAILED
```
- Hides passing tests entirely
- Shows only failures with context
- Extracts summary statistics

**Framework Detection** (runner.rs): auto-detects Cargo, pytest, Jest, Go test
- Limits failure output to 10 items

### 6. Find Result Budget (find_cmd.rs)

- Uses `ignore` crate's WalkBuilder (respects .gitignore)
- Groups by parent directory
- Budget-aware: `max_results` limits files, not directories
- Summary: `"42F 8D: .ts(20) .js(12) .json(5) .md(3) .css(2)"`

### 7. Directory Listing Compaction (ls.rs)

- Filters noise directories: node_modules, .git, target, __pycache__, etc.
- Human-readable sizes: `"1.2M"` instead of `"1234567"`
- Summary: file/directory counts + top 5 extensions

### 8. Tree Noise Filtering (tree.rs)

- Auto-injects `-I` ignore pattern for noise directories (unless `--all`)
- Removes summary lines ("5 directories, 23 files")
- Preserves tree structure characters

### 9. Cargo Clippy Grouping (cargo_cmd.rs)

- Groups warnings by lint rule name
- Sorts by frequency (most common first)
- Shows file:line locations
- Strips compilation noise lines

### 10. Diff Similarity Detection (diff_cmd.rs)

- **Jaccard index** on character sets with 0.5 threshold
- Lines above threshold = "modified" (shows before/after)
- Lines below = separate "added"/"removed"
- Never truncates diff content (design principle)

### 11. Token Counting Heuristic (tracking.rs)

```
tokens = ceil(characters / 4)
savings = (input_tokens - output_tokens) / input_tokens * 100
```

Simple ~4 chars/token approximation. Trades accuracy for zero-overhead measurement.

---

## Command-Specific Filters

| Command | Strategy | Key Technique | Typical Reduction |
|---------|----------|--------------|-------------------|
| `git status` | Stats extraction | Porcelain parsing, type grouping | 90-99% |
| `git diff` | Hunk limiting | 100 lines/file cap, recovery hint | 85-95% |
| `git log` | Truncation | 10 commits, trailer removal, body limit | 80-90% |
| `git add/commit/push` | Metrics extraction | File counts, insertions/deletions | 90%+ |
| `ls` | Noise filtering | Skip noisy dirs, human sizes, summary | 50-70% |
| `tree` | Pattern exclusion | Auto-ignore noise dirs, remove summary | 50-70% |
| `read` | Code filtering | 3-level language-aware reduction | 20-90% |
| `grep` | File grouping | HashMap grouping, per-file limits | 70-85% |
| `find` | Budget limiting | Directory grouping, extension summary | 60-80% |
| `diff` | Similarity | Jaccard index, full changes preserved | 30-50% |
| `cargo test` | Failure focus | Hide passing, show first 10 failures | 94-99% |
| `cargo clippy` | Rule grouping | Group by lint, sort by frequency | 80-90% |
| `cargo build` | Error only | Strip "Compiling" lines, keep errors | 60-80% |
| `go test` | NDJSON streaming | Parse JSON events per package | 90%+ |
| `pytest` | State machine | IDLE->TEST_START->PASS/FAIL tracking | 90%+ |
| `ruff` | Dual mode | JSON structured / text diff modes | 80%+ |
| `eslint/tsc` | Error extraction | Parse diagnostics, group by file | 70-90% |
| `docker ps/logs` | Compaction | Table reformatting, log dedup | 60-80% |
| `curl/wget` | Progress strip | Remove progress bars, keep response | 85-95% |

---

## Dependencies

### Core (Go equivalents needed)

| Rust Crate | Purpose | Go Equivalent |
|-----------|---------|---------------|
| clap 4 | CLI parsing with derive macros | cobra + pflags |
| anyhow 1.0 | Error handling with context | fmt.Errorf with %w |
| regex 1 | Regular expressions | regexp (stdlib) |
| lazy_static 1.4 | Compile-once regex | sync.Once or package-level var |
| serde + serde_json | JSON serialization | encoding/json (stdlib) |
| walkdir 2 | Directory traversal | filepath.WalkDir (stdlib) |
| ignore 0.4 | .gitignore-aware traversal | go-git/go-billy or custom |
| rusqlite 0.31 | SQLite (bundled) | mattn/go-sqlite3 or modernc.org/sqlite |
| colored 2 | Terminal colors | fatih/color |
| dirs 5 | Platform config/data dirs | os.UserConfigDir (stdlib) |
| toml 0.8 | TOML parsing | BurntSushi/toml |
| chrono 0.4 | Date/time | time (stdlib) |
| thiserror 1.0 | Error types | errors package (stdlib) |
| tempfile 3 | Temp files | os.CreateTemp (stdlib) |
| sha2 0.10 | SHA-256 hashing | crypto/sha256 (stdlib) |
| ureq 2 | HTTP requests | net/http (stdlib) |
| hostname 0.4 | Machine hostname | os.Hostname (stdlib) |
| which 8 | PATH binary lookup | exec.LookPath (stdlib) |
| flate2 1.0 | Gzip compression | compress/gzip (stdlib) |
| quick-xml 0.37 | XML parsing | encoding/xml (stdlib) |
| getrandom 0.4 | Random number generation | crypto/rand (stdlib) |

**Key advantage of Go**: many of RTK's 22 dependencies map to Go stdlib, reducing external deps significantly.

---

## Design Decisions

### 1. Single Binary, Zero Runtime Dependencies
Rust's static linking produces a self-contained binary. Go achieves the same natively.

### 2. Exit Code Preservation
Critical for CI/CD. Every module must propagate the underlying command's exit code. This is non-negotiable.

### 3. Fail-Safe: Return Raw Output on Filter Failure
If any filtering logic panics or errors, the original unfiltered output is returned. The user never loses data.

### 4. Proxy Pattern (Not Wrapper)
RTK doesn't modify command behavior -- it only transforms the output. The underlying command runs identically.

### 5. Hook-Based Transparent Integration
The AI tool never knows RTK exists. Hooks rewrite commands before execution. Exit code system:
- 0: Rewrite allowed (auto-permit)
- 1: No RTK equivalent (passthrough)
- 2: Deny rule matched
- 3: Ask rule matched (prompt user)

### 6. Module-Per-Command Architecture
Each CLI tool gets its own .rs file with a standard `run()` interface. Easy to add new commands (7-step process in ARCHITECTURE.md).

### 7. Configurable Limits
Config at `~/.config/rtk/config.toml` with sensible defaults:
- grep: 200 max total, 25 per file
- git status: 15 staged/modified, 10 untracked
- parser passthrough: 2000 char max

### 8. Tee for Recovery
On command failure, raw unfiltered output saved to `~/.local/share/rtk/tee/` so agents can re-read without re-executing. Configurable: Failures/Always/Never mode, max 20 files, 1MB each.

---

## Weaknesses / Improvement Opportunities

### From GitHub Issues

1. **No streaming for long-running commands** (#222) -- RTK buffers entire output before filtering. Long builds appear to hang.

2. **Output reordering bugs** (#187) -- Some filters reorder stderr/stdout lines relative to raw output.

3. **Windows hook support incomplete** (#330, #502) -- Falls back to CLAUDE.md mode on Windows despite hooks being technically possible.

4. **Hook PATH resolution failures** (#685) -- Homebrew installs put rtk in `/opt/homebrew/bin` which Claude Code's restricted PATH doesn't include. Silent failure with zero indication.

5. **find compound predicates unsupported** (#664) -- `rtk find` doesn't support complex find expressions like `-name "*.go" -o -name "*.rs"`.

6. **pnpm --filter breaks** (#259) -- Some flag combinations produce rtk usage output instead of actual results.

7. **cargo clippy -- separator not preserved** (#660) -- Double-dash handling has edge cases.

8. **6 hook bugs on master** (#361) -- Multiple hook rewriting edge cases.

9. **curl filter bypass needed** (#219) -- Some curl uses need unfiltered output.

### From Security Review (#640)

1. **Shell injection via `sh -c`** (CRITICAL) -- Unescaped user input flows to shell execution in runner.rs. Hook auto-approves commands.

2. **Telemetry without consent** -- Device identifiers transmit automatically, no opt-in.

3. **CI trust bypass** -- Environment variables can spoof CI contexts.

4. **Secrets in tracking DB** -- Command arguments stored in SQLite for 90 days may contain secrets.

5. **Path traversal via RTK_TEE_DIR** -- Tee directory can be overridden to arbitrary paths.

### Architectural Weaknesses

1. **72 modules with no shared filtering framework** -- Each command module reimplements similar patterns (regex-based line filtering, output grouping, truncation). No trait/interface for filters.

2. **No plugin system** -- Adding a command requires modifying main.rs and recompiling. No dynamic loading.

3. **Token counting is crude** -- `chars / 4` ignores actual tokenizer behavior. Could be off by 2-3x for non-English text.

4. **No streaming architecture** -- All output buffered in memory. Large `git log` or test outputs could consume significant RAM.

5. **Regex-heavy filtering** -- Many modules use regex for what could be structured parsing (e.g., git porcelain format is already machine-readable).

---

## Implementation Plan for Go Reimplementation

### Phase 1: Core Framework (Week 1)

```
cmd/
  rtk/
    main.go            # cobra root command + subcommand registration
internal/
  runner/
    exec.go            # Command execution with output capture
    exec_test.go
  filter/
    filter.go          # FilterLevel enum, language detection
    code.go            # Language-aware code filtering (minimal/aggressive)
    code_test.go
    truncate.go        # Smart truncation
  tracking/
    tracker.go         # SQLite tracking (modernc.org/sqlite for CGo-free)
    tracker_test.go
  config/
    config.go          # TOML config loading/saving
    config_test.go
  tee/
    tee.go             # Raw output persistence
    tee_test.go
  utils/
    text.go            # ANSI stripping, truncation, formatting
    detect.go          # Package manager detection, tool existence
    text_test.go
```

### Phase 2: Core Commands (Week 2)

Priority order (highest value first):
1. `git` -- status, diff, log, show, add, commit, push (biggest savings)
2. `ls` -- compact directory listing
3. `tree` -- noise-filtered tree
4. `read` -- smart file reading with code filtering
5. `grep` -- grouped search results
6. `find` -- budget-aware file finding
7. `diff` -- Jaccard similarity detection

### Phase 3: Language Ecosystem Commands (Week 3)

1. `go test/build/vet` -- NDJSON parsing, error extraction
2. `cargo test/build/clippy` -- if targeting Rust users
3. `npm/pnpm` -- JS ecosystem
4. `pytest/ruff/pip` -- Python ecosystem

### Phase 4: Infrastructure + Meta Commands (Week 4)

1. `gain` -- token savings analytics
2. `init` -- hook installation for Claude Code / Copilot / etc.
3. `rewrite` -- hook rewriting with exit code system
4. `docker/kubectl` -- container commands
5. `session` -- adoption stats

### Key Go Design Decisions

1. **Interface-based filter system** -- Define a `Filter` interface that all commands implement. Avoid RTK's ad-hoc pattern.

```go
type Filter interface {
    Name() string
    Match(cmd string) bool
    Filter(output []byte, exitCode int, verbose int) ([]byte, error)
}
```

2. **CGo-free SQLite** -- Use `modernc.org/sqlite` for true zero-dependency binary.

3. **Streaming support** -- Use `io.Reader` pipelines where possible to handle large outputs without full buffering. This addresses RTK's #222 issue.

4. **Structured parsing over regex** -- Use Go's `bufio.Scanner` with state machines rather than regex for structured outputs (git porcelain, JSON test output, etc.).

5. **cobra for CLI** -- Natural subcommand support, automatic help generation, familiar to Go ecosystem.

6. **Embed filter configs** -- Use `//go:embed` for default noise directory lists, language patterns, etc.

7. **No telemetry** -- Address RTK's security concern by omitting telemetry entirely.

8. **Safe command execution** -- Use `exec.Command` with explicit args (no shell interpolation) to avoid RTK's shell injection vulnerability.

### Improvements Over RTK

| Area | RTK | Go Version |
|------|-----|------------|
| Streaming | Buffered only | io.Reader pipeline |
| Filter architecture | Ad-hoc per module | Interface-based registry |
| Shell safety | `sh -c` with user input | exec.Command with args array |
| Dependencies | 22 crates | ~3-5 external (most in stdlib) |
| Plugin system | None (recompile) | Optional: plugin or yaegi |
| Token counting | chars/4 | Optional tiktoken integration |
| Windows | Incomplete hooks | First-class support |
| Telemetry | Opt-out | None |
