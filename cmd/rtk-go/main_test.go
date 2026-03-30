package main

import (
	"os"
	"testing"
)

func TestRunVersion(t *testing.T) {
	// Set flags for version mode
	os.Args = []string{"rtk-go", "--version"}
	// Reset flags for the test
	*showVersion = true
	*showHelp = false
	*raw = false
	*showReport = false

	code := run()
	if code != 0 {
		t.Errorf("run() with --version should return 0, got %d", code)
	}

	// Reset
	*showVersion = false
}

func TestRunHelp(t *testing.T) {
	*showHelp = true
	*showVersion = false
	*raw = false
	*showReport = false

	code := run()
	if code != 0 {
		t.Errorf("run() with --help should return 0, got %d", code)
	}

	// Reset
	*showHelp = false
}
