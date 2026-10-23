package httpclient

import (
	"crypto/tls"
	"net/http"
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

// TestNewHonorsNoProxy verifies the no-proxy bypass rules are applied.
//
// Given a proxy URI and a no-proxy rule matching the request host
// When a client is constructed
// Then the transport proxy function returns no proxy for that host.
func TestNewHonorsNoProxy(t *testing.T) {
	t.Parallel()

	client, err := New(Config{
		BaseURL: "https://example.test",
		Proxy: ProxyConfig{
			URI:      "http://proxy.example.test:3128",
			Username: "user",
			Password: "secret",
			NoProxy:  "example.test",
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

	if proxy != nil {
		t.Fatalf("proxy = %v, want nil for a bypassed host", proxy)
	}
}
