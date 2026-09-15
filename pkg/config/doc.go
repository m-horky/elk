// Package config loads the application's resolved configuration.
//
// Get loads the embedded defaults and applies configuration overrides from
// ELK_CONFIG_DIR, or /etc/elk when that variable is not set. The configuration
// file is elk.conf; additional .conf files in elk.conf.d are applied in
// lexical order.
//
// Load the resolved configuration with Get:
//
//	cfg, err := config.Get()
//	if err != nil {
//	  return err
//	}
//	timeout := cfg.HTTP.Timeout.Connect
//	_ = timeout
package config
