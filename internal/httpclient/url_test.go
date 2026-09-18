package httpclient

import (
	"net/url"
	"strings"
	"testing"
)

// TestParseBaseURL validates accepted and rejected HTTP base URLs.
//
// Given valid and invalid base URL strings
// When they are parsed
// Then only HTTP and HTTPS URLs with hosts are accepted.
func TestParseBaseURL(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		raw  string
		want bool
	}{
		{name: "https", raw: "https://example.test", want: true},
		{name: "http", raw: "http://example.test", want: true},
		{name: "missing host", raw: "https://", want: false},
		{name: "unsupported scheme", raw: "ftp://example.test", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseBaseURL(test.raw)
			if (err == nil) != test.want {
				t.Fatalf("parseBaseURL(%q) error = %v, want success: %v", test.raw, err, test.want)
			}
		})
	}
}

// TestJoinURLPreservesBasePathAndRemovesSecrets verifies safe URL joining.
//
// Given a base URL containing a service path and credentials
// When a request path is joined to it
// Then the base path is preserved and secrets are not exposed in the sanitized result.
func TestJoinURLPreservesBasePathAndRemovesSecrets(t *testing.T) {
	u, _ := url.Parse("https://user:password@example.test/api/?token=secret")

	got, err := JoinURL(u, "/")
	if err != nil {
		t.Fatal(err)
	}

	if got != "https://user:password@example.test/api?token=secret" { //nolint:gosec
		t.Fatalf("JoinURL() = %q", got)
	}

	if sanitizedURL(u) != "https://example.test/api/" {
		t.Fatalf("sanitizedURL() = %q", sanitizedURL(u))
	}
}

// TestJoinURLRejectsNilBase verifies nil base URLs are rejected.
//
// Given a nil base URL
// When a path is joined
// Then JoinURL returns an error instead of panicking.
func TestJoinURLRejectsNilBase(t *testing.T) {
	t.Parallel()

	if _, err := JoinURL(nil, "/"); err == nil {
		t.Fatal("JoinURL(nil) returned no error")
	}
}

// TestClientURLUsesConfiguredBase verifies the client URL method delegates to JoinURL.
//
// Given a client configured with a base path
// When its URL method resolves a request path
// Then the configured base path is retained.
func TestClientURLUsesConfiguredBase(t *testing.T) {
	t.Parallel()

	base, _ := url.Parse("https://example.test/service")
	client := &Client{base: base}

	got, err := client.URL("/status")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(got, "/service/status") {
		t.Fatalf("Client.URL() = %q, want service path", got)
	}
}
