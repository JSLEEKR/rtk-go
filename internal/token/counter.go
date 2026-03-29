// Package token provides token counting heuristics for LLM usage estimation.
package token

import "math"

// Count estimates the number of LLM tokens in a string using the chars/4 heuristic.
// This is an approximation that trades accuracy for zero-overhead measurement.
func Count(s string) int {
	if len(s) == 0 {
		return 0
	}
	return int(math.Ceil(float64(len(s)) / 4.0))
}

// CountBytes estimates the number of LLM tokens from a byte slice.
func CountBytes(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	return int(math.Ceil(float64(len(b)) / 4.0))
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

// Saved returns the number of tokens saved.
func (s Stats) Saved() int {
	return s.InputTokens - s.OutputTokens
}
