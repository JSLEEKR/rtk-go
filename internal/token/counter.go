// Package token provides token counting heuristics for LLM usage estimation.
package token

import (
	"math"
	"unicode/utf8"
)

// Count estimates the number of LLM tokens in a string using the chars/4 heuristic.
// C3 fix: Uses rune count (characters) instead of byte count for correct Unicode handling.
func Count(s string) int {
	if len(s) == 0 {
		return 0
	}
	return int(math.Ceil(float64(utf8.RuneCountInString(s)) / 4.0))
}

// CountBytes estimates the number of LLM tokens from a byte slice.
func CountBytes(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	return int(math.Ceil(float64(utf8.RuneCount(b)) / 4.0))
}

// Savings calculates the percentage of tokens saved.
// Returns 0 if inputTokens is 0.
func Savings(inputTokens, outputTokens int) float64 {
	if inputTokens == 0 {
		return 0
	}
	return float64(inputTokens-outputTokens) / float64(inputTokens) * 100
}

// Stats holds token counting statistics for a single operation.
type Stats struct {
	InputTokens  int
	OutputTokens int
}

// SavingsPercent returns the percentage saved.
func (s Stats) SavingsPercent() float64 {
	return Savings(s.InputTokens, s.OutputTokens)
}

// Saved returns the number of tokens saved, clamped to 0 (L3 fix).
func (s Stats) Saved() int {
	saved := s.InputTokens - s.OutputTokens
	if saved < 0 {
		return 0
	}
	return saved
}
