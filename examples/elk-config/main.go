package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/m-horky/elk/pkg/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(); err != nil {
		slog.Error("command failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	slog.Info("loading configuration")
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	enc := toml.NewEncoder(os.Stdout)
	enc.Indent = ""
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}

	slog.Info("configuration output written")
	return nil
}
