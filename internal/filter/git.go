package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JSLEEKR/rtk-go/internal/config"
)

// MaxStagedModified is the default maximum number of staged/modified files to show.
const MaxStagedModified = 15

// MaxUntracked is the maximum number of untracked files to show.
const MaxUntracked = 10

// MaxDiffLinesPerFile is the default maximum number of diff lines per file section.
const MaxDiffLinesPerFile = 100

// MaxLogCommits is the default maximum number of commits to show in git log.
const MaxLogCommits = 10

// --- Git Status Filter ---

// GitStatusFilter compresses git status output by parsing both porcelain
// and verbose format, summarizing by change type.
type GitStatusFilter struct{}

func (f *GitStatusFilter) Name() string { return "git-status" }

func (f *GitStatusFilter) Match(cmd string, args []string) bool {
	if cmd != "git" || len(args) == 0 {
		return false
	}
	return args[0] == "status"
}

// porcelainStatusRegex matches git status --porcelain format (e.g., " M file.go", "?? file.go")
var porcelainStatusRegex = regexp.MustCompile(`^([MADRCU?! ]{2})\s+(.+)$`)

func (f *GitStatusFilter) Apply(output string, exitCode int, cfg *config.FilterConfig) string {
	if output == "" {
		return ""
	}
	if exitCode != 0 {
		return output
	}

	maxStatus := MaxStagedModified
	if cfg != nil && cfg.GitStatusMax > 0 {
		maxStatus = cfg.GitStatusMax
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return output
	}

	// Try porcelain format first (H2 fix)
	if result := f.parsePorcelain(lines, maxStatus); result != "" {
		return result
	}

	// Fall back to verbose format
	return f.parseVerbose(lines, maxStatus)
}

func (f *GitStatusFilter) parsePorcelain(lines []string, maxStatus int) string {
	var staged, modified, untracked, deleted, conflicts []string

	isPorcelain := false
	for _, line := range lines {
		if m := porcelainStatusRegex.FindStringSubmatch(line); m != nil {
			isPorcelain = true
			status := m[1]
			file := m[2]

			// Index (staged) status
			switch status[0] {
			case 'A', 'R', 'C':
				staged = append(staged, file)
			case 'M':
				staged = append(staged, file)
			case 'D':
				deleted = append(deleted, file)
			case 'U':
				conflicts = append(conflicts, file)
			}

			// Worktree (unstaged) status
			switch status[1] {
			case 'M':
				// Only add to modified if not already staged
				if status[0] == ' ' {
					modified = append(modified, file)
				}
			case 'D':
				if status[0] == ' ' {
					deleted = append(deleted, file)
				}
			case 'U':
				conflicts = append(conflicts, file)
			}

			if status == "??" {
				// Reset: porcelain "??" means untracked
				// Remove from any previous category
				staged = removeLastIfMatch(staged, file)
				untracked = append(untracked, file)
			}
		}
	}

	if !isPorcelain {
		return ""
	}

	return f.formatOutput("", staged, modified, deleted, untracked, conflicts, maxStatus)
}

func removeLastIfMatch(slice []string, val string) []string {
	if len(slice) > 0 && slice[len(slice)-1] == val {
		return slice[:len(slice)-1]
	}
	return slice
}

func (f *GitStatusFilter) parseVerbose(lines []string, maxStatus int) string {
	var branch string
	var staged, modified, untracked, deleted, conflicts []string

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

		if strings.HasPrefix(line, "On branch ") {
			branch = strings.TrimPrefix(line, "On branch ")
			continue
		}
		if strings.HasPrefix(line, "HEAD detached at ") {
			branch = "HEAD detached at " + strings.TrimPrefix(line, "HEAD detached at ")
			continue
		}

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
			untracked = append(untracked, line)
		}
	}

	return f.formatOutput(branch, staged, modified, deleted, untracked, conflicts, maxStatus)
}

func (f *GitStatusFilter) formatOutput(branch string, staged, modified, deleted, untracked, conflicts []string, maxStatus int) string {
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
		return "clean"
	}

	if b.Len() > 0 {
		b.WriteString(" ")
	}
	b.WriteString(strings.Join(counts, ", "))

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

	writeFiles("Staged", staged, maxStatus)
	writeFiles("Modified", modified, maxStatus)
	writeFiles("Deleted", deleted, maxStatus)
	writeFiles("Untracked", untracked, MaxUntracked)
	writeFiles("Conflicts", conflicts, maxStatus)

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

