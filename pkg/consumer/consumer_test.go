package consumer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGetUsesEnvironmentOverride verifies Get reads the configured certificate path.
//
// Given an environment override pointing to a generated consumer certificate,
// when Get reads the identity, then the certificate's subject is returned.
func TestGetUsesEnvironmentOverride(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "cert.pem")

	contents := testCertificate(t)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ELK_CONSUMER_CERT", path)

	identity, err := Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if identity.UUID != "fake-uuid" || identity.OrgID != "fake-org" {
		t.Errorf("Get() = %+v, want subject from the generated certificate", identity)
	}
}

// testCertificate creates a self-signed PEM certificate with a known subject.
func testCertificate(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "fake-uuid",
			Organization: []string{"fake-org"},
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Minute),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create test certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
