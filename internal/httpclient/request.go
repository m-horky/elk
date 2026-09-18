package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Get performs a GET request for the specified path.
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

// do performs an HTTP request and handles its response.
//
//nolint:lll
func (c *Client) do(ctx context.Context, method, p string, query url.Values, body io.Reader, responseBody any) (int, error) { //nolint:funcorder,funlen
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
