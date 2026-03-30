package filter

import (
	"fmt"
	"strings"
	"testing"
)

// --- GrepFilter Tests ---

func TestGrepFilterName(t *testing.T) {
	f := &GrepFilter{}
	if f.Name() != "grep" {
		t.Errorf("Name() = %q, want %q", f.Name(), "grep")
	}
}

func TestGrepFilterMatch(t *testing.T) {
	f := &GrepFilter{}
	tests := []struct {
		cmd  string
		want bool
	}{
		{"grep", true},
		{"rg", true},
		{"ripgrep", true},
		{"find", false},
		{"git", false},
	}
	for _, tt := range tests {
		got := f.Match(tt.cmd, nil)
		if got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestGrepEmpty(t *testing.T) {
	f := &GrepFilter{}
	got := f.Apply("", 1, nil)
	if got != "no matches" {
		t.Errorf("Empty grep should return 'no matches', got: %q", got)
	}
}

func TestGrepGroupsByFile(t *testing.T) {
	f := &GrepFilter{}
	input := `src/main.go:10:func main() {
src/main.go:20:func helper() {
src/util.go:5:func util() {`

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "## src/main.go (2 matches)") {
		t.Errorf("Expected file grouping, got: %q", got)
	}
	if !strings.Contains(got, "## src/util.go (1 matches)") {
		t.Errorf("Expected file grouping, got: %q", got)
	}
	if !strings.Contains(got, "3 matches in 2 files") {
		t.Errorf("Expected summary, got: %q", got)
	}
}

func TestGrepPerFileLimiting(t *testing.T) {
	f := &GrepFilter{}
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf("bigfile.go:%d:match %d", i+1, i+1))
	}
	input := strings.Join(lines, "\n")

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "+25 more matches") {
		t.Errorf("Expected per-file limiting message, got: %q", got)
	}
}

func TestGrepTotalLimiting(t *testing.T) {
	f := &GrepFilter{}
	var lines []string
	for i := 0; i < 300; i++ {
		lines = append(lines, fmt.Sprintf("file%d.go:1:match", i))
	}
	input := strings.Join(lines, "\n")

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "showing 200") {
		t.Errorf("Expected total limiting message, got: %q", got)
	}
}

func TestGrepLineTruncation(t *testing.T) {
	f := &GrepFilter{}
	longLine := "file.go:1:" + strings.Repeat("x", 300)

	got := f.Apply(longLine, 0, nil)
	if !strings.Contains(got, "...") {
		t.Error("Long lines should be truncated")
	}
}

func TestGrepSeparatorLines(t *testing.T) {
	f := &GrepFilter{}
	input := `file.go:1:match1
--
file.go:5:match2`

	got := f.Apply(input, 0, nil)
	// The separator "--" should not appear as its own line
	for _, line := range strings.Split(got, "\n") {
		if strings.TrimSpace(line) == "--" {
			t.Error("Separator lines should be removed")
		}
	}
}

// L5 fix: Test Windows drive letter parsing
func TestGrepWindowsDriveLetter(t *testing.T) {
	f := &GrepFilter{}
	input := `C:\Users\test\file.go:10:func main() {
C:\Users\test\file.go:20:func helper() {`

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "C:\\Users\\test\\file.go") {
		t.Errorf("Expected Windows path preserved, got: %q", got)
	}
}

// --- FindFilter Tests ---

func TestFindFilterName(t *testing.T) {
	f := &FindFilter{}
	if f.Name() != "find" {
		t.Errorf("Name() = %q, want %q", f.Name(), "find")
	}
}

func TestFindFilterMatch(t *testing.T) {
	f := &FindFilter{}
	if !f.Match("find", nil) {
		t.Error("Should match find")
	}
	if !f.Match("fd", nil) {
		t.Error("Should match fd")
	}
	if f.Match("grep", nil) {
		t.Error("Should not match grep")
	}
}

func TestFindEmpty(t *testing.T) {
	f := &FindFilter{}
	got := f.Apply("", 0, nil)
	if got != "no results" {
		t.Errorf("Empty find should return 'no results', got: %q", got)
	}
}

func TestFindGroupsByDirectory(t *testing.T) {
	f := &FindFilter{}
	input := `src/main.go
src/util.go
tests/main_test.go`

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "src/") {
		t.Error("Should group by directory")
	}
	if !strings.Contains(got, "tests/") {
		t.Error("Should group by directory")
	}
	if !strings.Contains(got, "3F 2D") {
		t.Errorf("Expected file/dir summary, got: %q", got)
	}
}

func TestFindExtensionSummary(t *testing.T) {
	f := &FindFilter{}
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, fmt.Sprintf("src/file%d.go", i))
	}
	for i := 0; i < 5; i++ {
		lines = append(lines, fmt.Sprintf("src/file%d.ts", i))
	}
	input := strings.Join(lines, "\n")

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, ".go(10)") {
		t.Errorf("Expected .go count, got: %q", got)
	}
	if !strings.Contains(got, ".ts(5)") {
		t.Errorf("Expected .ts count, got: %q", got)
	}
}

func TestFindResultsLimiting(t *testing.T) {
	f := &FindFilter{}
	var lines []string
	for i := 0; i < 150; i++ {
		lines = append(lines, fmt.Sprintf("dir%d/file%d.go", i%5, i))
	}
	input := strings.Join(lines, "\n")

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "showing 100 of 150") {
		t.Errorf("Expected limiting message, got: %q", got)
	}
}

