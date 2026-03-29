# Comparison Report: rtk (Rust) vs rtk-go (Go)

## Overview

| Attribute | rtk (Rust) | rtk-go (Go) |
|-----------|-----------|-------------|
| Repository | github.com/rtk-ai/rtk | github.com/JSLEEKR/rtk-go |
| Stars | ~14,639 | New project |
| Version | 0.34.1 | 1.0.0 |
| Language | Rust (Edition 2021) | Go 1.22+ |
| License | MIT | MIT |
| Author | Patrick Szymkowiak | JSLEEKR |

---

## Feature Comparison

| Feature | rtk (Rust) | rtk-go (Go) | Notes |
|---------|-----------|-------------|-------|
| git status filter | Porcelain parsing | Verbose output parsing | Both achieve 90-99% reduction |
| git diff filter | 100 lines/file, recovery hints | 100 lines/file, recovery hints | Equivalent |
| git log filter | 10 commits, trailer removal | 10 commits, trailer removal | Equivalent |
| grep/rg grouping | HashMap grouping, per-file limits | Map grouping, per-file limits | Equivalent |
| find budget | ignore-crate WalkBuilder | Simple path grouping | rtk respects .gitignore natively |
| ls noise filtering | Size formatting, noise dirs | Noise dirs, item counts | rtk adds human-readable sizes |
| tree filtering | Auto-inject -I patterns | Not implemented | Gap |
| go test parsing | NDJSON streaming | JSON + verbose parsing | Equivalent |
| pytest parsing | State machine | State machine | Equivalent |
| npm/jest/vitest | Separate modules each | Single unified filter | rtk-go simpler |
| cargo build/clippy | Separate clippy grouping | Combined build filter | rtk has richer cargo support |
| Code filtering (read) | 3-level language-aware | Not implemented | Gap |
| Docker/kubectl | Dedicated modules | Not implemented | Gap |
| curl/wget | Progress stripping | Not implemented | Gap |
| GitHub CLI (gh) | Dedicated module | Not implemented | Gap |
| AWS CLI | Dedicated module | Not implemented | Gap |
| PostgreSQL (psql) | Dedicated module | Not implemented | Gap |
| Tee (raw output backup) | Configurable persistence | Not implemented | Gap |
| Hook system | PreToolUse hook rewriting | Not implemented | Gap |
| Telemetry | Opt-out device tracking | None | rtk-go is more privacy-respecting |
| Token tracking | SQLite persistent history | In-memory per-session | Gap |
| Session analytics | Adoption stats, gain reports | Single-session report | Gap |
| Custom filters (TOML) | User-defined TOML filters | Not implemented | Gap |
| Config format | TOML | YAML (stdlib parser) | Equivalent |

### Command Coverage

| Category | rtk | rtk-go |
|----------|-----|--------|
| Git commands | 12 subcommands | 3 (status, diff, log) |
| Search tools | grep, rg | grep, rg, find, ls |
| Test runners | cargo test, go test, pytest, jest, vitest, rspec, dotnet test | go test, pytest, npm/jest/vitest |
| Build tools | cargo, go, make, npm, tsc, next | go, cargo, make, npm, tsc |
| Linters | clippy, ruff, eslint, rubocop, mypy, golangci-lint | (via build filter regex) |
| Infrastructure | docker, kubectl, gh, aws, psql, curl, wget | None |
| **Total modules** | **72** | **11** |

---

## Architecture Comparison

### Filter Design

| Aspect | rtk | rtk-go |
|--------|-----|--------|
| Abstraction | No shared interface; each module has ad-hoc `run()` function | Unified `Filter` interface with `Name()`, `Match()`, `Apply()` |
| Registration | Manual match in main.rs command enum | `Registry` with ordered lookup |
| Fallback | TOML filter lookup, then raw passthrough | `GenericFilter` (ANSI strip, blank collapse, truncation) |
| Adding a filter | Modify main.rs routing + new .rs file | Implement `Filter` interface, add to `NewRegistry()` |

rtk-go's unified interface is a clear architectural improvement. In rtk, there is no shared contract between the 72 modules -- each reimplements similar patterns (regex line filtering, output grouping, truncation) independently.

### Command Execution

