package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	internalconfig "github.com/m-horky/elk/internal/config"
	elkfs "github.com/m-horky/elk/internal/fs"
)

const defaultLegacyPath = "/etc/rhsm/rhsm.conf"

func main() {
	path := defaultLegacyPath

	if len(os.Args) > 2 {
		fatal(errors.New("usage: test-elk-config-rhsm [rhsm.conf]"))
	}

	if len(os.Args) == 2 {
		path = os.Args[1]
	}

	partial, err := internalconfig.LoadRHSM(elkfs.Filesystem{}, path)
	if err != nil {
		fatal(err)
	}

	encoder := toml.NewEncoder(os.Stdout)
	encoder.Indent = ""

	if err := encoder.Encode(partial); err != nil {
		fatal(fmt.Errorf("write legacy configuration: %w", err))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
