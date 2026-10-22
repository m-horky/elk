package consumer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	iofs "io/fs"
	"math/big"
	"testing"
	"time"

	internalfs "github.com/m-horky/elk/internal/fs"
)

// TestRead verifies identity extraction from the certificate subject.
//
// Given a PEM certificate with a common name and one organization,
// when reading it, then both identity fields are returned.
func TestRead(t *testing.T) {
	path := "/virtual/consumer.pem"
	contents := newCertificate(t, pkix.Name{
		CommonName:   "fake-uuid",
		Organization: []string{"fake-org"},
	})
	filesystem := virtFS{files: map[string][]byte{path: contents}}

	got, err := Read(filesystem, path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got.UUID != "fake-uuid" || got.OrgID != "fake-org" {
		t.Errorf("Read() = %+v, want UUID and organization from certificate subject", got)
	}
}

// TestReadRejectsMissingOrganization verifies a subject without exactly one organization is rejected.
//
// Given a PEM certificate whose subject lists no organization, when reading it,
// then Read returns an error.
func TestReadRejectsMissingOrganization(t *testing.T) {
	path := "/virtual/consumer.pem"
	contents := newCertificate(t, pkix.Name{CommonName: "fake-uuid"})
	filesystem := virtFS{files: map[string][]byte{path: contents}}

	if _, err := Read(filesystem, path); err == nil {
		t.Fatal("Read() error = nil, want missing organization error")
	}
}

// TestReadRejectsMissingCommonName verifies a subject without a common name is rejected.
//
// Given a PEM certificate whose subject has an organization but no common name,
// when reading it, then Read returns an error.
func TestReadRejectsMissingCommonName(t *testing.T) {
	path := "/virtual/consumer.pem"
	contents := newCertificate(t, pkix.Name{Organization: []string{"fake-org"}})
	filesystem := virtFS{files: map[string][]byte{path: contents}}

	if _, err := Read(filesystem, path); err == nil {
		t.Fatal("Read() error = nil, want missing common name error")
	}
}

// TestReadRejectsMissingCertificate verifies an unreadable certificate path is rejected.
//
// Given a certificate path that does not exist, when reading it, then Read
// returns an error wrapping os.ErrNotExist.
func TestReadRejectsMissingCertificate(t *testing.T) {
	filesystem := virtFS{files: map[string][]byte{}}

	_, err := Read(filesystem, "/virtual/missing.pem")
	if err == nil {
		t.Fatal("Read() error = nil, want not-exist error")
	}
}

// newCertificate creates a self-signed PEM certificate with the supplied subject.
func newCertificate(t *testing.T, subject pkix.Name) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subject,
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Minute),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create test certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

type virtFS struct {
	internalfs.FS

	files map[string][]byte
}

var _ internalfs.FS = virtFS{}

// Read returns configured test contents up to maxBytes or a virtual not-exist error.
func (filesystem virtFS) Read(path string, maxBytes int64) ([]byte, error) {
	data, ok := filesystem.files[path]
	if !ok {
		return nil, &iofs.PathError{Op: "read", Path: path, Err: iofs.ErrNotExist}
	}

	if int64(len(data)) > maxBytes {
		return append([]byte(nil), data[:maxBytes]...), internalfs.ErrTruncated
	}

	return append([]byte(nil), data...), nil
}
