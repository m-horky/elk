// Package config provides the application's resolved configuration and the
// presence-aware values used to apply TOML overrides.
//
// Call Get to obtain a copy of the embedded default configuration:
//
//	cfg, err := config.Get()
//	if err != nil {
//		return err
//	}
//
// To apply an override file, decode it into Partial and pass the result to
// Config.Update:
//
//	var override config.Partial
//	if _, err := toml.Decode(data, &override); err != nil {
//		return err
//	}
//	cfg = cfg.Update(override)
//
// External callers shouldn't concern themselves with Partial objects; they
// should be plain consumers of config.Get().
package config
