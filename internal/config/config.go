package config

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/m-horky/elk/data/etc"
	elkfs "github.com/m-horky/elk/internal/fs"
)

var (
	configOnce   sync.Once //nolint:gochecknoglobals
	cachedConfig Config    //nolint:gochecknoglobals
	errConfig    error
)

// Source describes where configuration overrides are loaded from.
type Source struct {
	Filesystem elkfs.FS
	// MainPath is a path to a file where main configuration file exists.
	MainPath string
	// DropInsDir is a path to a directory where drop-in configuration files
	// may be placed.
	DropInsDir string
	// LegacyPath is a path to rhsm.conf, which is used as an additional source
	// of configuration options, unless disabled.
	LegacyPath string
}

// Get returns a copy of the configuration assembled from the embedded
// defaults and the supplied configuration source. Configuration is loaded at
// most once and cached for the lifetime of the process. If loading fails, the
// same error is returned on subsequent calls.
func Get(source Source) (Config, error) {
	configOnce.Do(func() {
		cachedConfig, errConfig = loadConfig(source)
	})

	if errConfig != nil {
		return Config{}, errConfig
	}

	return cachedConfig, nil
}

// loadConfig assembles configuration from embedded defaults and the supplied
// legacy, main, and drop-in configuration sources.
func loadConfig(source Source) (Config, error) {
	// Find the main configuration file and drop-ins in their application order.
	paths, err := discoverOverridePaths(source.Filesystem, source.MainPath, source.DropInsDir)
	if err != nil {
		return Config{}, err
	}
	slog.Debug("configuration override paths discovered", "count", len(paths), "paths", paths)

	// Read and decode each override configuration file.
	partials := make([]Partial, 0, len(paths))
	for _, path := range paths {
		slog.Debug("decoding configuration override", "path", path)
		data, err := source.Filesystem.Read(path)
		if err != nil {
			return Config{}, fmt.Errorf("read %s: %w", path, err)
		}

		var override Partial
		if _, err := toml.Decode(string(data), &override); err != nil {
			return Config{}, fmt.Errorf("decode %s: %w", path, err)
		}

		partials = append(partials, override)
	}

	// Start from the embedded configuration, which provides values for all fields.
	defaults, err := loadDefaultConfig()
	if err != nil {
		return Config{}, err
	}
	slog.Debug("embedded default configuration loaded")

	// Resolve the `interpret-legacy-configurations` option.
	effective := defaults
	for _, partial := range partials {
		effective = effective.Update(partial)
	}

	// Load the legacy source only when the Elk configuration enables it.
	legacy := Partial{}
	if effective.Compatibility.InterpretLegacy && source.LegacyPath != "" {
		slog.Debug("loading legacy configuration", "path", source.LegacyPath)
		legacy, err = LoadRHSM(source.Filesystem, source.LegacyPath)
		if err != nil {
			return Config{}, err
		}
	}

	cfg := defaults.Update(legacy)

	// Apply Elk overrides on top the built-in and legacy options.
	if len(partials) > 0 {
		slog.Debug("applying configuration overrides", "count", len(partials))
	}
	for _, partial := range partials {
		cfg = cfg.Update(partial)
	}

	return cfg, nil
}

// discoverOverridePaths returns a list of configuration files
// (main one, then the drop-ins) in the order in which they should be applied.
func discoverOverridePaths(filesystem elkfs.FS, mainPath, dropInPath string) ([]string, error) {
	paths := []string{}

	mainStat, err := filesystem.Stat(mainPath)
	if err == nil {
		if mainStat.IsRegular {
			paths = append(paths, mainPath)
		}
	} else if !errors.Is(err, iofs.ErrNotExist) {
		return nil, fmt.Errorf("stat %s: %w", mainPath, err)
	}

	dropInStat, err := filesystem.Stat(dropInPath)
	if errors.Is(err, iofs.ErrNotExist) {
		return paths, nil
	}

	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", dropInPath, err)
	}

	if !dropInStat.IsDir {
		return nil, fmt.Errorf("%s: %w", dropInPath, iofs.ErrInvalid)
	}

	entries, err := filesystem.ReadDir(dropInPath)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dropInPath, err)
	}

	var dropIns []string

	for _, entry := range entries {
		if entry.Name() == "" || entry.Name()[0] == '.' || filepath.Ext(entry.Name()) != ".conf" {
			continue
		}

		path := filepath.Join(dropInPath, entry.Name())

		stat, err := filesystem.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}

		if stat.IsRegular {
			dropIns = append(dropIns, path)
		}
	}

	sort.Strings(dropIns)

	return append(paths, dropIns...), nil
}

