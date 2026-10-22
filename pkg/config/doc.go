// Package config loads the application's resolved configuration.
//
// Get loads the embedded defaults and applies configuration overrides from
// /etc/rhc/elk.conf and the .conf files in /etc/rhc/elk.conf.d in lexical
// order. Settings from /etc/rhsm/rhsm.conf are translated and merged under the
// native defaults when legacy configuration interpretation is enabled. Path
// overrides are provided by the pkg/constants package.
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
