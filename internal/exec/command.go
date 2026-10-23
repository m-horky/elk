package exec

import "context"

// Runner executes a command from its configuration and returns its captured result.
// The context bounds the command's lifetime: cancellation or deadline expiry
// terminates the process and is reported as an error wrapping the context error.
type Runner interface {
	Run(ctx context.Context, cmd Command) (Result, error)
}

// Command contains the executable path, its arguments, environment, and working directory.
type Command struct {
	Path string   // path to the binary (prefer absolute)
	Args []string // arguments passed to the binary
	Env  []string // execution environment; leave nil for the minimal default
	Dir  string   // execution working directory; leave empty for CWD
}

// Result contains captured command output, the process exit code, and truncation status.
// Stdout and Stderr are each limited to maxOutputSize bytes.
type Result struct {
	Stdout          []byte
	Stderr          []byte
	ExitCode        int
	StdoutTruncated bool
	StderrTruncated bool
}
