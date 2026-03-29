package filter

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	filters := r.Filters()
	if len(filters) == 0 {
		t.Fatal("Registry should have filters")
	}
	// Last filter should be generic (catch-all)
	last := filters[len(filters)-1]
	if last.Name() != "generic" {
		t.Errorf("Last filter should be generic, got: %q", last.Name())
	}
}

func TestRegistryLookupGitStatus(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("git", []string{"status"})
	if f.Name() != "git-status" {
		t.Errorf("Expected git-status, got: %q", f.Name())
	}
}

func TestRegistryLookupGitDiff(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("git", []string{"diff"})
	if f.Name() != "git-diff" {
		t.Errorf("Expected git-diff, got: %q", f.Name())
	}
}

func TestRegistryLookupGitLog(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("git", []string{"log"})
	if f.Name() != "git-log" {
		t.Errorf("Expected git-log, got: %q", f.Name())
	}
}

func TestRegistryLookupGrep(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("grep", []string{"-r", "TODO"})
	if f.Name() != "grep" {
		t.Errorf("Expected grep, got: %q", f.Name())
	}
}

func TestRegistryLookupRg(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("rg", []string{"TODO"})
	if f.Name() != "grep" {
		t.Errorf("Expected grep, got: %q", f.Name())
	}
}

func TestRegistryLookupFind(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("find", []string{".", "-name", "*.go"})
	if f.Name() != "find" {
		t.Errorf("Expected find, got: %q", f.Name())
	}
}

func TestRegistryLookupLS(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("ls", []string{"-la"})
	if f.Name() != "ls" {
		t.Errorf("Expected ls, got: %q", f.Name())
	}
}

func TestRegistryLookupGoTest(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("go", []string{"test", "./..."})
	if f.Name() != "go-test" {
		t.Errorf("Expected go-test, got: %q", f.Name())
	}
}

func TestRegistryLookupPytest(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("pytest", []string{"-v"})
	if f.Name() != "pytest" {
		t.Errorf("Expected pytest, got: %q", f.Name())
	}
}

func TestRegistryLookupNPMTest(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("npm", []string{"test"})
	if f.Name() != "npm-test" {
		t.Errorf("Expected npm-test, got: %q", f.Name())
	}
}

func TestRegistryLookupGoBuild(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("go", []string{"build", "./..."})
	if f.Name() != "build" {
		t.Errorf("Expected build, got: %q", f.Name())
	}
}

func TestRegistryLookupUnknown(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("curl", []string{"https://example.com"})
	if f.Name() != "generic" {
		t.Errorf("Unknown command should fall back to generic, got: %q", f.Name())
	}
}

func TestRegistryLookupEmptyArgs(t *testing.T) {
	r := NewRegistry()
	f := r.Lookup("git", []string{})
	if f.Name() != "generic" {
		t.Errorf("Git with no args should fall back to generic, got: %q", f.Name())
	}
}

func TestRegistryAllFilterNames(t *testing.T) {
	r := NewRegistry()
	expectedNames := map[string]bool{
		"git-status": true,
		"git-diff":   true,
		"git-log":    true,
		"grep":       true,
		"find":       true,
		"ls":         true,
		"go-test":    true,
		"pytest":     true,
		"npm-test":   true,
		"build":      true,
		"generic":    true,
	}

	for _, f := range r.Filters() {
		if !expectedNames[f.Name()] {
			t.Errorf("Unexpected filter name: %q", f.Name())
		}
		delete(expectedNames, f.Name())
	}

	for name := range expectedNames {
		t.Errorf("Missing filter: %q", name)
	}
}
