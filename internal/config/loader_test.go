package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/m-horky/elk/internal/config/rhsm"
	internalfs "github.com/m-horky/elk/internal/fs"
)

// TestLoadHasDefaults verifies Load returns at least the default configuration.
//
// Given all files and directories are absent, when Load resolves the configuration,
// then it returns the default configuration.
func TestLoadHasDefaults(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	partial, err := newPartialFromDefaults()
	if err != nil {
		t.Fatal(err)
	}

	want, err := partial.resolve()
	if err != nil {
		t.Fatal(err)
	}

	got, err := Load(
		internalfs.Filesystem{},
		filepath.Join(root, "main.toml"),
		filepath.Join(root, "drop-ins"),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Load() mismatch (-want +got): %s", diff)
	}
}

// TestLoadReadsMainAndDropIns verifies documented configuration precedence.
//
// Given defaults, a main file, and the drop-ins, when Load resolves the configuration,
// then later layers win.
func TestLoadReadsMainAndDropIns(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "elk.conf")

	dropInsDir := filepath.Join(root, "elk.conf.d")
	if err := os.Mkdir(dropInsDir, 0o700); err != nil {
		t.Fatal(err)
	}

	write(t, mainPath, "[http.timeout]\nconnect=10\nrequest=10\nidle=10\n")
	write(t, filepath.Join(dropInsDir, "20.conf"), "[http.timeout]\nrequest=20\nidle=20\n")
	write(t, filepath.Join(dropInsDir, "30.conf"), "[http.timeout]\nidle=30\n")
	write(t, filepath.Join(dropInsDir, "40.conf"), "[http.timeout]\n")
	write(t, filepath.Join(dropInsDir, "50.conf"), "")

	got, err := Load(internalfs.Filesystem{}, mainPath, dropInsDir, nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.HTTP.Timeout.Connect != 10*time.Second {
		t.Errorf("connect timeout = %s, want 10", got.HTTP.Timeout.Connect)
	}

	if got.HTTP.Timeout.Request != 20*time.Second {
		t.Errorf("request timeout = %s, want 20", got.HTTP.Timeout.Request)
	}

	if got.HTTP.Timeout.Idle != 30*time.Second {
		t.Errorf("idle timeout = %s, want 30", got.HTTP.Timeout.Idle)
	}
}

// TestLoadAppliesRHSMFallback verifies RHSM settings are merged under the defaults.
//
// Given a resolved RHSM fallback and no native configuration, when Load
// resolves the configuration, then endpoint settings come from RHSM.
func TestLoadAppliesRHSMFallback(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	fallback := getRHSMconf()

	got, err := Load(
		internalfs.Filesystem{},
		filepath.Join(root, "main.toml"),
		filepath.Join(root, "drop-ins"),
		&fallback,
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.API.Subscriptions.URI.String() != "https://satellite.example.org:8443/subscription" {
		t.Errorf("subscriptions URI = %q, want RHSM-derived endpoint", got.API.Subscriptions.URI.String())
	}

	if got.API.Insights.Ingress.URI.String() !=
		"https://satellite.example.org:8443/redhat_access/r/insights/platform/ingress/v1" {
		t.Errorf("insights ingress URI = %q, want RHSM-derived endpoint", got.API.Insights.Ingress.URI.String())
	}

	if got.HTTP.Proxy.URI.String() != "http://rhsm.example.org:3128" {
		t.Errorf("proxy URI = %q, want RHSM-derived proxy", got.HTTP.Proxy.URI.String())
	}
}

// TestLoadSkipsRHSMFallbackWhenDisabled verifies the compatibility flag gates the fallback.
//
// Given a resolved RHSM fallback and a main file that disables legacy
// interpretation, when Load resolves the configuration, then endpoint settings
// keep their defaults.
func TestLoadSkipsRHSMFallbackWhenDisabled(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "elk.conf")
	write(t, mainPath, "[compatibility]\ninterpret-legacy-configurations = false\n")

	fallback := getRHSMconf()

	got, err := Load(internalfs.Filesystem{}, mainPath, filepath.Join(root, "missing"), &fallback)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.API.Subscriptions.URI.String() != "https://subscription.rhsm.redhat.com/subscription" {
		t.Errorf("subscriptions URI = %q, want embedded default", got.API.Subscriptions.URI.String())
	}
}

// TestLoadLetsNativeOverrideRHSM verifies native files override the RHSM fallback.
//
// Given a resolved RHSM fallback and a main file with a subscriptions URI,
// when Load resolves the configuration, then the native URI wins.
func TestLoadLetsNativeOverrideRHSM(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "elk.conf")
	write(t, mainPath, "[api.subscriptions]\nuri = \"https://native.example.org/subscription\"\n")

	fallback := getRHSMconf()

	got, err := Load(internalfs.Filesystem{}, mainPath, filepath.Join(root, "missing"), &fallback)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.API.Subscriptions.URI.String() != "https://native.example.org/subscription" {
		t.Errorf("subscriptions URI = %q, want native endpoint", got.API.Subscriptions.URI.String())
	}
}

// TestDisableProxy verifies an override with empty proxy URI disables an inherited proxy.
//
// Given an RHSM proxy and an explicitly empty proxy URI in the main file, when Load resolves,
// then the resolved proxy URI is empty.
func TestDisableProxy(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "elk.conf")
	write(t, mainPath, "[http.proxy]\nuri = \"\"\n")

	fallback := getRHSMconf()

	got, err := Load(internalfs.Filesystem{}, mainPath, filepath.Join(root, "missing"), &fallback)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.HTTP.Proxy.URI.String() != "" {
		t.Errorf("proxy URI = %q, want empty", got.HTTP.Proxy.URI.String())
	}
}

// TestLoadRejectsExplicitEmptyEndpoint verifies empty supplied URIs are not silently suppressed.
//
// Given an explicitly empty endpoint in the main file, when Load resolves,
// then it reports an invalid configuration.
func TestLoadRejectsExplicitEmptyEndpoint(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "elk.conf")
	write(t, mainPath, "[api.subscriptions]\nuri = \"\"\n")

	_, err := Load(internalfs.Filesystem{}, mainPath, filepath.Join(root, "missing"), nil)
	if err == nil {
		t.Fatal("Load() error = nil")
	}
}

// TestLoadRejectsInvalidTimeout verifies out-of-range timeouts are rejected.
//
// Given a main file with a negative connect timeout, when Load resolves,
// then it reports an invalid configuration.
func TestLoadRejectsInvalidTimeout(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "elk.conf")
	write(t, mainPath, "[http.timeout]\nconnect = -1\n")

	_, err := Load(internalfs.Filesystem{}, mainPath, filepath.Join(root, "missing"), nil)
	if err == nil {
		t.Fatal("Load() error = nil")
	}
}

// getRHSMconf returns deterministic legacy values for native-merge tests.
func getRHSMconf() rhsm.Config {
	return rhsm.Config{
		Server: rhsm.Server{
			Hostname:      "satellite.example.org",
			Prefix:        "/subscription",
			Port:          8443,
			TLSVerify:     true,
			ProxyHostname: "rhsm.example.org",
			ProxyScheme:   "http",
			ProxyPort:     3128,
			ProxyUser:     "",
			ProxyPassword: "",
			NoProxy:       "",
		},
		RHSM: rhsm.RHSM{RepoCACert: "/legacy/ca.pem", BaseURL: "https://cdn.example.org"},
	}
}

// write writes contents into a file on disk.
func write(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write to disk: %v", err)
	}
}
