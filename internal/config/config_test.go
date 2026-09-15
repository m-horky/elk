package config

import (
	"errors"
	iofs "io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	elkfs "github.com/m-horky/elk/internal/fs"
)

const (
	mainConfigPath = "/etc/elk/elk.conf"
	dropInPath     = "/etc/elk/elk.conf.d"
)

// TestGetReturnsEmbeddedDefaults verifies that Get decodes the embedded TOML
// defaults.
//
// Given the compiled default configuration, when Get is called, then the
// returned configuration contains the documented default values.
func TestGetReturnsEmbeddedDefaults(t *testing.T) {
	t.Parallel()

	got, err := Get(Source{Filesystem: elkfs.Filesystem{}})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if !got.Compatibility.InterpretLegacy {
		t.Error("Get().Compatibility.InterpretLegacy = false, want true")
	}

	if got.HTTP.Timeout.Connect != 30*time.Second ||
		got.HTTP.Timeout.Request != 30*time.Second ||
		got.HTTP.Timeout.Idle != 120*time.Second {
		t.Errorf("Get().HTTP.Timeout = %+v, want connect 30s, request 30s, and idle 120s", got.HTTP.Timeout)
	}

	if got.API.Subscriptions.URI != "https://subscription.rhsm.redhat.com/subscription" {
		t.Errorf("Get().API.Subscriptions.URI = %q, want embedded default", got.API.Subscriptions.URI)
	}
}

// TestConfigUpdatePreservesAbsentValues verifies presence-aware partial updates.
//
// Given a populated configuration and a partial override containing only one
// field, when Config.Update applies it, then absent fields retain their values.
func TestConfigUpdatePreservesAbsentValues(t *testing.T) {
	t.Parallel()

	base := Config{
		HTTP: HTTP{
			Timeout: Timeout{Connect: 30 * time.Second, Idle: 120 * time.Second},
			Proxy:   Proxy{URI: "https://proxy.example", User: "user"},
		},
	}
	override := Partial{
		HTTP: &PartialHTTP{
			Timeout: &PartialHTTPTimeout{Connect: new(int)},
		},
	}

	got := base.Update(override)
	if got.HTTP.Timeout.Connect != 0 {
		t.Errorf("Connect = %d, want 0", got.HTTP.Timeout.Connect)
	}

	if got.HTTP.Timeout.Idle != 120*time.Second {
		t.Errorf("Idle = %d, want unchanged value 120s", got.HTTP.Timeout.Idle)
	}

	if got.HTTP.Proxy.URI != "https://proxy.example" || got.HTTP.Proxy.User != "user" {
		t.Errorf("Proxy = %+v, want unchanged proxy", got.HTTP.Proxy)
	}
}

// TestConfigUpdateAppliesExplicitZeroValues verifies that explicit zero, false,
// and empty-string values are treated as overrides.
//
// Given non-zero, true, and non-empty base values, when an override explicitly
// supplies zero, false, and empty values, then all three values are replaced.
func TestConfigUpdateAppliesExplicitZeroValues(t *testing.T) {
	t.Parallel()

	base := Config{
		Compatibility: Compatibility{InterpretLegacy: true},
		HTTP: HTTP{
			Timeout: Timeout{Connect: 30 * time.Second},
			Proxy:   Proxy{URI: "https://proxy.example"},
		},
	}
	override := Partial{
		Compatibility: &PartialCompatibility{InterpretLegacyConfigurations: new(bool)},
		HTTP: &PartialHTTP{
			Timeout: &PartialHTTPTimeout{Connect: new(int)},
			Proxy:   &PartialHTTPProxy{URI: new(string)},
		},
	}

	got := base.Update(override)
	if got.Compatibility.InterpretLegacy {
		t.Error("InterpretLegacy = true, want false")
	}

	if got.HTTP.Timeout.Connect != 0 {
		t.Errorf("Connect = %d, want 0", got.HTTP.Timeout.Connect)
	}

	if got.HTTP.Proxy.URI != "" {
		t.Errorf("Proxy.URI = %q, want empty string", got.HTTP.Proxy.URI)
	}
}

// TestConfigUpdateAppliesNestedEndpoints verifies updates across API sections.
//
// Given nested subscription, RPM, ingress, and inventory overrides, when the
// configuration is updated, then each endpoint receives its supplied values.
func TestConfigUpdateAppliesNestedEndpoints(t *testing.T) {
	t.Parallel()

	subscriptionsURI := "subscriptions"
	rpmURI := "rpm"
	ingressURI := "ingress"
	inventoryURI := "inventory"
	override := Partial{
		API: &PartialAPI{
			Subscriptions: &PartialAPIHost{URI: &subscriptionsURI},
			Content:       &PartialAPIContent{RPM: &PartialAPIHost{URI: &rpmURI}},
			Insights: &PartialAPIInsights{
				Ingress:   &PartialAPIHost{URI: &ingressURI},
				Inventory: &PartialAPIHost{URI: &inventoryURI},
			},
		},
	}

	got := (Config{}).Update(override)
	if got.API.Subscriptions.URI != "subscriptions" {
		t.Errorf("Subscriptions.URI = %q", got.API.Subscriptions.URI)
	}

	if got.API.Content.RPM.URI != "rpm" {
		t.Errorf("Content.RPM.URI = %q", got.API.Content.RPM.URI)
	}

	if got.API.Insights.Ingress.URI != "ingress" {
		t.Errorf("Insights.Ingress.URI = %q", got.API.Insights.Ingress.URI)
	}

	if got.API.Insights.Inventory.URI != "inventory" {
		t.Errorf("Insights.Inventory.URI = %q", got.API.Insights.Inventory.URI)
	}
}

