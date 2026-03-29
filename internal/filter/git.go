package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// MaxStagedModified is the maximum number of staged/modified files to show.
const MaxStagedModified = 15

// MaxUntracked is the maximum number of untracked files to show.
const MaxUntracked = 10

// MaxDiffLinesPerFile is the maximum number of diff lines per file section.
const MaxDiffLinesPerFile = 100

// MaxLogCommits is the maximum number of commits to show in git log.
const MaxLogCommits = 10

// --- Git Status Filter ---

// GitStatusFilter compresses git status output by parsing porcelain format
// and summarizing by change type.
type GitStatusFilter struct{}

func (f *GitStatusFilter) Name() string { return "git-status" }

func (f *GitStatusFilter) Match(cmd string, args []string) bool {
	if cmd != "git" || len(args) == 0 {
		return false
	}
	return args[0] == "status"
}

func (f *GitStatusFilter) Apply(output string, exitCode int) string {
	if exitCode != 0 {
		return output
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return output
	}

	var branch string
	var staged, modified, untracked, deleted, conflicts []string

	// Track which section we're in to distinguish staged vs unstaged modifications
	const (
		sectionNone      = 0
		sectionStaged    = 1
		sectionUnstaged  = 2
		sectionUntracked = 3
	)
	currentSection := sectionNone

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect branch info
		if strings.HasPrefix(line, "On branch ") {
			branch = strings.TrimPrefix(line, "On branch ")
			continue
		}
		if strings.HasPrefix(line, "HEAD detached at ") {
			branch = "HEAD detached at " + strings.TrimPrefix(line, "HEAD detached at ")
			continue
		}

		// Track section headers
		if strings.HasPrefix(line, "Changes to be committed:") {
			currentSection = sectionStaged
			continue
		}
		if strings.HasPrefix(line, "Changes not staged") {
			currentSection = sectionUnstaged
			continue
		}
		if strings.HasPrefix(line, "Untracked files:") {
			currentSection = sectionUntracked
			continue
		}

		// Skip hint lines
		if strings.HasPrefix(line, "(use ") ||
			strings.HasPrefix(line, "no changes added") ||
			strings.HasPrefix(line, "Your branch is") {
			continue
		}
		if strings.Contains(line, "nothing to commit") {
			if branch != "" {
				return fmt.Sprintf("[%s] clean — nothing to commit", branch)
			}
			return "clean — nothing to commit"
		}

		// Parse status indicators with section awareness
		if strings.HasPrefix(line, "new file:") {
			staged = append(staged, strings.TrimSpace(strings.TrimPrefix(line, "new file:")))
		} else if strings.HasPrefix(line, "modified:") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "modified:"))
			if currentSection == sectionStaged {
				staged = append(staged, name)
			} else {
				modified = append(modified, name)
			}
		} else if strings.HasPrefix(line, "deleted:") {
			deleted = append(deleted, strings.TrimSpace(strings.TrimPrefix(line, "deleted:")))
		} else if strings.HasPrefix(line, "renamed:") {
			staged = append(staged, strings.TrimSpace(strings.TrimPrefix(line, "renamed:")))
		} else if strings.HasPrefix(line, "both modified:") {
			conflicts = append(conflicts, strings.TrimSpace(strings.TrimPrefix(line, "both modified:")))
		} else if !strings.Contains(line, ":") && line != "" {
			// Likely an untracked file (listed without prefix)
			untracked = append(untracked, line)
		}
	}

	var b strings.Builder
	if branch != "" {
		b.WriteString(fmt.Sprintf("[%s]", branch))
	}

	counts := make([]string, 0, 5)
	if len(staged) > 0 {
		counts = append(counts, fmt.Sprintf("%d staged", len(staged)))
	}
	if len(modified) > 0 {
		counts = append(counts, fmt.Sprintf("%d modified", len(modified)))
	}
	if len(deleted) > 0 {
		counts = append(counts, fmt.Sprintf("%d deleted", len(deleted)))
	}
	if len(untracked) > 0 {
		counts = append(counts, fmt.Sprintf("%d untracked", len(untracked)))
	}
	if len(conflicts) > 0 {
		counts = append(counts, fmt.Sprintf("%d conflicts", len(conflicts)))
	}

	if len(counts) == 0 {
		if branch != "" {
			return fmt.Sprintf("[%s] clean", branch)
		}
		return output // Couldn't parse, return raw
	}

	if b.Len() > 0 {
		b.WriteString(" ")
	}
	b.WriteString(strings.Join(counts, ", "))

	// List files with caps
	writeFiles := func(label string, files []string, max int) {
		if len(files) == 0 {
			return
		}
		b.WriteString(fmt.Sprintf("\n%s:", label))
		limit := len(files)
		if limit > max {
			limit = max
		}
		for _, f := range files[:limit] {
			b.WriteString(fmt.Sprintf("\n  %s", f))
		}
		if len(files) > max {
			b.WriteString(fmt.Sprintf("\n  ... and %d more", len(files)-max))
		}
	}

	writeFiles("Staged", staged, MaxStagedModified)
	writeFiles("Modified", modified, MaxStagedModified)
	writeFiles("Deleted", deleted, MaxStagedModified)
	writeFiles("Untracked", untracked, MaxUntracked)
	writeFiles("Conflicts", conflicts, MaxStagedModified)

	return b.String()
}

