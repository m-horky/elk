package main

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/m-horky/elk/internal/config"
	elkfs "github.com/m-horky/elk/internal/fs"
)

const (
	mainConfigPath = "/etc/elk/elk.conf"
	dropInPath     = "/etc/elk/elk.conf.d"
)

func main() {
	filesystem := elkfs.Filesystem{}

	cfg, err := config.Get()
	if err != nil {
		fatal(err)
	}

	paths, err := discoverOverridePaths(filesystem)
	if err != nil {
		fatal(err)
	}
	for _, path := range paths {
		data, err := filesystem.Read(path)
		if err != nil {
			fatal(fmt.Errorf("read %s: %w", path, err))
		}
		var override config.Partial
		if _, err := toml.Decode(string(data), &override); err != nil {
			fatal(fmt.Errorf("decode %s: %w", path, err))
		}
		cfg = cfg.Update(override)
	}

	enc := toml.NewEncoder(os.Stdout)
	enc.Indent = ""
	if err := enc.Encode(cfg); err != nil {
		fatal(fmt.Errorf("write configuration: %w", err))
	}
}

func discoverOverridePaths(filesystem elkfs.FS) ([]string, error) {
	paths := []string{}
	mainStat, err := filesystem.Stat(mainConfigPath)
	if err == nil {
		if mainStat.IsRegular {
			paths = append(paths, mainConfigPath)
		}
	} else if !errors.Is(err, iofs.ErrNotExist) {
		return nil, fmt.Errorf("stat %s: %w", mainConfigPath, err)
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
		entryPath := filepath.Join(dropInPath, entry.Name())
		entryStat, err := filesystem.Stat(entryPath)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", entryPath, err)
		}
		if entryStat.IsRegular {
			dropIns = append(dropIns, entryPath)
		}
	}
	sort.Strings(dropIns)
	return append(paths, dropIns...), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