func seconds(value int) time.Duration {
	if value <= 0 {
		return 0
	}

	return time.Duration(value) * time.Second
}

func loadDefaultConfig() (Config, error) {
	var p Partial
	if _, err := toml.Decode(etc.ElkDefaultToml, &p); err != nil {
		return Config{}, fmt.Errorf("decode embedded default configuration: %w", err)
	}

	cfg := (Config{}).Update(p)

	return cfg, nil
}

// Update returns a copy of cfg with values present in p applied. Pointer
// fields are used by Partial so that false, zero, and empty string values
// remain distinguishable from fields that were not supplied.
func (cfg Config) Update(p Partial) Config {
	if p.Compatibility != nil && p.Compatibility.InterpretLegacyConfigurations != nil {
		cfg.Compatibility.InterpretLegacy = *p.Compatibility.InterpretLegacyConfigurations
	}

	if p.HTTP != nil { //nolint:nestif
		if p.HTTP.Timeout != nil {
			if p.HTTP.Timeout.Connect != nil {
				cfg.HTTP.Timeout.Connect = seconds(*p.HTTP.Timeout.Connect)
			}

			if p.HTTP.Timeout.Request != nil {
				cfg.HTTP.Timeout.Request = seconds(*p.HTTP.Timeout.Request)
			}

			if p.HTTP.Timeout.Idle != nil {
				cfg.HTTP.Timeout.Idle = seconds(*p.HTTP.Timeout.Idle)
			}
		}

		if p.HTTP.Proxy != nil {
			if p.HTTP.Proxy.URI != nil {
				cfg.HTTP.Proxy.URI = *p.HTTP.Proxy.URI
			}

			if p.HTTP.Proxy.User != nil {
				cfg.HTTP.Proxy.User = *p.HTTP.Proxy.User
			}

			if p.HTTP.Proxy.Password != nil {
				cfg.HTTP.Proxy.Password = *p.HTTP.Proxy.Password
			}

			if p.HTTP.Proxy.NoProxy != nil {
				cfg.HTTP.Proxy.NoProxy = *p.HTTP.Proxy.NoProxy
			}
		}
	}

	if p.API == nil {
		return cfg
	}

	if p.API.Subscriptions != nil {
		cfg.API.Subscriptions = cfg.API.Subscriptions.Update(*p.API.Subscriptions)
	}

	if p.API.Content != nil && p.API.Content.RPM != nil {
		cfg.API.Content.RPM = cfg.API.Content.RPM.Update(*p.API.Content.RPM)
	}

	if p.API.Insights != nil {
		if p.API.Insights.Ingress != nil {
			cfg.API.Insights.Ingress = cfg.API.Insights.Ingress.Update(*p.API.Insights.Ingress)
		}

		if p.API.Insights.Inventory != nil {
			cfg.API.Insights.Inventory = cfg.API.Insights.Inventory.Update(*p.API.Insights.Inventory)
		}
	}

	return cfg
}

// Update returns a copy of e with values present in p applied.
func (e Endpoint) Update(p PartialAPIHost) Endpoint {
	if p.URI != nil {
		e.URI = *p.URI
	}

	if p.TLSVerify != nil {
		e.TLSVerify = *p.TLSVerify
	}

	if p.CAPath != nil {
		e.CAPath = *p.CAPath
	}

	return e
}
