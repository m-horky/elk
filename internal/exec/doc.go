// Package exec provides an injectable abstraction for running commands and
// capturing bounded output.
//
// Callers pass a context and a command configuration to a Runner. The context
// bounds the command's lifetime; cancellation or deadline expiry terminates the
// process and returns an error wrapping the context error. A nonzero process
// exit is reported in Result.ExitCode, while start and other execution failures
// are returned as errors.
//
// Stdout and Stderr are each limited to maxOutputSize bytes.
//
// Command.Path identifies the executable, while Command.Args contains
// only its additional arguments. Command.Dir independently sets the process
// working directory.
//
//	runner := OSRunner{}
//	result, err := runner.Run(context.Background(), Command{
//		Path: "/usr/bin/dnf",
//		Args: []string{"update"},
//	})
//
//	if err != nil {
//		// Handle a lookup, start, or context failure.
//	}
//	if result.ExitCode != 0 {
//		// Handle a nonzero process exit.
//	}
package exec
