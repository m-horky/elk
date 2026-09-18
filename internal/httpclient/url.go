package httpclient

import (
	"errors"
	"net/url"
	"path"
)

// parseBaseURL validates and parses an HTTP or HTTPS service URL.
func parseBaseURL(raw string) (*url.URL, error) {
	base, err := url.Parse(raw)
	if err != nil || base.Scheme == "" || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil, errors.New("invalid HTTP base URL")
	}

	return base, nil
}

// URL resolves a path against the client's configured base URL.
func (c *Client) URL(p string) (string, error) { return JoinURL(c.base, p) }

// JoinURL resolves a path against a base URL while preserving the base path.
func JoinURL(base *url.URL, p string) (string, error) {
	if base == nil {
		return "", errors.New("nil base URL")
	}

	u := *base
	// Preserve a configured base path: the probe's "/" is the service root.
	u.Path = path.Join("/", base.Path, p)
	u.RawPath = ""

	return u.String(), nil
}

// sanitizedURL removes credentials and request data before a URL is logged or returned in an error.
func sanitizedURL(u *url.URL) string {
	v := *u
	v.User = nil
	v.RawQuery = ""
	v.Fragment = ""

	return v.String()
}
