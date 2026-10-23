package exec

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// OSRunner runs commands through the operating system process API.
type OSRunner struct{}

// Compile-time validation that OSRunner implements Runner.
var _ Runner = OSRunner{}

// Run starts the configured command and captures its bounded output.
// A nonzero process exit is reported in Result; errors are reserved for start,
// other execution failures, and context cancellation or deadline expiry.
func (OSRunner) Run(ctx context.Context, cmd Command) (Result, error) {
	if cmd.Path == "" {
		return Result{}, errors.New("run command: executable path is empty")
	}

	if !strings.HasPrefix(cmd.Path, "/") {
		slog.Debug("prefer absolute paths over relative", "binary", cmd.Path)
	}

	// Do not spawn a process that is already known to be obsolete.
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("run command %q: %w", cmd.Path, err)
	}

	// Set an output limit for each process stream.
	stdout := limitedBuffer{limit: maxOutputSize}
	stderr := limitedBuffer{limit: maxOutputSize}

	// A nil environment selects the safe environment defaults.
	environment := cmd.Env
	if environment == nil {
		environment = []string{
			"PATH=/usr/sbin:/usr/bin:/sbin:/bin",
			"LC_ALL=C.UTF-8",
			"LANG=C.UTF-8",
			"LANGUAGE=C.UTF-8",
		}
	}

	process := exec.CommandContext(ctx, cmd.Path, cmd.Args...) // #nosec G204: path and args are caller-supplied.
	process.Env = environment
	process.Dir = cmd.Dir
	process.Stdout = &stdout
	process.Stderr = &stderr
	process.WaitDelay = time.Second // Limit how long Wait waits for pipes.

	// Run waits for the process and its output copying to finish.
	runErr := process.Run()

	// Read the buffers only after the command has finished writing to them.
	result := Result{
		Stdout:          stdout.buffer.Bytes(),
		Stderr:          stderr.buffer.Bytes(),
		StdoutTruncated: stdout.truncated,
		StderrTruncated: stderr.truncated,
	}

	if runErr == nil {
		return result, nil
	}

	// Cancellation usually surfaces as a kill-induced ExitError; report the context error instead.
	if ctx.Err() != nil {
		return result, fmt.Errorf("run command %q: %w", cmd.Path, ctx.Err())
	}

	// A nonzero exit is a command result; other wait failures are execution errors.
	if exitErr, ok := errors.AsType[*exec.ExitError](runErr); ok {
		result.ExitCode = exitErr.ExitCode()

		return result, nil
	}

	return result, fmt.Errorf("run command %q: %w", cmd.Path, runErr)
}
