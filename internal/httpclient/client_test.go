package httpclient

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestNewConfiguresTransport verifies client construction applies transport settings.
//
// Given a client configuration with explicit connection and idle timeouts
// When a client is constructed
// Then its transport contains those settings and can be closed successfully.
func TestNewConfiguresTransport(t *testing.T) {
	t.Parallel()

	client, err := New(Config{
		BaseURL:        "https://example.test/api",
		ConnectTimeout: 3 * time.Second,
		IdleTimeout:    7 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	if client.transport.DialContext == nil {
		t.Fatal("DialContext is nil")
	}

	if client.transport.IdleConnTimeout != 7*time.Second {
		t.Fatalf("IdleConnTimeout = %s, want 7s", client.transport.IdleConnTimeout)
	}

	if client.transport.TLSClientConfig == nil || client.transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatal("transport TLS configuration does not require TLS 1.2")
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

// TestNewRejectsInvalidProxy verifies invalid proxy configuration is reported.
//
// Given a proxy URI that cannot be parsed
// When a client is constructed
// Then construction returns a descriptive error.
func TestNewRejectsInvalidProxy(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		BaseURL: "https://example.test",
		Proxy:   ProxyConfig{URI: "://invalid"},
	})
	if err == nil || !strings.Contains(err.Error(), "parse proxy URL") {
		t.Fatalf("New() error = %v, want proxy parse error", err)
	}
}

// TestRedirectPolicyRestrictsOrigins verifies redirect handling remains origin-bound.
//
// Given a client with redirects enabled and a configured origin
// When its redirect policy evaluates same-origin and cross-origin redirects
// Then it accepts the former and rejects the latter.
func TestRedirectPolicyRestrictsOrigins(t *testing.T) {
	t.Parallel()

	base, _ := url.Parse("https://example.test/api")
	transport := http.DefaultTransport.(*http.Transport).Clone()
	client := newHTTPClient(Config{AllowRedirects: true}, base, transport)

	sameOrigin, _ := http.NewRequest(http.MethodGet, "https://EXAMPLE.TEST/api/next", nil)
	if err := client.CheckRedirect(sameOrigin, nil); err != nil {
		t.Fatalf("same-origin redirect error = %v", err)
	}

	otherOrigin, _ := http.NewRequest(http.MethodGet, "https://other.test/api/next", nil)
	if err := client.CheckRedirect(otherOrigin, nil); err == nil {
		t.Fatal("cross-origin redirect was accepted")
	}
}

// TestRedirectPolicyDisablesRedirects verifies disabled redirects return the last response.
//
// Given a client with redirects disabled
// When its redirect policy evaluates a redirect
// Then it returns http.ErrUseLastResponse.
func TestRedirectPolicyDisablesRedirects(t *testing.T) {
	t.Parallel()

	base, _ := url.Parse("https://example.test")
	transport := http.DefaultTransport.(*http.Transport).Clone()
	client := newHTTPClient(Config{}, base, transport)
	request, _ := http.NewRequest(http.MethodGet, "https://example.test/next", nil)

	if err := client.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect error = %v, want %v", err, http.ErrUseLastResponse)
	}
}

// TestNewUsesProxyCredentials verifies configured proxy credentials are attached.
//
// Given a proxy URI and credentials
// When a client is constructed
// Then the transport proxy function returns a URL containing those credentials.
func TestNewUsesProxyCredentials(t *testing.T) {
	t.Parallel()

	client, err := New(Config{
		BaseURL: "https://example.test",
		Proxy: ProxyConfig{
			URI:      "http://proxy.example.test:3128",
			Username: "user",
			Password: "secret",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	request, _ := http.NewRequest(http.MethodGet, "https://example.test", nil)

	proxy, err := client.transport.Proxy(request)
	if err != nil {
		t.Fatal(err)
	}

	if proxy == nil || proxy.User.Username() != "user" {
		t.Fatalf("proxy = %v, want credentials", proxy)
	}

	if password, ok := proxy.User.Password(); !ok || password != "secret" {
		t.Fatalf("proxy password = %q, want secret", password)
	}
}
