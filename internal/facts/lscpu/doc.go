// Package lscpu collects CPU topology facts from the lscpu binary.
//
// Collect invokes lscpu --json. Returned keys are normalized lscpu field names
// without a prefix, for example "cpu(s)" or "on-line_cpu(s)_list".
//
//	facts, _ := lscpu.Collect(context.Background(), exec.OSRunner{}, "/usr/bin/lscpu")
//	logicalCPUs := facts["cpu(s)"]
package lscpu
