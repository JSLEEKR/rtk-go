package filter

import (
	"fmt"
	"strings"
	"testing"
)

func TestGenericFilterName(t *testing.T) {
	f := &GenericFilter{}
	if f.Name() != "generic" {
		t.Errorf("Name() = %q, want %q", f.Name(), "generic")
	}
}

func TestGenericFilterMatchAll(t *testing.T) {
	f := &GenericFilter{}
	if !f.Match("anything", nil) {
		t.Error("Generic should match everything")
	}
	if !f.Match("", nil) {
		t.Error("Generic should match empty string")
	}
}

func TestGenericEmpty(t *testing.T) {
	f := &GenericFilter{}
	got := f.Apply("", 0)
	if got != "" {
		t.Errorf("Empty input should return empty, got: %q", got)
	}
}

func TestGenericStripsANSI(t *testing.T) {
	f := &GenericFilter{}
	input := "\x1b[31mERROR\x1b[0m: something failed"
	got := f.Apply(input, 0)
	if strings.Contains(got, "\x1b") {
		t.Error("ANSI codes should be stripped")
	}
	if !strings.Contains(got, "ERROR: something failed") {
		t.Errorf("Content should be preserved, got: %q", got)
	}
}

func TestGenericCollapsesBlankLines(t *testing.T) {
	f := &GenericFilter{}
	input := "line1\n\n\n\n\nline2"
	got := f.Apply(input, 0)
	// Should have at most 2 consecutive blank lines
	if strings.Contains(got, "\n\n\n") {
		t.Error("Should collapse 3+ blank lines to 2")
	}
	if !strings.Contains(got, "line1") || !strings.Contains(got, "line2") {
		t.Error("Content should be preserved")
	}
}

func TestGenericTrimsTrailingWhitespace(t *testing.T) {
	f := &GenericFilter{}
	input := "line1   \t  \nline2  \n"
	got := f.Apply(input, 0)
	for _, line := range strings.Split(got, "\n") {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("Line has trailing whitespace: %q", line)
		}
	}
}

func TestGenericTruncatesLongOutput(t *testing.T) {
	f := &GenericFilter{}
	var lines []string
	for i := 0; i < 500; i++ {
		lines = append(lines, fmt.Sprintf("line %d content", i))
	}
	input := strings.Join(lines, "\n")

	got := f.Apply(input, 0)
	if !strings.Contains(got, "lines omitted") {
		t.Error("Long output should be truncated")
	}
	if !strings.Contains(got, "line 0 content") {
		t.Error("Should preserve head")
	}
	if !strings.Contains(got, "line 499 content") {
		t.Error("Should preserve tail")
	}
}

func TestGenericShortOutputUnchanged(t *testing.T) {
	f := &GenericFilter{}
	input := "short output\nline 2\nline 3"
	got := f.Apply(input, 0)
	if strings.Contains(got, "omitted") {
		t.Error("Short output should not be truncated")
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
		{"no colors", "no colors"},
		{"\x1b[38;5;196mextended\x1b[0m", "extended"},
		{"", ""},
	}
	for _, tt := range tests {
		got := StripANSI(tt.input)
		if got != tt.want {
			t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenericMixedANSIAndBlankLines(t *testing.T) {
	f := &GenericFilter{}
	input := "\x1b[32mheader\x1b[0m\n\n\n\n\n\x1b[31merror\x1b[0m"
	got := f.Apply(input, 0)
	if strings.Contains(got, "\x1b") {
		t.Error("ANSI should be stripped")
	}
	if strings.Contains(got, "\n\n\n") {
		t.Error("Blank lines should be collapsed")
	}
}
