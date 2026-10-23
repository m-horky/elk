package imds

import (
	"strings"
	"testing"
	"time"

	"github.com/m-horky/elk/internal/httpclient"
)

// TestNormalizeAppliesDefaults verifies the IMDS defaults.
//
// Given a zero-value HTTP configuration with a configured proxy,
// when it is normalized for IMDS traffic,
// then connect and request timeouts default to 30 seconds, the response cap
// to the IMDS maximum, and the proxy is dropped.
func TestNormalizeAppliesDefaults(t *testing.T) {
	t.Parallel()

	cfg := normalize(httpclient.Config{
		Proxy: httpclient.ProxyConfig{URI: "http://proxy.example.test:3128"},
	})

	if cfg.ConnectTimeout != 30*time.Second {
		t.Errorf("ConnectTimeout = %v, want 30s", cfg.ConnectTimeout)
	}

	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("RequestTimeout = %v, want 30s", cfg.RequestTimeout)
	}

	if cfg.MaxResponseBody != MaxResponseBody {
		t.Errorf("MaxResponseBody = %d, want %d", cfg.MaxResponseBody, MaxResponseBody)
	}

	if cfg.Proxy.URI != "" {
		t.Errorf("Proxy.URI = %q, want empty", cfg.Proxy.URI)
	}
}

// TestNormalizePreservesTighterLimits verifies explicit configuration wins.
//
// Given a configuration with explicit timeouts and a response cap below the
// IMDS maximum,
// when it is normalized,
// then the explicit values are preserved.
func TestNormalizePreservesTighterLimits(t *testing.T) {
	t.Parallel()

	cfg := normalize(httpclient.Config{
		ConnectTimeout:  time.Second,
		RequestTimeout:  2 * time.Second,
		MaxResponseBody: 1024,
	})

	if cfg.ConnectTimeout != time.Second {
		t.Errorf("ConnectTimeout = %v, want 1s", cfg.ConnectTimeout)
	}

	if cfg.RequestTimeout != 2*time.Second {
		t.Errorf("RequestTimeout = %v, want 2s", cfg.RequestTimeout)
	}

	if cfg.MaxResponseBody != 1024 {
		t.Errorf("MaxResponseBody = %d, want 1024", cfg.MaxResponseBody)
	}
}

// TestNormalizeClampsResponseCap verifies the response cap maximum.
//
// Given a configuration with a response cap above the IMDS maximum,
// when it is normalized,
// then the cap is clamped to the IMDS maximum.
func TestNormalizeClampsResponseCap(t *testing.T) {
	t.Parallel()

	cfg := normalize(httpclient.Config{MaxResponseBody: MaxResponseBody + 1})

	if cfg.MaxResponseBody != MaxResponseBody {
		t.Errorf("MaxResponseBody = %d, want %d", cfg.MaxResponseBody, MaxResponseBody)
	}
}

// TestNewClientRejectsInvalidBaseURL verifies construction validates the address.
//
// Given a configuration with an invalid IMDS base URL,
// when NewClient is called,
// then construction fails with a descriptive error.
func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	t.Parallel()

	_, err := NewClient(httpclient.Config{BaseURL: "://invalid"})
	if err == nil {
		t.Fatal("NewClient() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "invalid HTTP base URL") {
		t.Errorf("NewClient() error = %v, want invalid base URL error", err)
	}
}
