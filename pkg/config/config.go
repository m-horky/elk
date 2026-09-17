package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	internalconfig "github.com/m-horky/elk/internal/config"
	elkfs "github.com/m-horky/elk/internal/fs"
)

const (
	defaultConfigDir  = "/etc/elk"
	defaultLegacyPath = "/etc/rhsm/rhsm.conf"
	configDirEnv      = "ELK_CONFIG_DIR"
)

// Config is the resolved application configuration.
type Config = internalconfig.Config

// Get loads the embedded defaults and application overrides. ELK_CONFIG_DIR,
// when set, replaces /etc/elk as the configuration directory.
func Get() (Config, error) {
	dir := os.Getenv(configDirEnv)
	if dir == "" {
		dir = defaultConfigDir
		slog.Debug("using default configuration directory", "directory", dir)
	} else {
		slog.Debug("using configured configuration directory", "directory", dir)
	}

	source := internalconfig.Source{
		Filesystem: elkfs.Filesystem{},
		MainPath:   filepath.Join(dir, "elk.conf"),
		DropInsDir: filepath.Join(dir, "elk.conf.d"),
		LegacyPath: defaultLegacyPath,
	}

	cfg, err := internalconfig.Get(source)
	if err != nil {
		return Config{}, fmt.Errorf("load application configuration: %w", err)
	}

	return cfg, nil
}
