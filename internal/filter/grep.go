package filter

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/JSLEEKR/rtk-go/internal/config"
)

// MaxGrepResults is the default maximum total grep matches to show.
const MaxGrepResults = 200

// MaxGrepPerFile is the default maximum matches per file.
const MaxGrepPerFile = 25

// GrepFilter groups grep/rg results by file and enforces limits.
type GrepFilter struct{}

func (f *GrepFilter) Name() string { return "grep" }

func (f *GrepFilter) Match(cmd string, args []string) bool {
	return cmd == "grep" || cmd == "rg" || cmd == "ripgrep"
}

func (f *GrepFilter) Apply(output string, exitCode int, cfg *config.FilterConfig) string {
	if output == "" {
		return "no matches"
	}

	maxResults := MaxGrepResults
	maxPerFile := MaxGrepPerFile
	if cfg != nil {
		if cfg.GrepMaxResults > 0 {
			maxResults = cfg.GrepMaxResults
		}
		if cfg.GrepMaxPerFile > 0 {
			maxPerFile = cfg.GrepMaxPerFile
		}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// Group results by file
	type match struct {
		lineNum string
		content string
	}
	groups := make(map[string][]match)
	var fileOrder []string
	totalMatches := 0

	for _, line := range lines {
		if line == "" || line == "--" {
			continue
		}

		// Parse "file:line:content" or "file:content" or "file-line-content"
		file, lineNum, content := parseGrepLine(line)
		if file == "" {
			continue
		}

		if _, exists := groups[file]; !exists {
			fileOrder = append(fileOrder, file)
		}

		groups[file] = append(groups[file], match{lineNum: lineNum, content: content})
		totalMatches++
	}

	if len(groups) == 0 {
		return output // couldn't parse, return raw
	}

	var b strings.Builder
	shown := 0

	for _, file := range fileOrder {
		matches := groups[file]
		if shown >= maxResults {
			break
		}

		b.WriteString(fmt.Sprintf("## %s (%d matches)\n", file, len(matches)))

		limit := len(matches)
		if limit > maxPerFile {
			limit = maxPerFile
		}
		remaining := maxResults - shown
		if limit > remaining {
			limit = remaining
		}

		for i := 0; i < limit; i++ {
			m := matches[i]
			if m.lineNum != "" {
				b.WriteString(fmt.Sprintf("  %s: %s\n", m.lineNum, truncateLine(m.content, 200)))
			} else {
				b.WriteString(fmt.Sprintf("  %s\n", truncateLine(m.content, 200)))
			}
			shown++
		}

		if len(matches) > limit {
			b.WriteString(fmt.Sprintf("  ... +%d more matches\n", len(matches)-limit))
		}
		b.WriteByte('\n')
	}

	b.WriteString(fmt.Sprintf("--- %d matches in %d files", totalMatches, len(groups)))
	if totalMatches > maxResults {
		b.WriteString(fmt.Sprintf(" (showing %d)", shown))
	}

	return b.String()
}

// windowsDriveRegex matches Windows drive letter paths like C:\path or D:/path
var windowsDriveRegex = regexp.MustCompile(`^[A-Za-z]:`)

// parseGrepLine parses a grep output line into file, lineNum, content.
// L5 fix: handles Windows drive letter colons.
func parseGrepLine(line string) (file, lineNum, content string) {
	parseLine := line

	// L5 fix: skip first colon if line starts with a drive letter (e.g., C:\path)
	drivePrefix := ""
	if windowsDriveRegex.MatchString(parseLine) && len(parseLine) > 2 {
		drivePrefix = parseLine[:2]
		parseLine = parseLine[2:]
	}

	// Try "file:lineNum:content" first
	parts := strings.SplitN(parseLine, ":", 3)
	if len(parts) == 3 {
		// Check if second part looks like a line number
		if isNumeric(parts[1]) {
			return drivePrefix + parts[0], parts[1], parts[2]
		}
		// Might be "file:content" with colons in content
		return drivePrefix + parts[0], "", strings.Join(parts[1:], ":")
	}
	if len(parts) == 2 {
		if isNumeric(parts[1]) {
			return drivePrefix + parts[0], parts[1], ""
		}
		return drivePrefix + parts[0], "", parts[1]
	}
	return "", "", line
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// truncateLine truncates a string using rune count instead of byte count (M4 fix).
func truncateLine(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "..."
}

// --- Find/LS Filters ---

// FindFilter compresses find output by grouping by directory
// and providing extension summaries.
type FindFilter struct{}

func (f *FindFilter) Name() string { return "find" }

func (f *FindFilter) Match(cmd string, args []string) bool {
	return cmd == "find" || cmd == "fd"
}

// MaxFindResults is the default maximum files to list.
const MaxFindResults = 100

func (f *FindFilter) Apply(output string, exitCode int, cfg *config.FilterConfig) string {
	if output == "" {
		return "no results"
	}

	maxResults := MaxFindResults
	if cfg != nil && cfg.FindMaxResults > 0 {
		maxResults = cfg.FindMaxResults
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) == 0 {
		return "no results"
	}

	// Group by parent directory
	dirFiles := make(map[string][]string)
	var dirOrder []string
	extCount := make(map[string]int)
	totalFiles := 0
	totalDirs := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		totalFiles++

		// Extract directory and filename
		dir, file := splitPath(line)
		if _, exists := dirFiles[dir]; !exists {
			dirOrder = append(dirOrder, dir)
			totalDirs++
		}
		dirFiles[dir] = append(dirFiles[dir], file)

		// Count extensions
		ext := getExtension(file)
		if ext != "" {
			extCount[ext]++
		}
	}

	var b strings.Builder

	shown := 0
	for _, dir := range dirOrder {
		files := dirFiles[dir]
		if shown >= maxResults {
			break
		}

		b.WriteString(dir)
		b.WriteString("/\n")

		limit := len(files)
		remaining := maxResults - shown
		if limit > remaining {
			limit = remaining
		}

		for i := 0; i < limit; i++ {
			b.WriteString(fmt.Sprintf("  %s\n", files[i]))
			shown++
		}

		if len(files) > limit {
			b.WriteString(fmt.Sprintf("  ... +%d more\n", len(files)-limit))
		}
	}

	// Extension summary
	b.WriteString(fmt.Sprintf("\n--- %dF %dD", totalFiles, totalDirs))
	if len(extCount) > 0 {
		b.WriteString(": ")
		// Sort by frequency (top 5)
		type extPair struct {
			ext   string
			count int
		}
		pairs := make([]extPair, 0, len(extCount))
		for ext, count := range extCount {
			pairs = append(pairs, extPair{ext, count})
		}
		// Insertion sort by count descending
		for i := 1; i < len(pairs); i++ {
			key := pairs[i]
			j := i - 1
			for j >= 0 && pairs[j].count < key.count {
				pairs[j+1] = pairs[j]
				j--
			}
			pairs[j+1] = key
		}
		limit := 5
		if len(pairs) < limit {
			limit = len(pairs)
		}
		extStrs := make([]string, limit)
		for i := 0; i < limit; i++ {
			extStrs[i] = fmt.Sprintf("%s(%d)", pairs[i].ext, pairs[i].count)
		}
		b.WriteString(strings.Join(extStrs, " "))
	}

	if totalFiles > maxResults {
		b.WriteString(fmt.Sprintf(" [showing %d of %d]", shown, totalFiles))
	}

	return b.String()
}

func splitPath(path string) (dir, file string) {
	// Normalize to forward slashes
	path = strings.ReplaceAll(path, "\\", "/")
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ".", path
	}
	return path[:idx], path[idx+1:]
}

