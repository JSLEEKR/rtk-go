# Target Selection: rtk (Rust Token Killer)

## Project Info
- **Name**: rtk
- **Repo**: https://github.com/rtk-ai/rtk
- **Stars**: 14,639
- **Language**: Rust
- **License**: Apache-2.0
- **Age**: 64 days
- **Trending Signal**: newcomer (score 9.7), momentum (9.7), trend_score 5.8
- **Recent Commits (30d)**: 228

## Why Selected

### Candidate Evaluation

| Candidate | Stars | Signal | Verdict |
|-----------|-------|--------|---------|
| rtk | 14.6K | newcomer 9.7 | SELECTED -- pure CLI, algorithmic core, learnable |
| dagu | 3.2K | momentum 5.0 | REJECTED -- overlap with flowrun (shipped #43) |
| cc-switch | 34K | momentum 9.7 | REJECTED -- UI/desktop heavy (Tauri app) |
| context7 | 50K | - | REJECTED -- too large (50K+) |
| voicebox | 14K | - | REJECTED -- cloud/AI model dependent, frontend heavy |

### Why rtk?

1. **Clear algorithmic core**: The tool intercepts CLI command output and applies format-specific parsers + compressors to reduce token count by 60-90%. This is pure data transformation -- no cloud dependencies, no UI.

2. **Interesting to learn**: Token reduction strategies for different command outputs (git diff, test runners, file listings, grep results) each require specialized parsing. Understanding what information LLMs actually need vs. noise is a valuable insight.

3. **Language crossover**: Original is Rust. Reimplementing in Go gives us a chance to compare ergonomics for CLI proxy tools. Go's string processing and subprocess handling are well-suited.

4. **Right size**: Core logic is ~15 command-specific filters + a proxy mechanism. Achievable in 1 day.

5. **High trending signal**: newcomer_score 9.7, momentum_score 9.7 -- this is genuinely hot right now in the AI coding tool ecosystem.

## Core Features to Reimplement (5)

1. **CLI Proxy Engine**: Intercept command execution, capture stdout/stderr, apply filters, return compressed output. The core `exec -> filter -> output` pipeline.

2. **Command-Specific Filters** (top 8):
   - `ls` / `tree` -- strip metadata, compress directory listings
   - `cat` / file read -- intelligent truncation, skip binary detection
   - `grep` / `rg` -- deduplicate paths, collapse repeated matches
   - `git status` -- strip noise, keep only changed files
   - `git diff` -- compress hunks, strip redundant context lines
   - `git log` -- summarize, strip decoration
   - `npm test` / `cargo test` / `go test` -- extract pass/fail summary, collapse verbose output
   - `docker ps` -- tabular compression

3. **Smart Truncation**: When output exceeds a threshold, intelligently truncate from the middle (preserving head/tail) rather than cutting at the end.

4. **Token Counting / Reporting**: Track tokens saved per command, show summary statistics. Helps users understand the value.

5. **Configuration System**: Per-command filter settings, custom rules, output length limits. YAML/TOML config file support.

## Features to Skip

- Rust-specific build system integration
- Shell integration / PATH hijacking mechanism (complex OS-level setup)
- Plugin/extension system
- Real-time streaming filters
- IDE integrations

## Implementation Plan

- **Language**: Go
- **Reimplementation Name**: `tokenshrink` (or similar -- avoid name collision with shipped `tokencost`)
- **Estimated Difficulty**: Medium
- **Estimated Time**: 1 day
- **Test Target**: 200+ tests

## Architecture Sketch

```
CLI Input (e.g., "tokenshrink git diff")
  |
  v
Command Parser -- identify which filter to apply
  |
  v
Subprocess Executor -- run actual command, capture output
  |
  v
Filter Registry -- lookup filter by command name
  |
  v
Command-Specific Filter -- parse + compress output
  |
  v
Smart Truncator -- enforce max length with intelligent cuts
  |
  v
Token Counter -- track savings statistics
  |
  v
Compressed Output -- to stdout
```
