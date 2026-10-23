package virtwhat

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-horky/elk/internal/exec"
)

// Virt describes the virtualization facts reported by virt-what.
//
// A non-nil IsGuest reports whether the system runs as a guest. A non-nil
// HostType contains the detected host or virtualization type, joined with
// commas when virt-what reports multiple values.
type Virt struct {
	// IsGuest reports whether the system is a virtualization guest.
	IsGuest *bool
	// HostType contains the detected host or virtualization type.
	HostType *string
}

// Collect runs virt-what at the supplied path and parses its output.
func Collect(ctx context.Context, runner exec.Runner, path string) (Virt, error) {
	result, err := runner.Run(ctx, exec.Command{Path: path})
	if err != nil {
		return Virt{}, fmt.Errorf("run virt-what: %w", err)
	}

	if result.ExitCode != 0 {
		return Virt{}, fmt.Errorf("run virt-what: exit code %d", result.ExitCode)
	}

	return parse(result.Stdout), nil
}

// parse converts virt-what output into virtualization facts.
//
// Empty lines are ignored, and the remaining trimmed lines are joined with a
// comma and a space. Empty output means the system is not a guest.
//
// A joined value containing "dom0" denotes a host,
// so IsGuest is false while HostType keeps the reported value.
func parse(output []byte) Virt {
	lines := strings.Split(string(output), "\n")
	values := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		values = append(values, line)
	}

	if len(values) == 0 {
		isGuest := false
		hostType := ""

		return Virt{IsGuest: &isGuest, HostType: &hostType}
	}

	hostType := strings.Join(values, ", ")
	isGuest := !strings.Contains(hostType, "dom0")

	return Virt{IsGuest: &isGuest, HostType: &hostType}
}