func (f *GitDiffFilter) Apply(output string, exitCode int, cfg *config.FilterConfig) string {
	if output == "" {
		return "no changes"
	}

	maxDiffLines := MaxDiffLinesPerFile
	if cfg != nil && cfg.GitDiffMaxLines > 0 {
		maxDiffLines = cfg.GitDiffMaxLines
	}

	lines := strings.Split(output, "\n")
	var result strings.Builder
	var currentFile string
	var currentFileHeader string // H3 fix: track file metadata for re-emission
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
				result.WriteString(fmt.Sprintf("\n  ... [truncated, %d+ lines in %s]\n", maxDiffLines, currentFile))
			}
			currentFile = m[2]
			currentFileHeader = line
			linesInFile = 0
			fileTruncated = false
			fileCount++
			result.WriteString(line)
			result.WriteByte('\n')
			continue
		}

		// Track file metadata lines (---/+++ headers)
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "index ") {
			if !fileTruncated {
				result.WriteString(line)
				result.WriteByte('\n')
				// Don't count metadata towards limit
			}
			continue
		}

		if linesInFile >= maxDiffLines && !diffHunkHeader.MatchString(line) {
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

		// H3 fix: re-emit file header when starting a new hunk after truncation
		if fileTruncated && diffHunkHeader.MatchString(line) {
			result.WriteString(fmt.Sprintf("\n  ... [truncated, %d+ lines in %s]\n", maxDiffLines, currentFile))
			result.WriteString(currentFileHeader)
			result.WriteByte('\n')
			fileTruncated = false
			linesInFile = 0
			// truncatedFiles already counted when fileTruncated was first set
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
		result.WriteString(fmt.Sprintf("\n  ... [truncated, %d+ lines in %s]\n", maxDiffLines, currentFile))
	}

	// Summary
	result.WriteString(fmt.Sprintf("\n--- %d file(s) changed, +%d -%d", fileCount, totalAdded, totalRemoved))
	if truncatedFiles > 0 {
		result.WriteString(fmt.Sprintf(" (%d file(s) truncated at %d lines)", truncatedFiles, maxDiffLines))
	}

	return result.String()
}

// --- Git Log Filter ---

// GitLogFilter compresses git log output by limiting commits shown
// and stripping trailers. Handles both standard and --oneline formats (H1 fix).
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

// onelineCommitRegex matches --oneline format: hash followed by space and message
var onelineCommitRegex = regexp.MustCompile(`^[0-9a-f]{7,40}\s`)

func isTrailer(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, prefix := range trailerPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// isCommitLine detects both standard ("commit <hash>") and --oneline ("<hash> <msg>") formats.
func isCommitLine(line string) bool {
	if strings.HasPrefix(line, "commit ") && len(line) > 7 {
		hash := strings.TrimPrefix(line, "commit ")
		hash = strings.TrimSpace(hash)
		return len(hash) >= 7
	}
	return onelineCommitRegex.MatchString(line)
}

func (f *GitLogFilter) Apply(output string, exitCode int, cfg *config.FilterConfig) string {
	if output == "" {
		return "no commits"
	}

	maxCommits := MaxLogCommits
	if cfg != nil && cfg.GitLogMaxCommits > 0 {
		maxCommits = cfg.GitLogMaxCommits
	}

	lines := strings.Split(output, "\n")
	var result strings.Builder
	commitCount := 0
	totalCommits := 0

	for _, line := range lines {
		if isCommitLine(line) {
			totalCommits++
		}
	}

	for _, line := range lines {
		if isCommitLine(line) {
			commitCount++
			if commitCount > maxCommits {
				break
			}
		}

		if commitCount > maxCommits {
			break
		}

		// Strip trailers
		if isTrailer(line) {
			continue
		}

		result.WriteString(line)
		result.WriteByte('\n')
	}

	if totalCommits > maxCommits {
		result.WriteString(fmt.Sprintf("\n... [showing %d of %d commits]", maxCommits, totalCommits))
	}

	return strings.TrimRight(result.String(), "\n")
}
