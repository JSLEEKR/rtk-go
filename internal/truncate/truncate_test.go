package truncate

import (
	"fmt"
	"strings"
	"testing"
)

func makeLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	return strings.Join(lines, "\n")
}

func TestSmartNoTruncation(t *testing.T) {
	input := makeLines(10)
	got := Smart(input, 20)
	if got != input {
		t.Error("Smart should not truncate when under limit")
	}
}

func TestSmartExactLimit(t *testing.T) {
	input := makeLines(20)
	got := Smart(input, 20)
	if got != input {
		t.Error("Smart should not truncate at exact limit")
	}
}

func TestSmartTruncates(t *testing.T) {
	input := makeLines(100)
	got := Smart(input, 30)
	if !strings.Contains(got, "... [") {
		t.Error("Smart should contain omission marker")
	}
	if !strings.Contains(got, "lines omitted") {
		t.Error("Smart should indicate lines omitted")
	}
	if !strings.Contains(got, "line 1") {
		t.Error("Smart should keep first line")
	}
	if !strings.Contains(got, "line 100") {
		t.Error("Smart should keep last line")
	}
}

func TestSmartPreservesHead(t *testing.T) {
	input := makeLines(100)
	got := Smart(input, 30)
	// keepHead = 20, so line 20 should be present
	if !strings.Contains(got, "line 20") {
		t.Error("Smart should preserve head lines")
	}
}

func TestSmartPreservesTail(t *testing.T) {
	input := makeLines(100)
	got := Smart(input, 30)
	// keepTail = 10, so line 91 should be present
	if !strings.Contains(got, "line 91") {
		t.Error("Smart should preserve tail lines")
	}
}

func TestSmartOmittedCount(t *testing.T) {
	input := makeLines(100)
	got := Smart(input, 30)
	// 100 - 20 - 10 = 70 omitted
	if !strings.Contains(got, "70 lines omitted") {
		t.Errorf("Expected 70 lines omitted, got:\n%s", got)
	}
}

func TestSmartZeroMaxLines(t *testing.T) {
	input := makeLines(300)
	got := Smart(input, 0)
	// Should use default (200)
	if !strings.Contains(got, "lines omitted") {
		t.Error("Zero maxLines should use default and truncate 300 lines")
	}
}

func TestHeadNoTruncation(t *testing.T) {
	input := makeLines(5)
	got := Head(input, 10)
	if got != input {
		t.Error("Head should not truncate when under limit")
	}
}

func TestHeadTruncates(t *testing.T) {
	input := makeLines(100)
	got := Head(input, 10)
	if !strings.Contains(got, "line 10") {
		t.Error("Head should keep line 10")
	}
	if strings.Contains(got, "line 11\n") {
		t.Error("Head should not keep line 11")
	}
	if !strings.Contains(got, "90 more lines") {
		t.Error("Head should show remaining count")
	}
}

func TestTailNoTruncation(t *testing.T) {
	input := makeLines(5)
	got := Tail(input, 10)
	if got != input {
		t.Error("Tail should not truncate when under limit")
	}
}

func TestTailTruncates(t *testing.T) {
	input := makeLines(100)
	got := Tail(input, 10)
	if !strings.Contains(got, "line 100") {
		t.Error("Tail should keep last line")
	}
	if !strings.Contains(got, "line 91") {
		t.Error("Tail should keep line 91")
	}
	if !strings.Contains(got, "90 lines above") {
		t.Error("Tail should show lines above count")
	}
}

func TestHeadZeroMaxLines(t *testing.T) {
	input := makeLines(5)
	got := Head(input, 0)
	if got != input {
		t.Error("Head with 0 maxLines should return input unchanged")
	}
}

func TestTailZeroMaxLines(t *testing.T) {
	input := makeLines(5)
	got := Tail(input, 0)
	if got != input {
		t.Error("Tail with 0 maxLines should return input unchanged")
	}
}

func TestSmartEmptyInput(t *testing.T) {
	got := Smart("", 10)
	if got != "" {
		t.Error("Smart should handle empty input")
	}
}

func TestHeadEmptyInput(t *testing.T) {
	got := Head("", 10)
	if got != "" {
		t.Error("Head should handle empty input")
	}
}

func TestTailEmptyInput(t *testing.T) {
	got := Tail("", 10)
	if got != "" {
		t.Error("Tail should handle empty input")
	}
}
