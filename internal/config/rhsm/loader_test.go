package rhsm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	internalfs "github.com/m-horky/elk/internal/fs"
)

// TestLoadHasDefaults verifies Load returns at least the default configuration.
//
// Given rhsm.conf is absent, when Load resolves the configuration,
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

	got, err := Load(internalfs.Filesystem{}, filepath.Join(root, "rhsm.conf"))
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Load() mismatch (-want +got): %s", diff)
	}
}

// TestLoadReadsMain verifies supplied values override the defaults.
//
// Given a rhsm.conf with a server hostname, when Load resolves the
// configuration, then the resolved server hostname comes from the file.
func TestLoadReadsMain(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "rhsm.conf")

	write(t, mainPath, "[server]\nhostname = example.org\n")

	got, err := Load(internalfs.Filesystem{}, mainPath)
	if err != nil {
		t.Fatal(err)
	}

	if got.Server.Hostname != "example.org" {
		t.Errorf("Load() hostname = %q, want %q", got.Server.Hostname, "example.org")
	}
}

// TestLoadReadsBaseURL verifies the content delivery base URL is loaded.
//
// Given a rhsm.conf with a rhsm baseurl, when Load resolves the
// configuration, then the resolved base URL comes from the file.
func TestLoadReadsBaseURL(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "rhsm.conf")

	write(t, mainPath, "[rhsm]\nbaseurl = https://cdn.example.org\n")

	got, err := Load(internalfs.Filesystem{}, mainPath)
	if err != nil {
		t.Fatal(err)
	}

	if got.RHSM.BaseURL != "https://cdn.example.org" {
		t.Errorf("Load() baseurl = %q, want %q", got.RHSM.BaseURL, "https://cdn.example.org")
	}
}

// TestInterpolation verifies the CA directory interpolation in certificate paths.
//
// Given a repo_ca_cert referencing %(ca_cert_dir)s, when the configuration is
// resolved, then the placeholder is replaced with the configured directory.
func TestInterpolation(t *testing.T) {
	partial, err := newPartialFromDefaults()
	if err != nil {
		t.Fatal(err)
	}

	got, err := partial.resolve()
	if err != nil {
		t.Fatal(err)
	}

	want := "/etc/rhsm/ca/redhat-uep.pem"
	if got.RHSM.RepoCACert != want {
		t.Errorf("RHSM.RepoCACert = %q, want %q", got.RHSM.RepoCACert, want)
	}

	main, err := newPartialFromINI([]byte("[rhsm]\nrepo_ca_cert = %(ca_cert_dir)scustom.pem"))
	if err != nil {
		t.Fatal(err)
	}

	err = partial.update(main)
	if err != nil {
		t.Fatal(err)
	}

	got, err = partial.resolve()
	if err != nil {
		t.Fatal(err)
	}

	want = "/etc/rhsm/ca/custom.pem"
	if got.RHSM.RepoCACert != want {
		t.Errorf("RHSM.RepoCACert = %q, want %q", got.RHSM.RepoCACert, want)
	}
}

// TestLoadRejectsInvalidBoolean verifies malformed values are reported.
//
// Given a rhsm.conf with an invalid server.insecure value, when Load resolves
// the configuration, then it returns an error.
func TestLoadRejectsInvalidBoolean(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "rhsm.conf")

	write(t, mainPath, "[server]\ninsecure = maybe\n")

	if _, err := Load(internalfs.Filesystem{}, mainPath); err == nil {
		t.Fatal("Load() error = nil, want invalid boolean error")
	}
}

// write writes contents into a file on disk.
func write(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write to disk: %v", err)
	}
}
