// Package report provides token savings reporting and analytics.
package report

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/JSLEEKR/rtk-go/internal/token"
)

// Entry records a single filter application.
type Entry struct {
	FilterName   string
	InputTokens  int
	OutputTokens int
}

// SavingsPercent returns the savings percentage for this entry.
func (e Entry) SavingsPercent() float64 {
	return token.Savings(e.InputTokens, e.OutputTokens)
}

// Reporter accumulates token savings across multiple command executions.
type Reporter struct {
	mu      sync.Mutex
	entries []Entry
}

// New creates a new Reporter.
func New() *Reporter {
	return &Reporter{}
}

// Record adds a filter application result.
func (r *Reporter) Record(filterName string, inputTokens, outputTokens int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, Entry{
		FilterName:   filterName,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	})
}

// Summary returns a formatted summary of all recorded savings.
func (r *Reporter) Summary() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.entries) == 0 {
		return "no commands recorded"
	}

	totalInput := 0
	totalOutput := 0
	filterCounts := make(map[string]int)
	filterSavings := make(map[string]int)

	for _, e := range r.entries {
		totalInput += e.InputTokens
		totalOutput += e.OutputTokens
		filterCounts[e.FilterName]++
		filterSavings[e.FilterName] += e.InputTokens - e.OutputTokens
	}

	var b strings.Builder
	b.WriteString("=== rtk-go Token Savings Report ===\n\n")
	b.WriteString(fmt.Sprintf("Commands:      %d\n", len(r.entries)))
	b.WriteString(fmt.Sprintf("Input tokens:  %d\n", totalInput))
	b.WriteString(fmt.Sprintf("Output tokens: %d\n", totalOutput))
	b.WriteString(fmt.Sprintf("Saved:         %d (%.1f%%)\n", totalInput-totalOutput, token.Savings(totalInput, totalOutput)))

	b.WriteString("\nBy filter:\n")

	// L6 fix: Sort by savings descending using sort.Slice
	type filterInfo struct {
		name    string
		count   int
		savings int
	}
	filters := make([]filterInfo, 0, len(filterCounts))
	for name, count := range filterCounts {
		filters = append(filters, filterInfo{name, count, filterSavings[name]})
	}
	sort.Slice(filters, func(i, j int) bool {
		return filters[i].savings > filters[j].savings
	})

	for _, f := range filters {
		b.WriteString(fmt.Sprintf("  %-15s %3d commands, %6d tokens saved\n", f.name, f.count, f.savings))
	}

	return b.String()
}

// TotalSaved returns the total tokens saved across all entries.
func (r *Reporter) TotalSaved() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	total := 0
	for _, e := range r.entries {
		total += e.InputTokens - e.OutputTokens
	}
	return total
}

// Entries returns a copy of all recorded entries.
func (r *Reporter) Entries() []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]Entry, len(r.entries))
	copy(cp, r.entries)
	return cp
}