// --- LSFilter Tests ---

func TestLSFilterName(t *testing.T) {
	f := &LSFilter{}
	if f.Name() != "ls" {
		t.Errorf("Name() = %q, want %q", f.Name(), "ls")
	}
}

func TestLSFilterMatch(t *testing.T) {
	f := &LSFilter{}
	if !f.Match("ls", nil) {
		t.Error("Should match ls")
	}
	if !f.Match("dir", nil) {
		t.Error("Should match dir")
	}
}

func TestLSEmpty(t *testing.T) {
	f := &LSFilter{}
	got := f.Apply("", 0, nil)
	if got != "empty directory" {
		t.Errorf("Expected 'empty directory', got: %q", got)
	}
}

func TestLSFiltersNoise(t *testing.T) {
	f := &LSFilter{}
	input := `node_modules
src
.git
package.json
__pycache__`

	got := f.Apply(input, 0, nil)
	if strings.Contains(got, "node_modules") {
		t.Error("node_modules should be filtered")
	}
	if strings.Contains(got, ".git\n") {
		t.Error(".git should be filtered")
	}
	if strings.Contains(got, "__pycache__") {
		t.Error("__pycache__ should be filtered")
	}
	if !strings.Contains(got, "src") {
		t.Error("src should remain")
	}
	if !strings.Contains(got, "package.json") {
		t.Error("package.json should remain")
	}
	if !strings.Contains(got, "3 noise dirs hidden") {
		t.Errorf("Expected noise count, got: %q", got)
	}
}

func TestLSItemCount(t *testing.T) {
	f := &LSFilter{}
	input := `file1.go
file2.go
file3.go`

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "3 items") {
		t.Errorf("Expected 3 items, got: %q", got)
	}
}

// M5 fix: Test ls -l parsing with 9+ fields
func TestLSLongFormat(t *testing.T) {
	f := &LSFilter{}
	input := `drwxr-xr-x  2 user group 4096 Jan  1 12:00 src
-rw-r--r--  1 user group  100 Jan  1 12:00 main.go
drwxr-xr-x  2 user group 4096 Jan  1 12:00 node_modules`

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "src") {
		t.Error("Should extract name from ls -l format")
	}
	if !strings.Contains(got, "main.go") {
		t.Error("Should extract name from ls -l format")
	}
	if strings.Contains(got, "node_modules") {
		t.Error("node_modules should still be filtered in ls -l format")
	}
}

// --- Helper function tests ---

func TestParseGrepLine(t *testing.T) {
	tests := []struct {
		line    string
		file    string
		lineNum string
		content string
	}{
		{"file.go:10:content", "file.go", "10", "content"},
		{"file.go:content", "file.go", "", "content"},
		{"file.go:10:content:with:colons", "file.go", "10", "content:with:colons"},
	}
	for _, tt := range tests {
		file, lineNum, content := parseGrepLine(tt.line)
		if file != tt.file || lineNum != tt.lineNum || content != tt.content {
			t.Errorf("parseGrepLine(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tt.line, file, lineNum, content, tt.file, tt.lineNum, tt.content)
		}
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"123", true},
		{"0", true},
		{"", false},
		{"abc", false},
		{"12a", false},
	}
	for _, tt := range tests {
		if got := isNumeric(tt.s); got != tt.want {
			t.Errorf("isNumeric(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

// M4 fix: Test rune-based truncation
func TestTruncateLineRunes(t *testing.T) {
	short := "hello"
	if got := truncateLine(short, 10); got != short {
		t.Errorf("Short line should not be truncated")
	}

	long := strings.Repeat("x", 20)
	got := truncateLine(long, 10)
	if len(got) != 13 { // 10 + "..."
		t.Errorf("Long line truncation, got len %d", len(got))
	}

	// Unicode test: 5 runes, each 3 bytes in UTF-8
	unicode := "hello" + strings.Repeat("\u4e16", 10) // 15 runes total
	got = truncateLine(unicode, 10)
	runes := []rune(got)
	// Should be 10 runes + "..." (3 runes) = 13 runes
	if len(runes) != 13 {
		t.Errorf("Unicode truncation: got %d runes, want 13", len(runes))
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path string
		dir  string
		file string
	}{
		{"src/main.go", "src", "main.go"},
		{"main.go", ".", "main.go"},
		{"a/b/c.go", "a/b", "c.go"},
		{"src\\main.go", "src", "main.go"},
	}
	for _, tt := range tests {
		dir, file := splitPath(tt.path)
		if dir != tt.dir || file != tt.file {
			t.Errorf("splitPath(%q) = (%q, %q), want (%q, %q)",
				tt.path, dir, file, tt.dir, tt.file)
		}
	}
}

func TestGetExtension(t *testing.T) {
	tests := []struct {
		filename string
		ext      string
	}{
		{"main.go", ".go"},
		{"file.test.ts", ".ts"},
		{"Makefile", ""},
		{".gitignore", ""},
	}
	for _, tt := range tests {
		got := getExtension(tt.filename)
		if got != tt.ext {
			t.Errorf("getExtension(%q) = %q, want %q", tt.filename, got, tt.ext)
		}
	}
}
