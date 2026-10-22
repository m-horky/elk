// Package certstore provides the filesystem implementation used by the public
// certificate stores.
//
// A Store persists certificate material in one directory using the RHSM file
// layout: certificate in cert.pem and private key in key.pem. Use the
// pkg/certstore package instead of this internal implementation.
package certstore
