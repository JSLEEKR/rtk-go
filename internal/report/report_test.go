package report

import (
	"strings"
	"testing"
)

func TestNewReporter(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
}

func TestRecordAndEntries(t *testing.T) {
	r := New()
	r.Record("git-status", 1000, 50)
	r.Record("grep", 500, 100)

	entries := r.Entries()
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	if entries[0].FilterName != "git-status" {
		t.Errorf("First entry name = %q, want git-status", entries[0].FilterName)
	}
	if entries[0].InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000", entries[0].InputTokens)
	}
	if entries[0].OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", entries[0].OutputTokens)
	}
}

func TestTotalSaved(t *testing.T) {
	r := New()
	r.Record("a", 1000, 100) // saved 900
	r.Record("b", 500, 200)  // saved 300

	if r.TotalSaved() != 1200 {
		t.Errorf("TotalSaved() = %d, want 1200", r.TotalSaved())
	}
}

func TestTotalSavedEmpty(t *testing.T) {
	r := New()
	if r.TotalSaved() != 0 {
		t.Errorf("TotalSaved() on empty = %d, want 0", r.TotalSaved())
	}
}

func TestSummaryEmpty(t *testing.T) {
	r := New()
	got := r.Summary()
	if got != "no commands recorded" {
		t.Errorf("Expected 'no commands recorded', got: %q", got)
	}
}

func TestSummaryWithEntries(t *testing.T) {
	r := New()
	r.Record("git-status", 1000, 50)
	r.Record("grep", 500, 100)
	r.Record("git-status", 800, 40)

	got := r.Summary()
	if !strings.Contains(got, "rtk-go Token Savings Report") {
		t.Error("Should contain title")
	}
	if !strings.Contains(got, "Commands:      3") {
		t.Errorf("Should show 3 commands, got: %q", got)
	}
	if !strings.Contains(got, "Input tokens:  2300") {
		t.Errorf("Should show total input, got: %q", got)
	}
	if !strings.Contains(got, "Output tokens: 190") {
		t.Errorf("Should show total output, got: %q", got)
	}
	if !strings.Contains(got, "git-status") {
		t.Error("Should list git-status filter")
	}
	if !strings.Contains(got, "grep") {
		t.Error("Should list grep filter")
	}
}

func TestEntrySavingsPercent(t *testing.T) {
	e := Entry{FilterName: "test", InputTokens: 1000, OutputTokens: 100}
	if e.SavingsPercent() != 90 {
		t.Errorf("SavingsPercent() = %f, want 90", e.SavingsPercent())
	}
}

func TestEntrySavingsPercentZeroInput(t *testing.T) {
	e := Entry{FilterName: "test", InputTokens: 0, OutputTokens: 0}
	if e.SavingsPercent() != 0 {
		t.Errorf("SavingsPercent() with zero input = %f, want 0", e.SavingsPercent())
	}
}

func TestSummarySortsBySavings(t *testing.T) {
	r := New()
	r.Record("low-savings", 100, 90)   // saved 10
	r.Record("high-savings", 1000, 50) // saved 950

	got := r.Summary()
	highIdx := strings.Index(got, "high-savings")
	lowIdx := strings.Index(got, "low-savings")
	if highIdx > lowIdx {
		t.Error("Higher savings should appear first")
	}
}

func TestEntriesReturnsCopy(t *testing.T) {
	r := New()
	r.Record("test", 100, 50)

	entries := r.Entries()
	entries[0].FilterName = "modified"

	// Original should not be affected
	original := r.Entries()
	if original[0].FilterName != "test" {
		t.Error("Entries() should return a copy")
	}
}
