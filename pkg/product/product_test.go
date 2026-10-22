package product

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

// TestGetUsesEnvironmentOverride verifies Get scans the configured product directory.
//
// Given an environment override pointing to a directory with a generated
// product certificate, when Get scans for products, then the certificate's
// product metadata is returned.
func TestGetUsesEnvironmentOverride(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "69.pem")

	if err := os.WriteFile(path, testCertificate(t), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ELK_PRODUCT_DIR", directory)
	t.Setenv("ELK_DEFAULT_PRODUCT_DIR", filepath.Join(directory, "absent"))

	products, err := Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("Get() returned %d products, want 1", len(products))
	}

	got := products[0]
	if got.ID != 69 || got.Name != "RHEL" || len(got.Versions) != 1 || got.Versions[0] != "10.0" {
		t.Errorf("Get() = %+v, want product metadata from the generated certificate", got)
	}
}

// testCertificate creates a self-signed PEM certificate with product extensions.
func testCertificate(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "fake-product"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Minute),
		ExtraExtensions: []pkix.Extension{
			{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 69, 1}, Value: []byte("RHEL")},
			{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 69, 2}, Value: []byte("10.0")},
			{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 69, 3}, Value: []byte("x86_64")},
		},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create test certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