| Aspect | rtk | rtk-go |
|--------|-----|--------|
| Execution method | `std::process::Command` but some paths use `sh -c` | `exec.Command` with explicit args only |
| Shell injection | Vulnerable (issue #640) | Not possible by design |
| Exit code handling | Preserved | Preserved |
| Output capture | Full buffer | Full buffer (streaming-ready via io.Reader) |
| Stdin forwarding | Yes | Yes |

### Error Handling

| Aspect | rtk | rtk-go |
|--------|-----|--------|
| Filter failures | Passthrough on error | `recover()` + return raw output |
| Crash safety | Rust's type system prevents most panics | Explicit panic recovery wrapper |
| Missing commands | Passthrough with exit code | Error message + exit code 1 |

---

## Improvements in rtk-go

1. **Unified Filter Interface** -- The single most important architectural improvement. rtk's 72 ad-hoc modules with no shared abstraction make the codebase hard to maintain and extend. rtk-go's `Filter` interface ensures consistency.

2. **Zero Shell Injection Surface** -- rtk has a known critical vulnerability (issue #640) where user input flows through `sh -c`. rtk-go uses `exec.Command` with explicit argument arrays exclusively. There is no code path that invokes a shell.

3. **Zero External Dependencies** -- rtk depends on 22 Rust crates. rtk-go uses only Go stdlib. This eliminates supply chain risk entirely. The YAML parser is hand-written (simple key-value format) rather than importing an external library.

4. **No Telemetry** -- rtk includes device tracking that transmits automatically with opt-out. rtk-go has no telemetry of any kind.

5. **Simpler Codebase** -- ~1,500 lines of Go across 11 filter files vs thousands of lines across 72 Rust files. The codebase is auditable in a single sitting.

6. **Explicit Panic Recovery** -- The `applyFilterSafe()` wrapper ensures that even if a filter panics, raw output is returned. This is a defense-in-depth measure that rtk lacks.

---

## Weaknesses of rtk-go

1. **Command Coverage Gap** -- rtk supports 45+ CLI tools; rtk-go supports ~15. Missing: docker, kubectl, gh, aws, psql, curl, wget, rspec, rubocop, mypy, golangci-lint, prisma, playwright, and more.

2. **No Hook System** -- rtk's killer feature is transparent integration via PreToolUse hooks that rewrite commands before the LLM sees them. rtk-go requires the user to manually prefix commands. This is a significant UX gap.

3. **No Code Filtering** -- rtk's `rtk read` command provides 3-level language-aware code compression (none/minimal/aggressive). rtk-go has no equivalent.

4. **No Persistent Tracking** -- rtk stores token savings history in SQLite for analytics over time. rtk-go only tracks per-session.

5. **No Tee Recovery** -- rtk saves raw unfiltered output to disk on failures so the LLM can re-read without re-executing. rtk-go does not.

6. **No Custom Filters** -- rtk supports user-defined TOML filters for project-specific commands. rtk-go requires code changes.

7. **No .gitignore Awareness** -- rtk's find uses the `ignore` crate to respect .gitignore. rtk-go's find filter just groups raw output.

8. **Git Status Parsing** -- rtk uses `git status --porcelain -b` for machine-readable output. rtk-go parses the verbose human-readable format, which is more fragile.

---

## What We Learned

### 1. Interface-Based Design Pays Off
rtk's lack of a shared abstraction across 72 modules is its biggest architectural weakness. The Go reimplementation proves that a unified `Filter` interface with `Name()`, `Match()`, and `Apply()` can cover all the same use cases with far less code. This is a textbook example of the Interface Segregation Principle.

### 2. Security as Architecture, Not Afterthought
rtk's shell injection vulnerability (issue #640) exists because `sh -c` was the easy path for command execution. By making `exec.Command` with explicit args the only execution path, rtk-go eliminates an entire class of vulnerabilities by design. Security constraints should shape architecture, not be bolted on later.

### 3. Zero-Dep Go is Viable for CLI Tools
All 22 of rtk's Rust dependencies map to Go stdlib equivalents. The only sacrifice is SQLite persistence (which could use `modernc.org/sqlite` for CGo-free). For CLI tools that prioritize simplicity and auditability, zero external dependencies is achievable and valuable.

### 4. 80/20 Rule for Filters
rtk's 72 modules provide diminishing returns. The top 10 filters (git, grep, find, test runners, build) cover 95%+ of real usage. The remaining 62 modules (prisma, playwright, aws, etc.) serve niche use cases. rtk-go's focused approach trades breadth for maintainability.

### 5. The Hook System is the Real Product
rtk's most valuable feature isn't any single filter -- it's the transparent hook system that rewrites commands before the LLM sees them. Without hooks, users must manually prefix every command with `rtk`. This is the biggest gap in rtk-go and the most important feature to add next.

### 6. Token Counting is Good Enough
Both projects use `chars/4` as a token estimation heuristic. Despite being inaccurate for non-English text, it's sufficient for the core value proposition: showing users that filtering saves tokens. Perfect accuracy would require importing a tokenizer library, breaking the zero-dep constraint.

### 7. Custom YAML Parser Risks
rtk-go's hand-written YAML parser avoids external dependencies but introduces bugs (the disabled list parsing issue found during this evaluation). For production use, a well-tested YAML library would be safer. The trade-off between zero deps and correctness is real.

---

## Summary Verdict

rtk-go successfully demonstrates that rtk's core value proposition (CLI output compression for LLMs) can be delivered with a cleaner architecture, better security, and zero dependencies. It covers the highest-value filters and proves the unified interface design.

However, rtk-go is a proof-of-concept compared to rtk's production maturity. The missing hook system, limited command coverage, and lack of persistent tracking mean it is not yet a drop-in replacement. The next priorities should be: (1) hook-based transparent integration, (2) `rtk read` equivalent, and (3) expanding filter coverage to docker/kubectl/gh.
