// Package httpclient contains the exploratory shared HTTP transport used by
// the draft endpoint probe. It is not the final service API.
package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

const maxErrorBody = 4 << 10

type Config struct {
	BaseURL        string
	CAFile         string
	TLSVerify      bool
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

type ProxyConfig struct{ URI, Username, Password, NoProxy string }

type Client struct {
	client    *http.Client
	transport *http.Transport
	base      *url.URL
	basicAuth *BasicAuth
}

type HTTPError struct {
	Method, URL, Status string
	StatusCode          int
	Body                []byte
	BodyTruncated       bool
}

func (e *HTTPError) Error() string {
	body := strings.TrimSpace(string(e.Body))
	if e.BodyTruncated {
		body += " (truncated)"
	}

	if body == "" {
		return fmt.Sprintf("%s %s: %s", e.Method, e.URL, e.Status)
	}

	return fmt.Sprintf("%s %s: %s: %s", e.Method, e.URL, e.Status, body)
}

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

	slog.Debug("HTTP client configured", "base_url", sanitizedURL(base), "tls_verify", cfg.TLSVerify, "insecure_tls", cfg.InsecureTLS, "custom_ca", roots != nil, "proxy", cfg.Proxy.URI != "", "request_timeout", cfg.RequestTimeout)
	tr := newTransport(cfg, roots)
	cl := newHTTPClient(cfg, base, tr)

	return &Client{client: cl, transport: tr, base: base, basicAuth: cfg.BasicAuth}, nil
}

func parseBaseURL(raw string) (*url.URL, error) {
	base, err := url.Parse(raw)
	if err != nil || base.Scheme == "" || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil, fmt.Errorf("invalid HTTP base URL") //nolint:err113
	}

	return base, nil
}

func loadRoots(path string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(path) // #nosec G304: path is explicitly configured by the operator.
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return nil, errors.New("CA file contains no certificates") //nolint:err113
	}

	return roots, nil
}

func newTransport(cfg Config, roots *x509.CertPool) *http.Transport {
	tlsConfig := new(tls.Config)

	tlsConfig.MinVersion = tls.VersionTLS12
	tlsConfig.RootCAs = roots
	tlsConfig.InsecureSkipVerify = cfg.InsecureTLS // #nosec G402: insecure mode is an explicit configuration choice.

	if cfg.Certificate != nil {
		tlsConfig.Certificates = []tls.Certificate{*cfg.Certificate}
	}

	tr := new(http.Transport)
	tr.TLSClientConfig = tlsConfig
	tr.DialContext = (&netDialer{timeout: cfg.ConnectTimeout}).DialContext
	tr.IdleConnTimeout = cfg.IdleTimeout

	tr.Proxy = http.ProxyFromEnvironment

	if cfg.Proxy.NoProxy != "" || cfg.Proxy.URI != "" {
		tr.Proxy = proxyFunc(cfg.Proxy)
	}

	return tr
}

func newHTTPClient(cfg Config, base *url.URL, tr *http.Transport) *http.Client {
	cl := new(http.Client)
	cl.Transport = tr
	cl.Timeout = cfg.RequestTimeout
	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !cfg.AllowRedirects {
			slog.Debug("HTTP redirect not followed", "url", sanitizedURL(req.URL), "reason", "redirects disabled")
			return http.ErrUseLastResponse
		}

		if req.URL.Scheme != base.Scheme || !strings.EqualFold(req.URL.Host, base.Host) {
			slog.Debug("HTTP redirect rejected", "url", sanitizedURL(req.URL), "reason", "origin not approved")
			return fmt.Errorf("redirect to unapproved origin %q", sanitizedURL(req.URL)) //nolint:err113
		}

		slog.Debug("HTTP redirect followed", "url", sanitizedURL(req.URL))
		return nil
	}

	return cl
}

// netDialer and proxyFunc are kept here to avoid exposing transport details.
type netDialer struct{ timeout time.Duration }

func (d *netDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := new(net.Dialer)
	dialer.Timeout = d.timeout

	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}

	return conn, nil
}

