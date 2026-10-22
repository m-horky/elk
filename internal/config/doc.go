// Package config loads native application configuration and exposes its
// resolved HTTP and API settings. Load starts with embedded defaults, applies
// an optional RHSM fallback, then merges the native configuration file and
// sorted .conf drop-ins. Values from later sources override earlier values.
// The RHSM fallback is applied only when the resolved configuration enables
// legacy configuration interpretation.
//
// For example, load RHSM settings first and pass them as a fallback while
// loading native settings:
//
//	filesystem := fs.Filesystem{}
//	rhsmConfig, _ := rhsm.Load(filesystem, "/etc/rhsm/rhsm.conf")
//	cfg, _ := config.Load(
//		filesystem,
//		"/etc/rhc/elk.conf",
//		"/etc/rhc/elk.conf.d",
//		&rhsmConfig,
//	)
//	_ = cfg.API.Subscriptions.URI
//	_ = cfg.HTTP.Timeout.Request
package config
