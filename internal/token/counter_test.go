package token

import (
	"strings"
	"testing"
)

func TestCount(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"single char", "a", 1},
		{"four chars", "abcd", 1},
		{"five chars", "abcde", 2},
		{"eight chars", "abcdefgh", 2},
		{"twelve chars", "abcdefghijkl", 3},
		{"thirteen chars", "abcdefghijklm", 4},
		{"hello world", "hello world", 3},
		{"newlines count", "a\nb\nc\n", 2},
		{"unicode chars", "héllo wörld", 4}, // bytes, not runes
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Count(tt.input)
			if got != tt.want {
				t.Errorf("Count(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestCountBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int
	}{
		{"nil", nil, 0},
		{"empty", []byte{}, 0},
		{"four bytes", []byte("abcd"), 1},
		{"five bytes", []byte("abcde"), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountBytes(tt.input)
			if got != tt.want {
				t.Errorf("CountBytes(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestSavings(t *testing.T) {
	tests := []struct {
		name   string
		input  int
		output int
		want   float64
	}{
		{"zero input", 0, 0, 0},
		{"no savings", 100, 100, 0},
		{"50% savings", 100, 50, 50},
		{"90% savings", 1000, 100, 90},
		{"100% savings", 100, 0, 100},
		{"negative savings", 100, 150, -50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Savings(tt.input, tt.output)
			if got != tt.want {
				t.Errorf("Savings(%d, %d) = %f, want %f", tt.input, tt.output, got, tt.want)
			}
		})
	}
}

func TestStats(t *testing.T) {
	s := Stats{InputTokens: 1000, OutputTokens: 100}
	if s.SavingsPercent() != 90 {
		t.Errorf("SavingsPercent() = %f, want 90", s.SavingsPercent())
	}
	if s.Saved() != 900 {
		t.Errorf("Saved() = %d, want 900", s.Saved())
	}
}

func TestCountLargeInput(t *testing.T) {
	large := strings.Repeat("x", 10000)
	got := Count(large)
	if got != 2500 {
		t.Errorf("Count(10000 chars) = %d, want 2500", got)
	}
}