func proxyFunc(cfg ProxyConfig) func(*http.Request) (*url.URL, error) {
	return func(r *http.Request) (*url.URL, error) {
		for host := range strings.SplitSeq(cfg.NoProxy, ",") {
			if strings.TrimSpace(host) == r.URL.Hostname() {
				return nil, nil
			}
		}

		u, err := url.Parse(cfg.URI)
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		if cfg.Username != "" {
			u.User = url.UserPassword(cfg.Username, cfg.Password)
		}

		return u, nil
	}
}

func (c *Client) URL(p string) (string, error) { return JoinURL(c.base, p) }
func JoinURL(base *url.URL, p string) (string, error) {
	if base == nil {
		return "", errors.New("nil base URL") //nolint:err113
	}

	u := *base
	// Preserve a configured base path: the probe's "/" is the service root.
	u.Path = path.Join("/", base.Path, p)
	u.RawPath = ""

	return u.String(), nil
}

func (c *Client) Get(ctx context.Context, p string) (int, error) {
	return c.do(ctx, http.MethodGet, p, nil, nil, nil)
}

// DoJSON executes a JSON request and decodes a successful JSON response. A
// nil response target permits successful empty responses.
func (c *Client) DoJSON(ctx context.Context, method, p string, query url.Values, requestBody, responseBody any) (int, error) { //nolint:lll
	var body io.Reader

	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return 0, fmt.Errorf("encode JSON request: %w", err)
		}

		body = bytes.NewReader(encoded)
	}

	return c.do(ctx, method, p, query, body, responseBody)
}

//nolint:lll
func (c *Client) do(ctx context.Context, method, p string, query url.Values, body io.Reader, responseBody any) (int, error) { //nolint:funcorder
	u, err := c.URL(p)
	if err != nil {
		return 0, err
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return 0, fmt.Errorf("parse request URL: %w", err)
	}

	parsed.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, parsed.String(), body)
	if err != nil {
		return 0, fmt.Errorf("create HTTP request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.basicAuth != nil {
		req.SetBasicAuth(c.basicAuth.Username, c.basicAuth.Password)
	}

	slog.Debug("HTTP request started", "method", req.Method, "url", sanitizedURL(req.URL), "body", body != nil)
	started := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

		slog.Debug("HTTP request failed", "method", req.Method, "url", sanitizedURL(req.URL), "duration", time.Since(started), "err", err)
		return 0, err //nolint:wrapcheck
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Debug("HTTP response received", "method", req.Method, "url", sanitizedURL(req.URL), "status", resp.StatusCode, "duration", time.Since(started))
		b, trunc := readErrorBody(resp.Body)

		return resp.StatusCode, &HTTPError{
			Method:        req.Method,
			URL:           sanitizedURL(req.URL),
			Status:        resp.Status,
			StatusCode:    resp.StatusCode,
			Body:          b,
			BodyTruncated: trunc,
		}
	}

	if responseBody == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		slog.Debug("HTTP response received", "method", req.Method, "url", sanitizedURL(req.URL), "status", resp.StatusCode, "duration", time.Since(started))

		return resp.StatusCode, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
		slog.Debug("HTTP response JSON decode failed", "method", req.Method, "url", sanitizedURL(req.URL), "status", resp.StatusCode, "duration", time.Since(started), "err", err)
		return resp.StatusCode, fmt.Errorf("decode JSON response: %w", err)
	}

	slog.Debug("HTTP response received", "method", req.Method, "url", sanitizedURL(req.URL), "status", resp.StatusCode, "duration", time.Since(started))
	return resp.StatusCode, nil
}

func readErrorBody(r io.Reader) ([]byte, bool) {
	b, err := io.ReadAll(io.LimitReader(r, maxErrorBody+1))
	if err != nil {
		return []byte("unable to read error body"), false
	}

	if len(b) > maxErrorBody {
		return b[:maxErrorBody], true
	}

	return b, false
}

func sanitizedURL(u *url.URL) string {
	v := *u
	v.User = nil
	v.RawQuery = ""
	v.Fragment = ""

	return v.String()
}

func (c *Client) Close() error { c.transport.CloseIdleConnections(); return nil } //nolint:nlreturn
