package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/m-horky/elk/pkg/config"
)

func main() {
	cfg, err := config.Get()
	if err != nil {
		fatal(err)
	}

	enc := toml.NewEncoder(os.Stdout)

	enc.Indent = ""
	if err := enc.Encode(cfg); err != nil {
		fatal(fmt.Errorf("write configuration: %w", err))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
