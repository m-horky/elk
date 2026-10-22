package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/m-horky/elk/internal/config/rhsm"
	internalfs "github.com/m-horky/elk/internal/fs"
)

const maxConfigSize = 1024 * 1024 // 1 MiB

// Load merges defaults, optional RHSM settings, the main file, and drop-ins.
// The RHSM fallback is applied only when the resolved configuration enables
// legacy configuration interpretation.
func Load(filesystem internalfs.FS, path, dropInDir string, rhsmFallback *rhsm.Config) (Config, error) {
	partial, err := newPartialFromDefaults()
	if err != nil {
		return Config{}, fmt.Errorf("cannot read default configuration: %w", err)
	}

	mainPartial, err := newPartialFromFile(filesystem, path)
	if err != nil {
		return Config{}, fmt.Errorf("cannot parse main configuration: %w", err)
	}

	dropIns, err := discoverDropIns(filesystem, dropInDir)
	if err != nil {
		return Config{}, err
	}

	dropInPartials := make([]*dtoConfig, 0, len(dropIns))
	for _, dropIn := range dropIns {
		dropInPartial, err := newPartialFromFile(filesystem, dropIn)
		if err != nil {
			return Config{}, fmt.Errorf("drop-in configuration %s: %w", dropIn, err)
		}

		dropInPartials = append(dropInPartials, dropInPartial)
	}

	// merge RHSM settings before the native layers, when enabled
	if interpretLegacy(partial, mainPartial, dropInPartials) && rhsmFallback != nil {
		rhsmPartial, err := newPartialFromRHSM(*rhsmFallback)
		if err != nil {
			return Config{}, fmt.Errorf("cannot convert RHSM configuration: %w", err)
		}

		if err := partial.update(&rhsmPartial); err != nil {
			return Config{}, fmt.Errorf("cannot merge RHSM configuration: %w", err)
		}
	}

	// merge main file
	if err := partial.update(mainPartial); err != nil {
		return Config{}, fmt.Errorf("cannot merge main configuration: %w", err)
	}

	// merge drop-in files
	for _, dropInPartial := range dropInPartials {
		if err := partial.update(dropInPartial); err != nil {
			return Config{}, fmt.Errorf("cannot merge drop-in configuration: %w", err)
		}
	}

	// resolve and validate
	resolved, err := partial.resolve()
	if err != nil {
		return Config{}, fmt.Errorf("resolve native configuration: %w", err)
	}

	return resolved, nil
}

// interpretLegacy resolves the legacy-interpretation flag from the defaults and native layers.
func interpretLegacy(partial *dtoConfig, main *dtoConfig, dropIns []*dtoConfig) bool {
	probe := *partial

	if err := probe.update(main); err != nil {
		return false
	}

	for _, dropIn := range dropIns {
		if err := probe.update(dropIn); err != nil {
			return false
		}
	}

	return probe.Compatibility.InterpretLegacy != nil && *probe.Compatibility.InterpretLegacy
}

// discoverDropIns finds paths of drop-ins.
func discoverDropIns(filesystem internalfs.FS, directory string) ([]string, error) {
	entries, err := filesystem.ReadDir(directory)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("cannot discover drop-in files: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	var result []string

	for _, entry := range entries {
		entryPath := filepath.Join(directory, entry.Name())
		if !strings.HasSuffix(strings.ToLower(entryPath), ".conf") {
			continue
		}

		result = append(result, entryPath)
	}

	return result, nil
}

// newPartialFromFile reads a file and decodes its content into a partial native configuration.
func newPartialFromFile(filesystem internalfs.FS, path string) (*dtoConfig, error) {
	raw, err := filesystem.Read(path, maxConfigSize)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &dtoConfig{}, nil
		}

		return nil, fmt.Errorf("read configuration file %s: %w", path, err)
	}

	parsed, err := newPartialFromTOML(raw)
	if err != nil {
		return nil, fmt.Errorf("parse configuration file %s: %w", path, err)
	}

	return parsed, nil
}

// newPartialFromTOML decodes a TOML document into a partial native configuration.
func newPartialFromTOML(data []byte) (*dtoConfig, error) {
	var result *dtoConfig
	if _, err := toml.Decode(string(data), &result); err != nil {
		return nil, errors.New("invalid TOML syntax or value type")
	}

	return result, nil
}
