package filter

import (
	"strings"
	"testing"
)

func TestBuildFilterName(t *testing.T) {
	f := &BuildFilter{}
	if f.Name() != "build" {
		t.Errorf("Name() = %q, want %q", f.Name(), "build")
	}
}

func TestBuildFilterMatch(t *testing.T) {
	f := &BuildFilter{}
	tests := []struct {
		cmd  string
		args []string
		want bool
	}{
		{"go", []string{"build"}, true},
		{"go", []string{"build", "./..."}, true},
		{"go", []string{"vet"}, true},
		{"go", []string{"install"}, true},
		{"go", []string{"test"}, false},
		{"cargo", []string{"build"}, true},
		{"cargo", []string{"clippy"}, true},
		{"cargo", []string{"check"}, true},
		{"cargo", []string{"run"}, false},
		{"make", nil, true},
		{"cmake", nil, true},
		{"tsc", nil, true},
	}
	for _, tt := range tests {
		got := f.Match(tt.cmd, tt.args)
		if got != tt.want {
			t.Errorf("Match(%q, %v) = %v, want %v", tt.cmd, tt.args, got, tt.want)
		}
	}
}

func TestBuildEmptySuccess(t *testing.T) {
	f := &BuildFilter{}
	got := f.Apply("", 0, nil)
	if got != "build succeeded" {
		t.Errorf("Expected 'build succeeded', got: %q", got)
	}
}

func TestBuildEmptyFailure(t *testing.T) {
	f := &BuildFilter{}
	got := f.Apply("", 1, nil)
	if got != "build failed (no output)" {
		t.Errorf("Expected 'build failed', got: %q", got)
	}
}

func TestBuildStripsCompileProgress(t *testing.T) {
	f := &BuildFilter{}
	input := `   Compiling serde v1.0.0
   Compiling tokio v1.0.0
   Compiling myapp v0.1.0
    Finished dev [unoptimized + debuginfo] target(s) in 30.5s`

	got := f.Apply(input, 0, nil)
	if strings.Contains(got, "Compiling serde") {
		t.Error("Should strip Compiling lines")
	}
	if !strings.Contains(got, "build succeeded") {
		t.Errorf("Expected success message, got: %q", got)
	}
	if !strings.Contains(got, "progress lines hidden") {
		t.Errorf("Expected hidden count, got: %q", got)
	}
}

func TestBuildKeepsErrors(t *testing.T) {
	f := &BuildFilter{}
	input := `   Compiling myapp v0.1.0
error[E0308]: mismatched types
  --> src/main.rs:10:5
   |
10 |     let x: u32 = "hello";
   |                   ^^^^^^^ expected u32, found &str
error: aborting due to previous error`

	got := f.Apply(input, 1, nil)
	if !strings.Contains(got, "error") {
		t.Errorf("Should keep error lines, got: %q", got)
	}
	if !strings.Contains(got, "mismatched types") {
		t.Errorf("Should keep error detail, got: %q", got)
	}
}

func TestBuildKeepsWarnings(t *testing.T) {
	f := &BuildFilter{}
	input := `   Compiling myapp v0.1.0
warning: unused variable: x
  --> src/main.rs:5:9
   |
5  |     let x = 1;
   |         ^ help: consider prefixing with an underscore: _x
    Finished dev [unoptimized + debuginfo] target(s) in 1.5s`

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "warning") {
		t.Errorf("Should keep warnings, got: %q", got)
	}
	if !strings.Contains(got, "1 warning") {
		t.Errorf("Expected warning count, got: %q", got)
	}
}

func TestBuildStripsMakeEnter(t *testing.T) {
	f := &BuildFilter{}
	input := `make[1]: Entering directory '/tmp/build'
gcc -c main.c -o main.o
make[1]: Leaving directory '/tmp/build'`

	got := f.Apply(input, 0, nil)
	if strings.Contains(got, "Entering directory") {
		t.Error("Should strip make enter/leave lines")
	}
}

func TestBuildMultipleErrors(t *testing.T) {
	f := &BuildFilter{}
	input := `error: cannot find type
error: undefined reference
error: linking failed`

	got := f.Apply(input, 1, nil)
	if !strings.Contains(got, "3 error(s)") {
		t.Errorf("Expected error count, got: %q", got)
	}
}

func TestBuildGoVet(t *testing.T) {
	f := &BuildFilter{}
	if !f.Match("go", []string{"vet", "./..."}) {
		t.Error("Should match go vet")
	}
}

func TestBuildSuccessNoWarnings(t *testing.T) {
	f := &BuildFilter{}
	input := `   Compiling myapp v0.1.0
    Finished dev target(s) in 1.0s`

	got := f.Apply(input, 0, nil)
	if !strings.Contains(got, "build succeeded") {
		t.Errorf("Expected success, got: %q", got)
	}
	if strings.Contains(got, "warning") {
		t.Error("Should not mention warnings when there are none")
	}
}
