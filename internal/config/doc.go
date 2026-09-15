// Package config decodes the application's default configuration and merges
// TOML overrides from an explicit source.
//
// Source identifies the filesystem, main override file, and drop-in directory
// used by Get. Get starts with the embedded defaults, applies the main file
// when present, and then applies regular .conf files from the drop-in
// directory in lexical order. The result is cached for the lifetime of the
// process.
//
// Load configuration from a filesystem source:
//
//	source := config.Source{
//		Filesystem: fs.Filesystem{},
//		MainPath:   "/etc/elk/elk.conf",
//		DropInsDir: "/etc/elk/elk.conf.d",
//	}
//	cfg, err := config.Get(source)
//	if err != nil {
//		return err
//	}
//
// Config.Update applies a Partial value to a configuration. Partial and its
// nested types use pointer fields so that an omitted TOML value can be
// distinguished from an explicit false, zero, or empty string. For example,
// this updates only the subscriptions endpoint's TLS verification setting:
//
//	tlsVerify := true
//	cfg = cfg.Update(config.Partial{
//		API: &config.PartialAPI{
//			Subscriptions: &config.PartialAPIHost{TLSVerify: &tlsVerify},
//		},
//	})
//
// Developers are suggested to use the public pkg.config.Get() method instead.
package config
