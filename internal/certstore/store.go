// Package certstore provides the filesystem implementation used by the public
// certificate stores.
package certstore

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var (
	errEmptyDirectory = errors.New("certificate store directory must not be empty")
	errNoCertificate  = errors.New("certificate contains no PEM CERTIFICATE block")
)

const (
	certificateFile = "cert.pem"
	privateKeyFile  = "key.pem"
	directoryMode   = 0o700
	fileMode        = 0o600
)

// Store persists certificate material in one directory.
type Store struct {
	directory string
}

// New creates or opens a certificate store rooted at directory.
func New(directory string) (*Store, error) {
	if directory == "" {
		return nil, errEmptyDirectory
	}

	if err := os.MkdirAll(directory, directoryMode); err != nil {
		return nil, fmt.Errorf("create certificate store directory: %w", err)
	}

	if err := os.Chmod(directory, directoryMode); err != nil {
		return nil, fmt.Errorf("set certificate store directory permissions: %w", err)
	}

	return &Store{directory: directory}, nil
}

// Save stores a PEM certificate and its PEM private key.
func (s *Store) Save(certificate, privateKey []byte) error {
	if _, err := tls.X509KeyPair(certificate, privateKey); err != nil {
		return fmt.Errorf("validate certificate and private key: %w", err)
	}

	if err := writeFile(filepath.Join(s.directory, certificateFile), certificate); err != nil {
		return fmt.Errorf("save certificate: %w", err)
	}

	if err := writeFile(filepath.Join(s.directory, privateKeyFile), privateKey); err != nil {
		return fmt.Errorf("save private key: %w", err)
	}

	return nil
}

// Load returns the stored PEM certificate and private key.
func (s *Store) Load() ([]byte, []byte, error) {
	certificate, err := readFile(filepath.Join(s.directory, certificateFile))
	if err != nil {
		return nil, nil, fmt.Errorf("load certificate: %w", err)
	}

	privateKey, err := readFile(filepath.Join(s.directory, privateKeyFile))
	if err != nil {
		return nil, nil, fmt.Errorf("load private key: %w", err)
	}

	if _, err := tls.X509KeyPair(certificate, privateKey); err != nil {
		return nil, nil, fmt.Errorf("validate certificate and private key: %w", err)
	}

	return certificate, privateKey, nil
}

// SaveCertificate stores a PEM encoded X.509 certificate without a private key.
func (s *Store) SaveCertificate(certificate []byte) error {
	block, _ := pem.Decode(certificate)
	if block == nil || block.Type != "CERTIFICATE" {
		return errNoCertificate
	}

	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}

	if err := writeFile(filepath.Join(s.directory, certificateFile), certificate); err != nil {
		return fmt.Errorf("save certificate: %w", err)
	}

	return nil
}

// LoadCertificate returns the stored PEM encoded X.509 certificate.
func (s *Store) LoadCertificate() ([]byte, error) {
	certificate, err := readFile(filepath.Join(s.directory, certificateFile))
	if err != nil {
		return nil, fmt.Errorf("load certificate: %w", err)
	}

	block, _ := pem.Decode(certificate)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errNoCertificate
	}

	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	return certificate, nil
}

// Delete removes all certificate material in the store.
func (s *Store) Delete() error {
	for _, name := range []string{certificateFile, privateKeyFile} {
		if err := os.Remove(filepath.Join(s.directory, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete %s: %w", name, err)
		}
	}

	return nil
}

func writeFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, fileMode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	if err := os.Chmod(path, fileMode); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}

	return nil
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path) // #nosec G304: paths are confined to the configured certificate store.
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
		}

		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}
