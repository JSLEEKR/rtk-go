package filter

import (
	"fmt"
	"strings"
	"testing"
)

// --- GitStatusFilter Tests ---

func TestGitStatusFilterName(t *testing.T) {
	f := &GitStatusFilter{}
	if f.Name() != "git-status" {
		t.Errorf("Name() = %q, want %q", f.Name(), "git-status")
	}
}

func TestGitStatusFilterMatch(t *testing.T) {
	f := &GitStatusFilter{}
	tests := []struct {
		cmd  string
		args []string
		want bool
	}{
		{"git", []string{"status"}, true},
		{"git", []string{"status", "--short"}, true},
		{"git", []string{"diff"}, false},
		{"git", []string{}, false},
		{"ls", []string{"status"}, false},
	}
	for _, tt := range tests {
		got := f.Match(tt.cmd, tt.args)
		if got != tt.want {
			t.Errorf("Match(%q, %v) = %v, want %v", tt.cmd, tt.args, got, tt.want)
		}
	}
}

func TestGitStatusClean(t *testing.T) {
	f := &GitStatusFilter{}
	input := `On branch main
nothing to commit, working tree clean`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "clean") {
		t.Errorf("Expected clean status, got: %q", got)
	}
	if !strings.Contains(got, "main") {
		t.Errorf("Expected branch name, got: %q", got)
	}
}

func TestGitStatusModified(t *testing.T) {
	f := &GitStatusFilter{}
	input := `On branch feature/test
Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
	modified:   src/main.go
	modified:   src/util.go

Untracked files:
  (use "git add <file>..." to include in what will be committed)
	newfile.txt
	another.txt`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "feature/test") {
		t.Errorf("Expected branch name, got: %q", got)
	}
	if !strings.Contains(got, "2 modified") {
		t.Errorf("Expected 2 modified, got: %q", got)
	}
	if !strings.Contains(got, "2 untracked") {
		t.Errorf("Expected 2 untracked, got: %q", got)
	}
}

func TestGitStatusStaged(t *testing.T) {
	f := &GitStatusFilter{}
	input := `On branch main
Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
	new file:   README.md
	new file:   main.go`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "2 staged") {
		t.Errorf("Expected 2 staged, got: %q", got)
	}
}

func TestGitStatusDeleted(t *testing.T) {
	f := &GitStatusFilter{}
	input := `On branch main
Changes not staged for commit:
	deleted:    old-file.go`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "1 deleted") {
		t.Errorf("Expected 1 deleted, got: %q", got)
	}
}

func TestGitStatusStagedModification(t *testing.T) {
	f := &GitStatusFilter{}
	input := `On branch main
Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
	modified:   staged-file.go

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
	modified:   unstaged-file.go`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "1 staged") {
		t.Errorf("Expected 1 staged (for staged modification), got: %q", got)
	}
	if !strings.Contains(got, "1 modified") {
		t.Errorf("Expected 1 modified (for unstaged modification), got: %q", got)
	}
	if !strings.Contains(got, "staged-file.go") {
		t.Errorf("Expected staged-file.go in output, got: %q", got)
	}
	if !strings.Contains(got, "unstaged-file.go") {
		t.Errorf("Expected unstaged-file.go in output, got: %q", got)
	}
}

func TestGitStatusExitCodeNonZero(t *testing.T) {
	f := &GitStatusFilter{}
	input := "fatal: not a git repository"
	got := f.Apply(input, 128)
	if got != input {
		t.Error("Non-zero exit should return raw output")
	}
}

func TestGitStatusEmptyOutput(t *testing.T) {
	f := &GitStatusFilter{}
	got := f.Apply("", 0)
	if got != "" {
		t.Errorf("Empty input should return empty, got: %q", got)
	}
}

func TestGitStatusCompression(t *testing.T) {
	f := &GitStatusFilter{}
	// Simulate verbose git status (~200 tokens)
	input := `On branch main
Your branch is up to date with 'origin/main'.

Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
	new file:   a.go

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git restore <file>..." to discard changes in working directory)
	modified:   b.go
	modified:   c.go

Untracked files:
  (use "git add <file>..." to include in what will be committed)
	d.txt`

	got := f.Apply(input, 0)
	if len(got) >= len(input) {
		t.Error("Filtered output should be smaller than input")
	}
}

// --- GitDiffFilter Tests ---

func TestGitDiffFilterName(t *testing.T) {
	f := &GitDiffFilter{}
	if f.Name() != "git-diff" {
		t.Errorf("Name() = %q, want %q", f.Name(), "git-diff")
	}
}

