// Package candlepin contains the Candlepin client. The API surface is
// intentionally small while the Candlepin wire contracts are being confirmed.
package candlepin

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/m-horky/elk/internal/httpclient"
)

type (
	Config      struct{ HTTP httpclient.Config }
	Certificate = tls.Certificate
)

type Client struct{ http *httpclient.Client }

// NewClient constructs a certificate-authenticated Candlepin client. The
// certificate is supplied by the application or credential package; this
// package does not select or persist certificate files.
func NewClient(cfg Config, certificate Certificate) (*Client, error) {
	if len(certificate.Certificate) == 0 || certificate.PrivateKey == nil {
		return nil, fmt.Errorf("candlepin client certificate is incomplete") //nolint:err113
	}

	cfg.HTTP.Certificate = &certificate

	client, err := httpclient.New(cfg.HTTP)
	if err != nil {
		return nil, fmt.Errorf("construct Candlepin HTTP client: %w", err)
	}

	return &Client{http: client}, nil
}

// NewProbeClient constructs an unauthenticated client for exploratory connectivity checks.
func NewProbeClient(cfg Config) (*Client, error) {
	client, err := httpclient.New(cfg.HTTP)
	if err != nil {
		return nil, fmt.Errorf("construct Candlepin probe client: %w", err)
	}

	return &Client{http: client}, nil
}

// Probe performs a connectivity check against the Candlepin service root.
func (c *Client) Probe(ctx context.Context) (int, error) { return c.http.Get(ctx, "/") } //nolint:wrapcheck

// Close releases the resources held by the Candlepin client.
func (c *Client) Close() error { return c.http.Close() } //nolint:wrapcheck
