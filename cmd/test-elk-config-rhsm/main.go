package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
	internalconfig "github.com/m-horky/elk/internal/config"
	elkfs "github.com/m-horky/elk/internal/fs"
)

const defaultLegacyPath = "/etc/rhsm/rhsm.conf"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(); err != nil {
		if !errors.Is(err, errUsage) {
			slog.Error("command failed", "err", err)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	path := defaultLegacyPath

	if len(os.Args) > 2 {
		return fmt.Errorf("%w: usage: test-elk-config-rhsm [rhsm.conf]", errUsage)
	}

	if len(os.Args) == 2 {
		path = os.Args[1]
	}

	slog.Info("loading legacy configuration", "path", path)
	partial, err := internalconfig.LoadRHSM(elkfs.Filesystem{}, path)
	if err != nil {
		return err
	}

	encoder := toml.NewEncoder(os.Stdout)
	encoder.Indent = ""

	if err := encoder.Encode(partial); err != nil {
		return fmt.Errorf("write legacy configuration: %w", err)
	}

	slog.Info("legacy configuration output written")
	return nil
}

var errUsage = errors.New("invalid command usage")
