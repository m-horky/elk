package imds

import (
	"fmt"
	"time"

	"github.com/m-horky/elk/internal/httpclient"
)

const (
	// defaultTimeout bounds both connecting to and requesting from an IMDS.
	defaultTimeout = time.Second
	// MaxResponseBody caps every IMDS response; all real responses are tiny.
	MaxResponseBody = 512 * 1024 // 512 KiB
)

// NewClient constructs the HTTP client shared by the provider IMDS clients.
// Connect and request timeouts use defaultTimeout if not set, responses are
// capped at MaxResponseBody, and configured proxies are ignored so metadata
// traffic stays on the link-local path.
func NewClient(cfg httpclient.Config) (*httpclient.Client, error) {
	client, err := httpclient.New(normalize(cfg))
	if err != nil {
		return nil, fmt.Errorf("construct IMDS HTTP client: %w", err)
	}

	return client, nil
}

// normalize applies the shared IMDS defaults to an HTTP client configuration.
func normalize(cfg httpclient.Config) httpclient.Config {
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = defaultTimeout
	}

	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultTimeout
	}

	if cfg.MaxResponseBody <= 0 || cfg.MaxResponseBody > MaxResponseBody {
		cfg.MaxResponseBody = MaxResponseBody
	}

	// Ignore proxy, if set; IMDS needs to be accessed locally.
	cfg.Proxy.URI = ""

	return cfg
}
