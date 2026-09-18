package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	BaseURL        string
	CAFile         string
	InsecureTLS    bool
	ConnectTimeout time.Duration
	RequestTimeout time.Duration
	IdleTimeout    time.Duration
	Proxy          ProxyConfig
	AllowRedirects bool // restricted to the configured origin
	Certificate    *tls.Certificate
	BasicAuth      *BasicAuth
}

type BasicAuth struct {
	Username string
	Password string
}

type ProxyConfig struct{ URI, Username, Password string }

type Client struct {
	client    *http.Client
	transport *http.Transport
	base      *url.URL
	basicAuth *BasicAuth
}

// New constructs an HTTP client from the supplied configuration.
func New(cfg Config) (*Client, error) {
	base, err := parseBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 30 * time.Second
	}

	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = 120 * time.Second
	}

	var roots *x509.CertPool
	if cfg.CAFile != "" && cfg.CAFile != "system" {
		roots, err = loadRoots(cfg.CAFile)
		if err != nil {
			return nil, err
		}
	}

	slog.Debug("HTTP client configured", "base_url", sanitizedURL(base),
		"insecure_tls", cfg.InsecureTLS,
		"custom_ca", roots != nil, "proxy", cfg.Proxy.URI != "",
		"request_timeout", cfg.RequestTimeout)

	tr, err := newTransport(cfg, roots)
	if err != nil {
		return nil, err
	}

	cl := newHTTPClient(cfg, base, tr)

	return &Client{client: cl, transport: tr, base: base, basicAuth: cfg.BasicAuth}, nil
}

// loadRoots reads and parses a PEM-encoded certificate authority bundle.
func loadRoots(path string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(path) // #nosec G304: path is explicitly configured by the operator.
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return nil, errors.New("CA file contains no certificates")
	}

	return roots, nil
}

// newTransport builds an HTTP transport with the configured TLS, proxy, and timeout settings.
func newTransport(cfg Config, roots *x509.CertPool) (*http.Transport, error) {
	tlsConfig := new(tls.Config)

	tlsConfig.MinVersion = tls.VersionTLS12
	tlsConfig.RootCAs = roots
	tlsConfig.InsecureSkipVerify = cfg.InsecureTLS // #nosec G402: insecure mode is an explicit configuration choice.

	if cfg.Certificate != nil {
		tlsConfig.Certificates = []tls.Certificate{*cfg.Certificate}
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = tlsConfig
	dialer := &net.Dialer{Timeout: cfg.ConnectTimeout}
	tr.DialContext = dialer.DialContext
	tr.IdleConnTimeout = cfg.IdleTimeout
	tr.Proxy = http.ProxyFromEnvironment

	// TODO: Honor the configuration file's no-proxy setting when proxy bypass support is added.
	if cfg.Proxy.URI != "" {
		u, err := url.Parse(cfg.Proxy.URI)
		if err != nil {
			return nil, fmt.Errorf("parse proxy URL: %w", err)
		}

		if cfg.Proxy.Username != "" {
			u.User = url.UserPassword(cfg.Proxy.Username, cfg.Proxy.Password)
		}

		tr.Proxy = http.ProxyURL(u)
	}

	return tr, nil
}

// newHTTPClient builds an HTTP client with request and same-origin redirect policies.
func newHTTPClient(cfg Config, base *url.URL, tr *http.Transport) *http.Client {
	cl := new(http.Client)
	cl.Transport = withUserAgent(tr)
	cl.Timeout = cfg.RequestTimeout
	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !cfg.AllowRedirects {
			slog.Debug("HTTP redirect not followed", "url", sanitizedURL(req.URL), "reason", "redirects disabled")

			return http.ErrUseLastResponse
		}

		if req.URL.Scheme != base.Scheme || !strings.EqualFold(req.URL.Host, base.Host) {
			slog.Debug("HTTP redirect rejected", "url", sanitizedURL(req.URL), "reason", "origin not approved")

			return fmt.Errorf("redirect to unapproved origin %q", sanitizedURL(req.URL))
		}

		slog.Debug("HTTP redirect followed", "url", sanitizedURL(req.URL))

		return nil
	}

	return cl
}

// Close releases idle connections held by the client.
func (c *Client) Close() error { c.transport.CloseIdleConnections(); return nil } //nolint:nlreturn
