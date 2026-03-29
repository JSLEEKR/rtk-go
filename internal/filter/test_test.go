package filter

import (
	"strings"
	"testing"
)

// --- GoTestFilter Tests ---

func TestGoTestFilterName(t *testing.T) {
	f := &GoTestFilter{}
	if f.Name() != "go-test" {
		t.Errorf("Name() = %q, want %q", f.Name(), "go-test")
	}
}

func TestGoTestFilterMatch(t *testing.T) {
	f := &GoTestFilter{}
	tests := []struct {
		cmd  string
		args []string
		want bool
	}{
		{"go", []string{"test"}, true},
		{"go", []string{"test", "./..."}, true},
		{"go", []string{"build"}, false},
		{"go", []string{}, false},
		{"python", []string{"test"}, false},
	}
	for _, tt := range tests {
		got := f.Match(tt.cmd, tt.args)
		if got != tt.want {
			t.Errorf("Match(%q, %v) = %v, want %v", tt.cmd, tt.args, got, tt.want)
		}
	}
}

func TestGoTestEmpty(t *testing.T) {
	f := &GoTestFilter{}
	got := f.Apply("", 0)
	if got != "no test output" {
		t.Errorf("Expected 'no test output', got: %q", got)
	}
}

func TestGoTestJSONAllPass(t *testing.T) {
	f := &GoTestFilter{}
	input := `{"Action":"run","Package":"myapp","Test":"TestA"}
{"Action":"output","Package":"myapp","Test":"TestA","Output":"=== RUN   TestA\n"}
{"Action":"pass","Package":"myapp","Test":"TestA","Elapsed":0.01}
{"Action":"run","Package":"myapp","Test":"TestB"}
{"Action":"pass","Package":"myapp","Test":"TestB","Elapsed":0.02}
{"Action":"pass","Package":"myapp","Elapsed":0.5}`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "PASS") {
		t.Errorf("Expected PASS, got: %q", got)
	}
	if !strings.Contains(got, "2 tests passed") {
		t.Errorf("Expected 2 passed, got: %q", got)
	}
}

func TestGoTestJSONWithFailures(t *testing.T) {
	f := &GoTestFilter{}
	input := `{"Action":"run","Package":"myapp","Test":"TestA"}
{"Action":"pass","Package":"myapp","Test":"TestA","Elapsed":0.01}
{"Action":"run","Package":"myapp","Test":"TestB"}
{"Action":"output","Package":"myapp","Test":"TestB","Output":"expected 1 got 2\n"}
{"Action":"fail","Package":"myapp","Test":"TestB","Elapsed":0.02}
{"Action":"fail","Package":"myapp","Elapsed":0.5}`

	got := f.Apply(input, 1)
	if !strings.Contains(got, "FAIL") {
		t.Errorf("Expected FAIL, got: %q", got)
	}
	if !strings.Contains(got, "1/2 tests failed") {
		t.Errorf("Expected 1/2 failed, got: %q", got)
	}
}

func TestGoTestVerboseAllPass(t *testing.T) {
	f := &GoTestFilter{}
	input := `=== RUN   TestA
--- PASS: TestA (0.01s)
=== RUN   TestB
--- PASS: TestB (0.02s)
ok  	myapp	0.5s`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "PASS") {
		t.Errorf("Expected PASS, got: %q", got)
	}
	if !strings.Contains(got, "2 tests passed") {
		t.Errorf("Expected 2 passed, got: %q", got)
	}
}

func TestGoTestVerboseWithFailure(t *testing.T) {
	f := &GoTestFilter{}
	input := `=== RUN   TestA
--- PASS: TestA (0.01s)
=== RUN   TestB
    main_test.go:15: expected 1 got 2
--- FAIL: TestB (0.02s)
FAIL	myapp	0.5s`

	got := f.Apply(input, 1)
	if !strings.Contains(got, "FAIL") {
		t.Errorf("Expected FAIL, got: %q", got)
	}
	if !strings.Contains(got, "1/2") {
		t.Errorf("Expected 1/2 ratio, got: %q", got)
	}
}

func TestGoTestSkipped(t *testing.T) {
	f := &GoTestFilter{}
	input := `{"Action":"run","Package":"myapp","Test":"TestA"}
{"Action":"pass","Package":"myapp","Test":"TestA","Elapsed":0.01}
{"Action":"run","Package":"myapp","Test":"TestB"}
{"Action":"skip","Package":"myapp","Test":"TestB","Elapsed":0}
{"Action":"pass","Package":"myapp","Elapsed":0.5}`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "1 skipped") {
		t.Errorf("Expected skipped count, got: %q", got)
	}
}

// --- PytestFilter Tests ---

func TestPytestFilterName(t *testing.T) {
	f := &PytestFilter{}
	if f.Name() != "pytest" {
		t.Errorf("Name() = %q, want %q", f.Name(), "pytest")
	}
}

