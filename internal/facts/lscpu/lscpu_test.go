package lscpu

import (
	"context"
	"io/fs"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/m-horky/elk/internal/exec"
)

const testPath = "/usr/bin/lscpu"

// fakeRunner is a function-backed exec.Runner used to drive Collect in tests.
type fakeRunner struct {
	result       exec.Result
	err          error
	command      exec.Command
	invocations  int
	contextKey   contextKey
	contextValue any
}

// Run records the requested context and command and returns the configured result.
func (runner *fakeRunner) Run(ctx context.Context, cmd exec.Command) (exec.Result, error) {
	runner.command = cmd
	runner.contextValue = ctx.Value(runner.contextKey)
	runner.invocations++

	return runner.result, runner.err
}

// contextKey is the key type for values attached to test contexts.
type contextKey struct{}

// TestCollect verifies that Collect uses JSON output with a default environment.
//
// Given a runner returning valid lscpu JSON, when Collect runs, then it returns nested,
// null-handled, normalized facts after a single invocation with the expected arguments.
func TestCollect(t *testing.T) {
	t.Parallel()

	fixture := readFixture(t, "lscpu.json")
	runner := &fakeRunner{result: exec.Result{Stdout: fixture}}

	got, err := Collect(context.Background(), runner, testPath)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	want := map[string]string{
		"architecture":        "x86_64",
		"cpu_op-mode(s)":      "32-bit, 64-bit",
		"on-line_cpu(s)_list": "0-15",
		"model_name":          "Intel(R) Core(TM) i7-1365U",
		"vendor_id":           "GenuineIntel",
		"l1d":                 "640 KiB",
		"l2":                  "24 MiB",
		"nested":              "yes",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Collect() facts mismatch (-want +got):\n%s", diff)
	}

	if runner.invocations != 1 {
		t.Errorf("Collect() invocations = %d, want 1", runner.invocations)
	}
}

// TestNormalizeKey verifies field name normalization.
//
// Given lscpu field names with whitespace, trailing colons, spaces, and punctuation,
// when normalizeKey runs, then it produces subscription-manager-compatible keys.
func TestNormalizeKey(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"trailing colon":         {input: "CPU op-mode(s):", want: "cpu_op-mode(s)"},
		"surrounding whitespace": {input: "  CPU(s)  ", want: "cpu(s)"},
		"multiple words":         {input: "On-line CPU(s) list:", want: "on-line_cpu(s)_list"},
		"hyphen and parentheses": {input: "Socket(s):", want: "socket(s)"},
		"no colon":               {input: "Core(s) per socket", want: "core(s)_per_socket"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeKey(test.input); got != test.want {
				t.Errorf("normalizeKey(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

// readFixture reads a file from the package testdata directory.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := fs.ReadFile(os.DirFS("testdata"), name)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", name, err)
	}

	return data
}
