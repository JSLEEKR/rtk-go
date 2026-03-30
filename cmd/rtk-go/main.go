// rtk-go is a CLI proxy that reduces LLM token consumption by filtering command output.
// Inspired by rtk (https://github.com/rtk-ai/rtk). Reimplemented from scratch in Go
// with a unified filter interface, no shell injection, and zero external dependencies.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JSLEEKR/rtk-go/internal/proxy"
)

var (
	version = "1.0.0"

	showVersion  = flag.Bool("version", false, "Print version and exit")
	showHelp     = flag.Bool("help", false, "Print help and exit")
	raw          = flag.Bool("raw", false, "Pass through output without filtering")
	showReport   = flag.Bool("report", false, "Show token savings report after execution")
)

func main() {
	os.Exit(run())
}

func run() int {
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, "rtk-go %s\n", version)
		return 0
	}

	if *showHelp || flag.NArg() == 0 {
		printUsage()
		return 0
	}

	args := flag.Args()
	cmdName := args[0]
	cmdArgs := args[1:]

	p := proxy.New()
	p.Passthrough = *raw
	p.ShowReport = *showReport

	exitCode, err := p.Run(cmdName, cmdArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rtk-go: %s\n", err)
		return 1
	}

	if *showReport {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, p.Reporter.Summary())
	}

	return exitCode
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `rtk-go %s — LLM Token Reduction CLI

USAGE:
    rtk-go [FLAGS] <command> [args...]

EXAMPLES:
    rtk-go git status          Filter git status output
    rtk-go git diff            Compress git diff hunks
    rtk-go grep -r "TODO" .    Group grep results by file
    rtk-go go test ./...       Show only test failures
    rtk-go --raw git log       Pass through unfiltered

FLAGS:
    --version    Print version and exit
    --help       Print this help message
    --raw        Disable filtering (passthrough mode)
    --report     Show token savings report after execution

SUPPORTED FILTERS:
    git status     Compact status summary (90-99%% reduction)
    git diff       Hunk limiting with recovery hints (85-95%%)
    git log        Commit truncation, trailer removal (80-90%%)
    grep/rg        File-grouped results with limits (70-85%%)
    find/fd        Directory-grouped with ext summary (60-80%%)
    ls             Noise directory filtering (50-70%%)
    go test        JSON/verbose failure extraction (90%%+)
    pytest         State machine failure parsing (90%%+)
    npm test       Jest/Vitest summary extraction (90%%+)
    go build       Progress stripping, error focus (60-80%%)
    cargo build    Compilation noise removal (60-80%%)
    (fallback)     ANSI stripping, blank collapse, truncation

SECURITY:
    rtk-go uses exec.Command with explicit argument arrays.
    No shell interpolation. No sh -c. No injection vectors.

CONFIG:
    ~/.config/rtk-go/config.yaml

`, version)
}