func TestPytestFilterMatch(t *testing.T) {
	f := &PytestFilter{}
	tests := []struct {
		cmd  string
		args []string
		want bool
	}{
		{"pytest", nil, true},
		{"pytest", []string{"-v"}, true},
		{"python", []string{"-m", "pytest"}, true},
		{"python3", []string{"-m", "pytest"}, true},
		{"python", []string{"script.py"}, false},
		{"go", []string{"test"}, false},
	}
	for _, tt := range tests {
		got := f.Match(tt.cmd, tt.args)
		if got != tt.want {
			t.Errorf("Match(%q, %v) = %v, want %v", tt.cmd, tt.args, got, tt.want)
		}
	}
}

func TestPytestEmpty(t *testing.T) {
	f := &PytestFilter{}
	got := f.Apply("", 0)
	if got != "no test output" {
		t.Errorf("Expected 'no test output', got: %q", got)
	}
}

func TestPytestAllPassed(t *testing.T) {
	f := &PytestFilter{}
	input := `============================= test session starts ==============================
collected 5 items
tests/test_main.py .....                                                 [100%]
============================== 5 passed in 0.5s ===============================`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "5 passed") {
		t.Errorf("Expected passed summary, got: %q", got)
	}
}

func TestPytestWithFailures(t *testing.T) {
	f := &PytestFilter{}
	input := `============================= test session starts ==============================
collected 3 items
tests/test_main.py ..F                                                   [100%]
================================= FAILURES =================================
_________________________________ test_bad _________________________________

    def test_bad():
>       assert 1 == 2
E       assert 1 == 2

tests/test_main.py:10: AssertionError
=========================== short test summary info ============================
FAILED tests/test_main.py::test_bad
========================= 2 passed, 1 failed in 0.5s =========================`

	got := f.Apply(input, 1)
	if !strings.Contains(got, "1 failure") {
		t.Errorf("Expected failure count, got: %q", got)
	}
	if !strings.Contains(got, "assert 1 == 2") {
		t.Errorf("Expected failure detail, got: %q", got)
	}
}

func TestPytestSummaryExtraction(t *testing.T) {
	f := &PytestFilter{}
	input := `============================= test session starts ==============================
collected 100 items
tests/test_all.py ....................................                    [100%]
========================= 100 passed in 2.5s ==========================`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "100 passed") {
		t.Errorf("Expected summary, got: %q", got)
	}
	// Should be significantly smaller than input
	if len(got) >= len(input) {
		t.Error("Filtered output should be smaller")
	}
}

// --- NPMTestFilter Tests ---

func TestNPMTestFilterName(t *testing.T) {
	f := &NPMTestFilter{}
	if f.Name() != "npm-test" {
		t.Errorf("Name() = %q, want %q", f.Name(), "npm-test")
	}
}

func TestNPMTestFilterMatch(t *testing.T) {
	f := &NPMTestFilter{}
	tests := []struct {
		cmd  string
		args []string
		want bool
	}{
		{"npm", []string{"test"}, true},
		{"npx", []string{"jest"}, true},
		{"pnpm", []string{"test"}, true},
		{"yarn", []string{"test"}, true},
		{"jest", nil, true},
		{"vitest", nil, true},
		{"npm", []string{"install"}, false},
	}
	for _, tt := range tests {
		got := f.Match(tt.cmd, tt.args)
		if got != tt.want {
			t.Errorf("Match(%q, %v) = %v, want %v", tt.cmd, tt.args, got, tt.want)
		}
	}
}

func TestNPMTestEmpty(t *testing.T) {
	f := &NPMTestFilter{}
	got := f.Apply("", 0)
	if got != "no test output" {
		t.Errorf("Expected 'no test output', got: %q", got)
	}
}

func TestNPMTestAllPassed(t *testing.T) {
	f := &NPMTestFilter{}
	input := `> myapp@1.0.0 test
> jest

PASS  tests/main.test.js
  ✓ should work (5ms)

Tests: 1 passed, 1 total
Test Suites: 1 passed, 1 total`

	got := f.Apply(input, 0)
	if !strings.Contains(got, "1 passed") {
		t.Errorf("Expected passed count, got: %q", got)
	}
}

func TestNPMTestWithFailure(t *testing.T) {
	f := &NPMTestFilter{}
	input := `> myapp@1.0.0 test
> jest

FAIL  tests/main.test.js
  ● should work

    expect(received).toBe(expected)
    Expected: 2
    Received: 1

Tests: 1 failed, 1 total
Test Suites: 1 failed, 1 total`

	got := f.Apply(input, 1)
	if !strings.Contains(got, "1 failed") {
		t.Errorf("Expected failure summary, got: %q", got)
	}
}

func TestNPMTestPassedNoOutput(t *testing.T) {
	f := &NPMTestFilter{}
	input := `> myapp@1.0.0 test
> jest --silent

` // no summary lines, but exit 0

	got := f.Apply(input, 0)
	if got != "all tests passed" {
		t.Errorf("Expected 'all tests passed', got: %q", got)
	}
}