type configTestFS struct {
	stats   map[string]elkfs.FileInfo
	statErr map[string]error
	entries []os.DirEntry
	files   map[string][]byte
}

func (f configTestFS) Read(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, &os.PathError{Op: "read", Path: path, Err: iofs.ErrNotExist}
	}

	return data, nil
}

func (f configTestFS) ReadDir(path string) ([]os.DirEntry, error) {
	if path != dropInPath {
		return nil, errors.New("unexpected ReadDir path") //nolint:err113
	}

	return f.entries, nil
}

func (f configTestFS) Stat(path string) (elkfs.FileInfo, error) {
	if err := f.statErr[path]; err != nil {
		return elkfs.FileInfo{}, err
	}

	info, ok := f.stats[path]
	if !ok {
		return elkfs.FileInfo{}, &os.PathError{Op: "stat", Path: path, Err: iofs.ErrNotExist}
	}

	return info, nil
}

func (f configTestFS) Open(string) (elkfs.File, error) { //nolint:ireturn // required by elkfs.FS
	return nil, errors.New("unexpected Open call") //nolint:err113
}

type configTestDirEntry struct {
	name string
	mode os.FileMode
}

func (e configTestDirEntry) Name() string               { return e.name }
func (e configTestDirEntry) IsDir() bool                { return e.mode.IsDir() }
func (e configTestDirEntry) Type() os.FileMode          { return e.mode }
func (e configTestDirEntry) Info() (os.FileInfo, error) { return configTestFileInfo(e), nil }

type configTestFileInfo configTestDirEntry

func (i configTestFileInfo) Name() string       { return i.name }
func (i configTestFileInfo) Size() int64        { return 0 }
func (i configTestFileInfo) Mode() os.FileMode  { return i.mode }
func (i configTestFileInfo) ModTime() time.Time { return time.Time{} }
func (i configTestFileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i configTestFileInfo) Sys() any           { return nil }

// TestDiscoverOverridePaths verifies filtering and deterministic ordering of
// configuration files.
//
// Given a main file and mixed drop-in entries, when discovery runs, then it
// returns the main file followed by regular .conf files in lexical order.
func TestDiscoverOverridePaths(t *testing.T) {
	t.Parallel()

	filesystem := configTestFS{
		stats: map[string]elkfs.FileInfo{
			mainConfigPath: {IsRegular: true},
			dropInPath:     {IsDir: true},
			filepath.Join(dropInPath, "10-site.conf"):   {IsRegular: true},
			filepath.Join(dropInPath, "20-site.conf"):   {IsRegular: true},
			filepath.Join(dropInPath, "directory.conf"): {IsDir: true},
		},
		entries: []os.DirEntry{
			configTestDirEntry{name: "20-site.conf"},
			configTestDirEntry{name: "README"},
			configTestDirEntry{name: ".hidden.conf"},
			configTestDirEntry{name: "directory.conf", mode: os.ModeDir},
			configTestDirEntry{name: "10-site.conf"},
		},
	}

	got, err := discoverOverridePaths(filesystem, mainConfigPath, dropInPath)
	if err != nil {
		t.Fatalf("discoverOverridePaths() error = %v", err)
	}

	want := []string{mainConfigPath, filepath.Join(dropInPath, "10-site.conf"), filepath.Join(dropInPath, "20-site.conf")}
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

	got, err := discoverOverridePaths(configTestFS{}, mainConfigPath, dropInPath)
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

	statErr := errors.New("permission denied") //nolint:err113

	filesystem := new(configTestFS)
	filesystem.statErr = map[string]error{mainConfigPath: statErr}

	_, err := discoverOverridePaths(*filesystem, mainConfigPath, dropInPath)
	if !errors.Is(err, statErr) {
		t.Fatalf("discoverOverridePaths() error = %v, want %v", err, statErr)
	}
}

// TestLoadConfigAppliesOverrides verifies override files are applied on top
// of the main file.
//
// Given a main file, when two override files are defined with conflicting content,
// then the latter file overwrites the former.
func TestLoadConfigAppliesOverrides(t *testing.T) {
	t.Parallel()

	filesystem := configTestFS{
		stats: map[string]elkfs.FileInfo{
			dropInPath: {IsDir: true},
			filepath.Join(dropInPath, "10-site.conf"): {IsRegular: true},
			filepath.Join(dropInPath, "20-site.conf"): {IsRegular: true},
		},
		entries: []os.DirEntry{
			configTestDirEntry{name: "20-site.conf"},
			configTestDirEntry{name: "10-site.conf"},
		},
		files: map[string][]byte{
			filepath.Join(dropInPath, "10-site.conf"): []byte("[http.timeout]\nconnect = 10\n"),
			filepath.Join(dropInPath, "20-site.conf"): []byte("[http.timeout]\nconnect = 20\nrequest = 25\n"),
		},
	}

	got, err := loadConfig(Source{
		Filesystem: filesystem,
		MainPath:   mainConfigPath,
		DropInsDir: dropInPath,
	})
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if got.HTTP.Timeout.Connect != 20*time.Second {
		t.Errorf("HTTP.Timeout.Connect = %d, want 20s", got.HTTP.Timeout.Connect)
	}

	if got.HTTP.Timeout.Request != 25*time.Second {
		t.Errorf("HTTP.Timeout.Request = %d, want 25s", got.HTTP.Timeout.Request)
	}

	if got.HTTP.Timeout.Idle != 120*time.Second {
		t.Errorf("HTTP.Timeout.Idle = %d, want default 120s", got.HTTP.Timeout.Idle)
	}
}
