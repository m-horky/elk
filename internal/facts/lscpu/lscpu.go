package lscpu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/m-horky/elk/internal/exec"
)

// ErrNoData reports that lscpu produced valid JSON without any fields.
var ErrNoData = errors.New("lscpu returned no data")

// lscpuOutput mirrors the top-level document emitted by lscpu --json.
type lscpuOutput struct {
	Lscpu []lscpuEntry `json:"lscpu"`
}

// lscpuEntry mirrors one field entry in lscpu JSON output.
type lscpuEntry struct {
	Field    string       `json:"field"`
	Data     *string      `json:"data"`
	Children []lscpuEntry `json:"children"`
}

// Collect runs lscpu --json and returns its normalized field values.
//
// Only JSON output is supported. A nil environment selects the runner's fixed
// minimal environment. Collect returns a nil map when the invocation fails,
// exits nonzero, produces truncated output, yields malformed JSON, or reports
// no fields at all; the last case reports ErrNoData.
func Collect(ctx context.Context, runner exec.Runner, path string) (map[string]string, error) {
	result, err := runner.Run(ctx, exec.Command{
		Path: path,
		Args: []string{"--json"},
	})
	if err != nil {
		return nil, fmt.Errorf("run %s --json: %w", path, err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("run %s --json: exit code %d", path, result.ExitCode)
	}

	if result.StdoutTruncated {
		return nil, fmt.Errorf("run %s --json: output truncated", path)
	}

	facts, err := parse(result.Stdout)
	if err != nil {
		return nil, err
	}

	if len(facts) == 0 {
		return nil, ErrNoData
	}

	return facts, nil
}

// parse decodes lscpu JSON output into normalized field values.
// Entries without a field name or data are skipped, while their children are
// still visited.
func parse(data []byte) (map[string]string, error) {
	var output lscpuOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parse lscpu JSON: %w", err)
	}

	facts := make(map[string]string)
	collectEntries(output.Lscpu, facts)

	return facts, nil
}

// collectEntries recursively stores entries that carry a field name and data.
func collectEntries(entries []lscpuEntry, facts map[string]string) {
	for _, entry := range entries {
		if entry.Field != "" && entry.Data != nil {
			key := normalizeKey(entry.Field)
			if key != "" {
				facts[key] = *entry.Data
			}
		}

		collectEntries(entry.Children, facts)
	}
}

// normalizeKey converts an lscpu field name to its fact key form by trimming
// whitespace, removing a trailing colon, lowercasing, and replacing spaces
// with underscores.
func normalizeKey(field string) string {
	key := strings.TrimSpace(field)
	key = strings.TrimSuffix(key, ":")
	key = strings.ToLower(key)

	return strings.ReplaceAll(key, " ", "_")
}
