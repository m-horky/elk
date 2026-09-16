// Package config decodes the application's default configuration and
// merges TOML overrides from an explicit source.
//
// Developers are suggested to use the pkg/config package instead.
//
// Source identifies the filesystem, main override file, drop-in directory, and
// optional legacy rhsm.conf path used by Get. The public wrapper supplies
// /etc/rhsm/rhsm.conf; an empty path disables the compatibility source. Get
// starts with the embedded defaults, optionally
// applies translated legacy settings, then applies the main file and regular
// .conf files from the drop-in directory in lexical order. The result is cached
// for the lifetime of the process.
//
// Load configuration from a filesystem source:
//
//	source := config.Source{
//		Filesystem: fs.Filesystem{},
//		MainPath:   "/etc/elk/elk.conf",
//		DropInsDir: "/etc/elk/elk.conf.d",
//		LegacyPath: "/etc/rhsm/rhsm.conf",
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
// A Partial object might be created by parsing a TOML file containing
// some configuration options, or it might be parsed from the legacy rhsm.conf
// file:
//
//	var override Partial
//	_, err := toml.Decode(data, &override)
//
//	rhsm, err := LoadRHSM(fs, "/etc/rhsm/rhsm.conf")
//	fmt.Printf("%#v\n%#v\n", override, rhsm)
package config
