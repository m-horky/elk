package constants

import "testing"

// TestPath verifies path resolution uses a non-empty environment override.
//
// Given an environment variable that is set, when a path is resolved,
// then the configured value is returned.
func TestPath(t *testing.T) {
	t.Setenv(string(RHSMConfigPath), "/configured/path")

	if got := Path(RHSMConfigPath); got != "/configured/path" {
		t.Errorf("Path() = %q, want /configured/path", got)
	}
}

// TestPathFallsBackToDefault verifies path resolution falls back to the system default.
//
// Given an environment variable that is empty, when a path is resolved,
// then the system default is returned.
func TestPathFallsBackToDefault(t *testing.T) {
	t.Setenv(string(RHSMConfigPath), "")

	if got := Path(RHSMConfigPath); got != "/etc/rhsm/rhsm.conf" {
		t.Errorf("Path() = %q, want /etc/rhsm/rhsm.conf", got)
	}
}
