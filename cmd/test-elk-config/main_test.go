package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	elkfs "github.com/m-horky/elk/internal/fs"
)

type discoveryFS struct {
	stats   map[string]elkfs.FileInfo
	statErr map[string]error
	entries []os.DirEntry
}

func (f discoveryFS) Read(string) ([]byte, error) {
	return nil, errors.New("unexpected Read call")
}

func (f discoveryFS) ReadDir(path string) ([]os.DirEntry, error) {
	if path != dropInPath {
		return nil, errors.New("unexpected ReadDir path")
	}
	return f.entries, nil
}

func (f discoveryFS) Stat(path string) (elkfs.FileInfo, error) {
	if err := f.statErr[path]; err != nil {
		return elkfs.FileInfo{}, err
	}
	info, ok := f.stats[path]
	if !ok {
		return elkfs.FileInfo{}, &os.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}
	return info, nil
}

func (f discoveryFS) Open(string) (elkfs.File, error) {
	return nil, errors.New("unexpected Open call")
}

type directoryEntryInfo struct {
	name string
	mode os.FileMode
}

func (i directoryEntryInfo) Name() string               { return i.name }
func (i directoryEntryInfo) IsDir() bool                { return i.mode.IsDir() }
func (i directoryEntryInfo) Type() os.FileMode          { return i.mode }
func (i directoryEntryInfo) Info() (os.FileInfo, error) { return fileInfo(i), nil }

type fileInfo directoryEntryInfo

func (i fileInfo) Name() string       { return i.name }
func (i fileInfo) Size() int64        { return 0 }
func (i fileInfo) Mode() os.FileMode  { return i.mode }
func (i fileInfo) ModTime() time.Time { return time.Time{} }
func (i fileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fileInfo) Sys() any           { return nil }

// TestDiscoverOverridePaths verifies filtering and deterministic ordering of
// configuration files.
//
// Given a main file and mixed drop-in entries, when discovery runs, then it
// returns the main file followed by regular .conf files in lexical order.
func TestDiscoverOverridePaths(t *testing.T) {
	t.Parallel()

	filesystem := discoveryFS{
		stats: map[string]elkfs.FileInfo{
			mainConfigPath: {IsRegular: true},
			dropInPath:     {IsDir: true},
			filepath.Join(dropInPath, "10-site.conf"):   {IsRegular: true},
			filepath.Join(dropInPath, "20-site.conf"):   {IsRegular: true},
			filepath.Join(dropInPath, "directory.conf"): {IsDir: true},
		},
		entries: []os.DirEntry{
			directoryEntryInfo{name: "20-site.conf"},
			directoryEntryInfo{name: "README"},
			directoryEntryInfo{name: ".hidden.conf"},
			directoryEntryInfo{name: "directory.conf", mode: os.ModeDir},
			directoryEntryInfo{name: "10-site.conf"},
		},
	}

	got, err := discoverOverridePaths(filesystem)
	if err != nil {
		t.Fatalf("discoverOverridePaths() error = %v", err)
	}
	want := []string{
		mainConfigPath,
		filepath.Join(dropInPath, "10-site.conf"),
		filepath.Join(dropInPath, "20-site.conf"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverOverridePaths() = %v, want %v", got, want)
	}
}

// TestDiscoverOverridePathsAllowsMissingFiles verifies that absent
// configuration paths are allowed.
//
// Given no main file or drop-in directory, when discovery runs, then it
// returns no paths without error.
func TestDiscoverOverridePathsAllowsMissingFiles(t *testing.T) {
	t.Parallel()

	got, err := discoverOverridePaths(discoveryFS{})
	if err != nil {
		t.Fatalf("discoverOverridePaths() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("discoverOverridePaths() = %v, want no paths", got)
	}
}

// TestDiscoverOverridePathsPropagatesStatError verifies filesystem errors are
// not hidden during discovery.
//
// Given a Stat error for the main file, when discovery runs, then it returns
// an error wrapping the original failure.
func TestDiscoverOverridePathsPropagatesStatError(t *testing.T) {
	t.Parallel()

	statErr := errors.New("permission denied")
	filesystem := discoveryFS{statErr: map[string]error{mainConfigPath: statErr}}

	_, err := discoverOverridePaths(filesystem)
	if !errors.Is(err, statErr) {
		t.Fatalf("discoverOverridePaths() error = %v, want %v", err, statErr)
	}
}
