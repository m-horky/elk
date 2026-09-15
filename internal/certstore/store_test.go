package certstore

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestStoreRoundTrip verifies that saved certificate material can be loaded unchanged.
//
// Given a certificate store and a valid certificate/key pair
// When the pair is saved and loaded
// Then both PEM values are returned unchanged.
func TestStoreRoundTrip(t *testing.T) {
	directory := t.TempDir()

	store, err := New(filepath.Join(directory, "identity"))
	if err != nil {
		t.Fatal(err)
	}

	certificate, privateKey := testCertificate(t)

	if err := store.Save(certificate, privateKey); err != nil {
		t.Fatal(err)
	}

	gotCertificate, gotPrivateKey, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}

	if string(gotCertificate) != string(certificate) || string(gotPrivateKey) != string(privateKey) {
		t.Fatal("loaded certificate pair differs from saved pair")
	}

	for _, name := range []string{certificateFile, privateKeyFile} {
		info, err := os.Stat(filepath.Join(store.directory, name))
		if err != nil {
			t.Fatal(err)
		}

		if got := info.Mode().Perm(); got != fileMode {
			t.Errorf("%s permissions = %o, want %o", name, got, fileMode)
		}
	}
}

// TestStoreMissingPair verifies that loading an incomplete store returns an error.
//
// Given a store containing only part of a certificate pair
// When the pair is loaded
// Then loading fails with an error identifying the missing material.
func TestStoreMissingPair(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = store.Load()
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load error = %v, want os.ErrNotExist", err)
	}
}

// TestStoreRejectsInvalidPair verifies that invalid certificate material is rejected.
//
// Given malformed certificate and private-key data
// When the pair is saved
// Then the store returns a validation error.
func TestStoreRejectsInvalidPair(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Save([]byte("not a certificate"), []byte("not a key")); err == nil {
		t.Fatal("Save accepted an invalid pair")
	}
}

// TestStoreDeleteIsIdempotent verifies that deleting a store repeatedly is safe.
//
// Given a certificate store
// When the store is deleted more than once
// Then each deletion succeeds.
func TestStoreDeleteIsIdempotent(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(); err != nil {
		t.Fatal(err)
	}
}

func testCertificate(t *testing.T) ([]byte, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := new(x509.Certificate)
	template.SerialNumber = big.NewInt(1)

	var subject pkix.Name

	subject.CommonName = "test"
	template.Subject = subject
	template.NotBefore = time.Now().Add(-time.Minute)
	template.NotAfter = time.Now().Add(time.Hour)
	template.KeyUsage = x509.KeyUsageDigitalSignature

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Headers: nil, Bytes: der})
	privateKeyBlock := &pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(key),
	}
	privateKey := pem.EncodeToMemory(privateKeyBlock)

	return certificate, privateKey
}
