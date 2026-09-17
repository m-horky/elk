// Package certstore provides access to the certificate material used by Elk.
package certstore

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"

	internalcertstore "github.com/m-horky/elk/internal/certstore"
)

const (
	DefaultIdentityDir    = "/var/lib/elk/identity"
	DefaultEntitlementDir = "/var/lib/elk/entitlement"
	DefaultProductDir     = "/var/lib/elk/product"

	IdentityDirEnv    = "ELK_X509_IDENTITY_DIR"
	EntitlementDirEnv = "ELK_X509_ENTITLEMENT_DIR"
	ProductDirEnv     = "ELK_X509_PRODUCT_DIR"
)

// KeyPair contains a PEM encoded certificate and its PEM encoded private key.
type KeyPair struct {
	CertificatePEM []byte
	PrivateKeyPEM  []byte
}

// Certificate contains a PEM encoded X.509 certificate.
type Certificate struct {
	CertificatePEM []byte
}

// IdentityStore stores the system consumer identity certificate and key.
type IdentityStore struct{ store *internalcertstore.Store }

// EntitlementStore stores an entitlement certificate and key.
type EntitlementStore struct{ store *internalcertstore.Store }

// ProductStore stores a product certificate without a private key.
type ProductStore struct{ store *internalcertstore.Store }

// NewIdentityStore opens the identity store, using ELK_X509_IDENTITY_DIR or the default
// identity directory.
func NewIdentityStore() (*IdentityStore, error) {
	store, err := newStore(IdentityDirEnv, DefaultIdentityDir)
	if err != nil {
		return nil, fmt.Errorf("open identity certificate store: %w", err)
	}

	return &IdentityStore{store: store}, nil
}

// NewEntitlementStore opens the entitlement store, using
// ELK_X509_ENTITLEMENT_DIR or the default entitlement directory.
func NewEntitlementStore() (*EntitlementStore, error) {
	store, err := newStore(EntitlementDirEnv, DefaultEntitlementDir)
	if err != nil {
		return nil, fmt.Errorf("open entitlement certificate store: %w", err)
	}

	return &EntitlementStore{store: store}, nil
}

// NewProductStore opens the product store, using ELK_X509_PRODUCT_DIR or the
// default product directory.
func NewProductStore() (*ProductStore, error) {
	store, err := newStore(ProductDirEnv, DefaultProductDir)
	if err != nil {
		return nil, fmt.Errorf("open product certificate store: %w", err)
	}

	return &ProductStore{store: store}, nil
}

func (s *IdentityStore) Save(pair KeyPair) error {
	if err := s.store.Save(pair.CertificatePEM, pair.PrivateKeyPEM); err != nil {
		return fmt.Errorf("save identity certificate: %w", err)
	}

	return nil
}

func (s *IdentityStore) Load() (KeyPair, error) {
	certificate, privateKey, err := s.store.Load()
	if err != nil {
		return KeyPair{}, fmt.Errorf("load identity certificate: %w", err)
	}

	return KeyPair{CertificatePEM: certificate, PrivateKeyPEM: privateKey}, nil
}

func (s *IdentityStore) LoadTLS() (tls.Certificate, error) {
	pair, err := s.Load()
	if err != nil {
		return tls.Certificate{}, err
	}

	certificate, err := tls.X509KeyPair(pair.CertificatePEM, pair.PrivateKeyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("load TLS certificate: %w", err)
	}

	return certificate, nil
}

func (s *IdentityStore) Delete() error {
	if err := s.store.Delete(); err != nil {
		return fmt.Errorf("delete identity certificate: %w", err)
	}

	return nil
}

func (s *EntitlementStore) Save(pair KeyPair) error {
	if err := s.store.Save(pair.CertificatePEM, pair.PrivateKeyPEM); err != nil {
		return fmt.Errorf("save entitlement certificate: %w", err)
	}

	return nil
}

func (s *EntitlementStore) Load() (KeyPair, error) {
	certificate, privateKey, err := s.store.Load()
	if err != nil {
		return KeyPair{}, fmt.Errorf("load entitlement certificate: %w", err)
	}

	return KeyPair{CertificatePEM: certificate, PrivateKeyPEM: privateKey}, nil
}

func (s *EntitlementStore) LoadTLS() (tls.Certificate, error) {
	pair, err := s.Load()
	if err != nil {
		return tls.Certificate{}, err
	}

	certificate, err := tls.X509KeyPair(pair.CertificatePEM, pair.PrivateKeyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("load TLS certificate: %w", err)
	}

	return certificate, nil
}

func (s *EntitlementStore) Delete() error {
	if err := s.store.Delete(); err != nil {
		return fmt.Errorf("delete entitlement certificate: %w", err)
	}

	return nil
}

func (s *ProductStore) Save(certificate Certificate) error {
	if err := s.store.SaveCertificate(certificate.CertificatePEM); err != nil {
		return fmt.Errorf("save product certificate: %w", err)
	}

	return nil
}

func (s *ProductStore) Load() (Certificate, error) {
	certificate, err := s.store.LoadCertificate()
	if err != nil {
		return Certificate{}, fmt.Errorf("load product certificate: %w", err)
	}

	return Certificate{CertificatePEM: certificate}, nil
}

func (s *ProductStore) Delete() error {
	if err := s.store.Delete(); err != nil {
		return fmt.Errorf("delete product certificate: %w", err)
	}

	return nil
}

func newStore(environment, fallback string) (*internalcertstore.Store, error) {
	directory := os.Getenv(environment)
	if directory == "" {
		directory = fallback
		slog.Debug("using default certificate store directory", "directory", directory)
	} else {
		slog.Debug("using configured certificate store directory", "directory", directory)
	}

	store, err := internalcertstore.New(directory)
	if err != nil {
		return nil, fmt.Errorf("create certificate store: %w", err)
	}

	return store, nil
}
