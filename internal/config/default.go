package config

import _ "embed"

// defaultConfig contains the package's embedded configuration used by newPartialFromDefaults.
//
//go:embed default.conf
var defaultConfig []byte

// newPartialFromDefaults decodes the embedded default configuration into a partial configuration.
func newPartialFromDefaults() (*dtoConfig, error) {
	return newPartialFromTOML(defaultConfig)
}
