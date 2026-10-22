package config

import (
	"fmt"
	"log/slog"

	internalconfig "github.com/m-horky/elk/internal/config"
	"github.com/m-horky/elk/internal/config/rhsm"
	elkfs "github.com/m-horky/elk/internal/fs"
	"github.com/m-horky/elk/pkg/constants"
)

// Config is the resolved application configuration.
type Config = internalconfig.Config

// HTTP contains the resolved outgoing HTTP settings.
type HTTP = internalconfig.HTTP

// Timeout contains the resolved HTTP transport timeouts.
type Timeout = internalconfig.Timeout

// Proxy describes the HTTP proxy, its credentials, and bypass rules.
type Proxy = internalconfig.Proxy

// API contains the resolved remote endpoints.
type API = internalconfig.API

// Content contains the resolved content-delivery endpoints.
type Content = internalconfig.Content

// Insights contains the resolved Insights endpoints.
type Insights = internalconfig.Insights

// Endpoint describes a configured service endpoint.
type Endpoint = internalconfig.Endpoint

// Get loads the embedded defaults and application overrides from the system
// configuration paths, or their environment overrides.
func Get() (Config, error) {
	mainPath := constants.Path(constants.ConfigPath)
	dropInsDir := constants.Path(constants.ConfigDropInsDir)
	rhsmPath := constants.Path(constants.RHSMConfigPath)

	slog.Debug("loading application configuration",
		"path", mainPath, "drop-ins", dropInsDir, "rhsm", rhsmPath)

	fallback, err := rhsm.Load(elkfs.Filesystem{}, rhsmPath)
	if err != nil {
		return Config{}, fmt.Errorf("load RHSM configuration: %w", err)
	}

	cfg, err := internalconfig.Load(elkfs.Filesystem{}, mainPath, dropInsDir, &fallback)
	if err != nil {
		return Config{}, fmt.Errorf("load application configuration: %w", err)
	}

	return cfg, nil
}
