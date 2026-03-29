// Package filter provides a unified interface for command-specific output filters.
// Unlike rtk's 72 separate Rust modules with no shared abstraction, rtk-go uses
// a single Filter interface that all command filters implement.
package filter

// Filter is the unified interface for all command-specific output filters.
// Each filter knows how to identify commands it handles and how to compress output.
type Filter interface {
	// Name returns the human-readable name of this filter.
	Name() string

	// Match returns true if this filter should handle the given command.
	// cmd is the base command name (e.g., "git"), args are the arguments.
	Match(cmd string, args []string) bool

	// Apply filters the raw command output, returning compressed output.
	// exitCode is preserved from the original command for context-aware filtering.
	Apply(output string, exitCode int) string
}

// Registry holds all registered filters and provides lookup by command.
type Registry struct {
	filters []Filter
}

// NewRegistry creates a Registry with the default set of filters.
func NewRegistry() *Registry {
	return &Registry{
		filters: []Filter{
			&GitStatusFilter{},
			&GitDiffFilter{},
			&GitLogFilter{},
			&GrepFilter{},
			&FindFilter{},
			&LSFilter{},
			&GoTestFilter{},
			&PytestFilter{},
			&NPMTestFilter{},
			&BuildFilter{},
			&GenericFilter{},
		},
	}
}

// Lookup finds the first matching filter for the given command and args.
// Returns the GenericFilter as fallback if no specific filter matches.
func (r *Registry) Lookup(cmd string, args []string) Filter {
	for _, f := range r.filters {
		if f.Match(cmd, args) {
			return f
		}
	}
	// Should not reach here since GenericFilter matches everything,
	// but return a GenericFilter as safety net.
	return &GenericFilter{}
}

// Filters returns all registered filters.
func (r *Registry) Filters() []Filter {
	return r.filters
}