func TestGitDiffFilterMatch(t *testing.T) {
	f := &GitDiffFilter{}
	tests := []struct {
		cmd  string
		args []string
		want bool
	}{
		{"git", []string{"diff"}, true},
		{"git", []string{"diff", "--cached"}, true},
		{"git", []string{"status"}, false},
		{"git", []string{}, false},
		{"diff", []string{}, false},
	}
	for _, tt := range tests {
		got := f.Match(tt.cmd, tt.args)
		if got != tt.want {
			t.Errorf("Match(%q, %v) = %v, want %v", tt.cmd, tt.args, got, tt.want)
		}
	}
}

func TestGitDiffEmpty(t *testing.T) {
	f := &GitDiffFilter{}
	got := f.Apply("", 0)
	if got != "no changes" {
		t.Errorf("Empty diff should return 'no changes', got: %q", got)
	}
}

func TestGitDiffSimple(t *testing.T) {
	f := &GitDiffFilter{}
	input := `diff --git a/main.go b/main.go
index abc123..def456 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

+import "fmt"
+
 func main() {
-    println("hello")
+    fmt.Println("hello")
 }`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "1 file(s) changed") {
		t.Errorf("Expected file count, got: %q", got)
	}
	if !strings.Contains(got, "+3") {
		t.Errorf("Expected addition count, got: %q", got)
	}
}

func TestGitDiffTruncatesLargeFile(t *testing.T) {
	f := &GitDiffFilter{}
	var b strings.Builder
	b.WriteString("diff --git a/big.go b/big.go\n")
	b.WriteString("@@ -1,200 +1,200 @@\n")
	for i := 0; i < 200; i++ {
		b.WriteString(fmt.Sprintf("+line %d added\n", i))
	}

	got := f.Apply(b.String(), 0)
	if !strings.Contains(got, "truncated") {
		t.Error("Large diff should be truncated")
	}
}

func TestGitDiffMultipleFiles(t *testing.T) {
	f := &GitDiffFilter{}
	input := `diff --git a/a.go b/a.go
@@ -1 +1 @@
-old
+new
diff --git a/b.go b/b.go
@@ -1 +1 @@
-old
+new`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "2 file(s) changed") {
		t.Errorf("Expected 2 files, got: %q", got)
	}
}

// --- GitLogFilter Tests ---

func TestGitLogFilterName(t *testing.T) {
	f := &GitLogFilter{}
	if f.Name() != "git-log" {
		t.Errorf("Name() = %q, want %q", f.Name(), "git-log")
	}
}

func TestGitLogFilterMatch(t *testing.T) {
	f := &GitLogFilter{}
	if !f.Match("git", []string{"log"}) {
		t.Error("Should match git log")
	}
	if f.Match("git", []string{"status"}) {
		t.Error("Should not match git status")
	}
}

func TestGitLogEmpty(t *testing.T) {
	f := &GitLogFilter{}
	got := f.Apply("", 0)
	if got != "no commits" {
		t.Errorf("Empty log should return 'no commits', got: %q", got)
	}
}

func TestGitLogStripsTrailers(t *testing.T) {
	f := &GitLogFilter{}
	input := `commit abc123
Author: Test <test@example.com>
Date:   Mon Jan 1 00:00:00 2024 +0000

    Add feature

    Signed-off-by: Test <test@example.com>
    Co-authored-by: Bot <bot@example.com>`

	got := f.Apply(input, 0)
	if strings.Contains(got, "Signed-off-by") {
		t.Error("Trailers should be stripped")
	}
	if strings.Contains(got, "Co-authored-by") {
		t.Error("Trailers should be stripped")
	}
	if !strings.Contains(got, "Add feature") {
		t.Error("Commit message should be preserved")
	}
}

func TestGitLogTruncates(t *testing.T) {
	f := &GitLogFilter{}
	var b strings.Builder
	for i := 0; i < 20; i++ {
		b.WriteString(fmt.Sprintf("commit %040d\n", i))
		b.WriteString("Author: Test <test@example.com>\n\n")
		b.WriteString(fmt.Sprintf("    Commit %d\n\n", i))
	}

	got := f.Apply(b.String(), 0)
	if !strings.Contains(got, "showing 10 of 20") {
		t.Errorf("Expected truncation message, got: %q", got)
	}
}

func TestGitLogFewCommits(t *testing.T) {
	f := &GitLogFilter{}
	input := `commit abc123
Author: Test <test@example.com>

    First commit`

	got := f.Apply(input, 0)
	if strings.Contains(got, "showing") {
		t.Error("Should not show truncation for few commits")
	}
}