func getExtension(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx <= 0 || idx == len(filename)-1 {
		return ""
	}
	return filename[idx:]
}

// --- LS Filter ---

// LSFilter compresses ls output by filtering noise directories
// and providing summaries.
type LSFilter struct{}

// NoiseDirectories are directories commonly excluded from listings.
var NoiseDirectories = map[string]bool{
	"node_modules": true, ".git": true, "target": true,
	"__pycache__": true, ".next": true, "dist": true,
	"build": true, ".cache": true, "vendor": true,
	".tox": true, ".mypy_cache": true, ".pytest_cache": true,
	"coverage": true, ".nyc_output": true, ".venv": true,
	"venv": true, "env": true,
}

func (f *LSFilter) Name() string { return "ls" }

func (f *LSFilter) Match(cmd string, args []string) bool {
	return cmd == "ls" || cmd == "dir"
}

func (f *LSFilter) Apply(output string, exitCode int, cfg *config.FilterConfig) string {
	if output == "" {
		return "empty directory"
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	var filtered []string
	noiseSkipped := 0

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}

		// Strip common ls -l metadata prefix (keep just the name)
		// M5 fix: use len(fields) >= 9 as additional check (ls -l always has 9+ fields)
		fields := strings.Fields(name)
		if len(fields) >= 9 && len(fields[0]) >= 10 &&
			(fields[0][0] == 'd' || fields[0][0] == '-' || fields[0][0] == 'l') {
			// Looks like ls -l output (permissions, links, owner, group, size, month, day, time, name)
			name = fields[len(fields)-1]
		}

		if NoiseDirectories[name] {
			noiseSkipped++
			continue
		}

		filtered = append(filtered, name)
	}

	var b strings.Builder
	for _, f := range filtered {
		b.WriteString(f)
		b.WriteByte('\n')
	}

	b.WriteString(fmt.Sprintf("--- %d items", len(filtered)))
	if noiseSkipped > 0 {
		b.WriteString(fmt.Sprintf(" (%d noise dirs hidden)", noiseSkipped))
	}

	return b.String()
}
