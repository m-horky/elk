package constants

import "os"

type pathID string

const (
	ConfigPath        pathID = "ELK_CONFIG"
	ConfigDropInsDir  pathID = "ELK_DROPINS"
	RHSMConfigPath    pathID = "ELK_RHSM_CONFIG"
	LockPath          pathID = "ELK_LOCK"
	IdentityDir       pathID = "ELK_CONSUMER_DIR"
	IdentityCertPath  pathID = "ELK_CONSUMER_CERT"
	IdentityKeyPath   pathID = "ELK_CONSUMER_KEY"
	EntitlementDir    pathID = "ELK_ENTITLEMENT_DIR"
	ProductDir        pathID = "ELK_PRODUCT_DIR"
	DefaultProductDir pathID = "ELK_DEFAULT_PRODUCT_DIR"
	OSReleasePath     pathID = "ELK_OS_RELEASE"
	LSCPUBin          pathID = "ELK_BIN_LSCPU"
	VirtWhatBin       pathID = "ELK_BIN_VIRT_WHAT"
	CloudCacheDir     pathID = "ELK_CLOUD_CACHE_DIR"
)

var paths = map[pathID]string{ //nolint:gochecknoglobals
	ConfigPath:        "/etc/rhc/elk.conf",
	ConfigDropInsDir:  "/etc/rhc/elk.conf.d/",
	RHSMConfigPath:    "/etc/rhsm/rhsm.conf",
	LockPath:          "/run/elk/elk.lock",
	IdentityDir:       "/etc/pki/consumer",
	IdentityCertPath:  "/etc/pki/consumer/cert.pem",
	IdentityKeyPath:   "/etc/pki/consumer/key.pem",
	EntitlementDir:    "/etc/pki/entitlement",
	ProductDir:        "/etc/pki/product",
	DefaultProductDir: "/etc/pki/product-default",
	OSReleasePath:     "/usr/lib/os-release",
	LSCPUBin:          "/usr/bin/lscpu",
	VirtWhatBin:       "/usr/sbin/virt-what",
	CloudCacheDir:     "/var/cache/elk/cloud",
}

// Path returns the system path for name, preferring a non-empty environment override.
func Path(name pathID) string {
	if path := os.Getenv(string(name)); path != "" {
		return path
	}

	return paths[name]
}