// --- Git Diff Filter ---

// GitDiffFilter compresses git diff output by limiting lines per file
// and providing recovery hints.
type GitDiffFilter struct{}

func (f *GitDiffFilter) Name() string { return "git-diff" }

func (f *GitDiffFilter) Match(cmd string, args []string) bool {
	if cmd != "git" || len(args) == 0 {
		return false
	}
	return args[0] == "diff"
}

var diffFileHeader = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
var diffHunkHeader = regexp.MustCompile(`^@@ .+ @@`)

func (f *GitDiffFilter) Apply(output string, exitCode int) string {
	if output == "" {
		return "no changes"
	}

	lines := strings.Split(output, "\n")
	var result strings.Builder
	var currentFile string
	linesInFile := 0
	truncatedFiles := 0
	totalAdded := 0
	totalRemoved := 0
	fileCount := 0
	fileTruncated := false

	for _, line := range lines {
		if m := diffFileHeader.FindStringSubmatch(line); m != nil {
			// New file section
			if fileTruncated {
				result.WriteString(fmt.Sprintf("\n  ... [truncated, %d+ lines in %s]\n", MaxDiffLinesPerFile, currentFile))
			}
			currentFile = m[2]
			linesInFile = 0
			fileTruncated = false
			fileCount++
			result.WriteString(line)
			result.WriteByte('\n')
			continue
		}

		if linesInFile >= MaxDiffLinesPerFile && !diffHunkHeader.MatchString(line) {
			if !fileTruncated {
				fileTruncated = true
				truncatedFiles++
			}
			// Still count additions/removals
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				totalAdded++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				totalRemoved++
			}
			continue
		}

		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			totalAdded++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			totalRemoved++
		}

		result.WriteString(line)
		result.WriteByte('\n')
		linesInFile++
	}

	if fileTruncated {
		result.WriteString(fmt.Sprintf("\n  ... [truncated, %d+ lines in %s]\n", MaxDiffLinesPerFile, currentFile))
	}

	// Summary
	result.WriteString(fmt.Sprintf("\n--- %d file(s) changed, +%d -%d", fileCount, totalAdded, totalRemoved))
	if truncatedFiles > 0 {
		result.WriteString(fmt.Sprintf(" (%d file(s) truncated at %d lines)", truncatedFiles, MaxDiffLinesPerFile))
	}

	return result.String()
}

// --- Git Log Filter ---

// GitLogFilter compresses git log output by limiting commits shown
// and stripping trailers.
type GitLogFilter struct{}

func (f *GitLogFilter) Name() string { return "git-log" }

func (f *GitLogFilter) Match(cmd string, args []string) bool {
	if cmd != "git" || len(args) == 0 {
		return false
	}
	return args[0] == "log"
}

var trailerPrefixes = []string{
	"Signed-off-by:",
	"Co-authored-by:",
	"Reviewed-by:",
	"Acked-by:",
	"Tested-by:",
	"Cc:",
}

func isTrailer(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, prefix := range trailerPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func (f *GitLogFilter) Apply(output string, exitCode int) string {
	if output == "" {
		return "no commits"
	}

	lines := strings.Split(output, "\n")
	var result strings.Builder
	commitCount := 0
	totalCommits := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "commit ") && len(line) > 7 {
			// Check if this looks like a commit hash line
			hash := strings.TrimPrefix(line, "commit ")
			hash = strings.TrimSpace(hash)
			if len(hash) >= 7 {
				totalCommits++
			}
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "commit ") && len(line) > 7 {
			commitCount++
			if commitCount > MaxLogCommits {
				break
			}
		}

		if commitCount > MaxLogCommits {
			break
		}

		// Strip trailers
		if isTrailer(line) {
			continue
		}

		result.WriteString(line)
		result.WriteByte('\n')
	}

	if totalCommits > MaxLogCommits {
		result.WriteString(fmt.Sprintf("\n... [showing %d of %d commits]", MaxLogCommits, totalCommits))
	}

	return strings.TrimRight(result.String(), "\n")
}
