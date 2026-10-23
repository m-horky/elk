package constants

import (
	"testing"
)

// TestAddressDefault verifies address resolution falls back to the provider defaults.
//
// Given empty environment variables, when IMDS addresses are resolved,
// then the default base URLs are returned.
func TestAddressDefault(t *testing.T) {
	t.Parallel()

	tests := map[addressID]string{
		AWSIMDSAddress:    "http://169.254.169.254",
		AzureIMDSAddress:  "http://169.254.169.254",
		GoogleIMDSAddress: "http://metadata.google.internal",
	}

	for id, want := range tests {
		if got := Address(id); got != want {
			t.Errorf("Address(%q) = %q, want %q", id, got, want)
		}
	}
}

// TestAddressOverride verifies a non-empty environment override wins.
//
// Given a set environment variable, when an IMDS address is resolved,
// then the configured value is returned.
func TestAddressOverride(t *testing.T) {
	t.Setenv(string(AWSIMDSAddress), "http://127.0.0.1:5000")

	if got := Address(AWSIMDSAddress); got != "http://127.0.0.1:5000" {
		t.Errorf("Address(AWSIMDSAddress) = %q, want http://127.0.0.1:5000", got)
	}

	if got := Address(AzureIMDSAddress); got != "http://169.254.169.254" {
		t.Errorf("Address(AzureIMDSAddress) = %q, want http://169.254.169.254", got)
	}
}

// TestAddressIPv6Literal verifies IPv6 bracket literals are returned verbatim.
//
// Given an IPv6 override, when the address is resolved,
// then the bracketed literal is returned unchanged.
func TestAddressIPv6Literal(t *testing.T) {
	t.Setenv(string(AWSIMDSAddress), "http://[fd00:ec2::254]")

	if got := Address(AWSIMDSAddress); got != "http://[fd00:ec2::254]" {
		t.Errorf("Address(AWSIMDSAddress) = %q, want http://[fd00:ec2::254]", got)
	}
}
