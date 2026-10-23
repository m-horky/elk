package distribution

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	internalfs "github.com/m-horky/elk/internal/fs"
)

// fakeFS serves configured contents or a configured error for every read.
type fakeFS struct {
	internalfs.FS

	data     []byte
	err      error
	maxBytes int64
}

// Read returns the configured data or error and records the byte limit.
func (filesystem *fakeFS) Read(_ string, maxBytes int64) ([]byte, error) {
	filesystem.maxBytes = maxBytes

	if filesystem.err != nil {
		return nil, filesystem.err
	}

	return append([]byte(nil), filesystem.data...), nil
}

var _ internalfs.FS = (*fakeFS)(nil)

// TestCollect verifies that Collect reads with the size limit and parses.
//
// Given a filesystem containing an os-release fixture, when Collect reads the
// configured path, then it parses the distribution facts.
func TestCollect(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("testdata", "rhel-os-release"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	filesystem := &fakeFS{data: data}
	want := Distribution{
		ID:              new("rhel"),
		Name:            new("Red Hat Enterprise Linux"),
		Version:         new("9.4"),
		VersionModifier: new("ga"),
	}

	got, err := Collect(filesystem, "/usr/lib/os-release")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Collect() mismatch (-want +got):\n%s", diff)
	}

	if filesystem.maxBytes != maxSize {
		t.Fatalf("Read() maxBytes = %d, want %d", filesystem.maxBytes, maxSize)
	}
}

// TestCollectReadError verifies that filesystem read errors are returned.
//
// Given a filesystem whose read fails, when Collect reads the configured path,
// then it returns the wrapped read error and no facts.
func TestCollectReadError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read failure")
	filesystem := &fakeFS{err: readErr}

	got, err := Collect(filesystem, "/usr/lib/os-release")
	if !errors.Is(err, readErr) {
		t.Fatalf("Collect() error = %v, want %v", err, readErr)
	}

	if diff := cmp.Diff(Distribution{}, got); diff != "" {
		t.Fatalf("Collect() mismatch (-want +got):\n%s", diff)
	}
}
