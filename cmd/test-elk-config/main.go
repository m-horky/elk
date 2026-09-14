package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/m-horky/elk/internal/config"
)

const (
	mainConfigPath = "/etc/elk/elk.conf"
	dropInPath     = "/etc/elk/elk.conf.d"
)

func main() {
	cfg, err := config.Get()
	if err != nil {
		fatal(err)
	}

	paths, err := discoverOverridePaths()
	if err != nil {
		fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
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

func discoverOverridePaths() ([]string, error) {
	paths := []string{}
	if _, err := os.Stat(mainConfigPath); err == nil {
		paths = append(paths, mainConfigPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	entries, err := os.ReadDir(dropInPath)
	if errors.Is(err, fs.ErrNotExist) {
		return paths, nil
	}
	if err != nil {
		return nil, err
	}

	var dropIns []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name()[0] == '.' || filepath.Ext(entry.Name()) != ".conf" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if info.Mode().IsRegular() {
			dropIns = append(dropIns, filepath.Join(dropInPath, entry.Name()))
		}
	}
	sort.Strings(dropIns)
	return append(paths, dropIns...), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
