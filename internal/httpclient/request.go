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

// defaultMaxResponseBody bounds response bodies.
const defaultMaxResponseBody = 8 * 1024 * 1024 // 8 MiB

// Request describes an HTTP request executed by Client.Do.
type Request struct {
	// Method is the HTTP method. An empty method is treated as GET.
	Method string
	// Path is an escaped URL path resolved against the client's configured base
	// URL. Callers are responsible for escaping user-provided path segments with
	// url.PathEscape before embedding them in Path.
	Path string
	// Query contains optional URL query parameters.
	Query url.Values
	// Headers contains optional request headers.
	Headers http.Header
	// Body is an optional request body.
	Body io.Reader
}

// Response is a successfully read HTTP response returned by Client.Do.
type Response struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// Header contains the response headers.
	Header http.Header
	// Body is the raw response body, bounded by the client's configured limit.
	Body []byte
	// BodyTruncated reports whether Body was truncated at the configured limit.
	BodyTruncated bool
}

// Get performs a GET request for the specified path and returns its successful
// response. Non-2xx responses return an *HTTPError. The response body is
// bounded by Config.MaxResponseBody.
func (c *Client) Get(ctx context.Context, p string) (*Response, error) {
	return c.Do(ctx, Request{Method: http.MethodGet, Path: p})
}

// NewJSONRequest builds a request whose body is the JSON encoding of body.
// The returned request sets the JSON content type; callers may add further
// headers before passing it to Client.Do.
func NewJSONRequest(method, p string, query url.Values, body any) (Request, error) {
	request := Request{Method: method, Path: p, Query: query}
	if body == nil {
		return request, nil
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return Request{}, fmt.Errorf("encode JSON request: %w", err)
	}

	request.Headers = http.Header{"Content-Type": []string{"application/json"}}
	request.Body = bytes.NewReader(encoded)

	return request, nil
}

// Do executes request and returns its response.
//
// Non-2xx responses return an *HTTPError carrying a bounded diagnostic body.
//
// A successful response that exceeds Config.MaxResponseBody is
// returned with a truncated Body, BodyTruncated set, and an error.
func (c *Client) Do(ctx context.Context, request Request) (*Response, error) {
	resp, err := c.execute(ctx,
		request.Method, request.Path, request.Query, request.Body, request.Headers)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	success := isSuccessStatus(resp.StatusCode)

	limit := c.maxResponseBody
	if !success {
		limit = min(limit, maxErrorBody)
	}

	body, truncated, err := readBoundedBody(resp.Body, limit)
	if err != nil {
		return nil, fmt.Errorf("read %s response body: %w", requestDescription(resp), err)
	}

	if !success {
		return nil, buildHTTPError(resp, body, truncated)
	}

	response := &Response{
		StatusCode:    resp.StatusCode,
		Header:        resp.Header,
		Body:          body,
		BodyTruncated: truncated,
	}

	if truncated {
		return response, fmt.Errorf("%s: response body exceeds %d bytes", requestDescription(resp), limit)
	}

	return response, nil
}

// newRequest builds an HTTP request for a path resolved against the configured base URL.
func (c *Client) newRequest(
	ctx context.Context, method, p string, query url.Values, body io.Reader, headers http.Header,
) (*http.Request, error) {
	u, err := c.URL(p)
	if err != nil {
		return nil, err
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("parse request URL: %w", err)
	}

	parsed.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, parsed.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}

	for key, values := range headers {
		req.Header[key] = append([]string(nil), values...)
	}

	if c.basicAuth != nil {
		req.SetBasicAuth(c.basicAuth.Username, c.basicAuth.Password)
	}

	return req, nil
}

// execute builds and executes an HTTP request, returning the response for the caller to consume.
func (c *Client) execute(
	ctx context.Context, method, p string, query url.Values, body io.Reader, headers http.Header,
) (*http.Response, error) {
	req, err := c.newRequest(ctx, method, p, query, body, headers)
	if err != nil {
		return nil, err
	}

	slog.Debug("HTTP request started", "method", req.Method, "url", sanitizedURL(req.URL), "body", body != nil)

	started := time.Now()

	resp, err := c.client.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

		slog.Debug("HTTP request failed", "method", req.Method, "url", sanitizedURL(req.URL),
			"duration", time.Since(started), "err", err)

		return nil, err //nolint:wrapcheck
	}

	slog.Debug("HTTP response received", "method", req.Method, "url", sanitizedURL(req.URL),
		"status", resp.StatusCode, "duration", time.Since(started))

	return resp, nil
}

// isSuccessStatus reports whether status is in the successful 2xx range.
func isSuccessStatus(status int) bool {
	return status >= 200 && status < 300
}

// readBoundedBody reads at most limit bytes and reports whether the body was
// truncated. A non-positive limit uses the package default; read failures are
// the only errors.
func readBoundedBody(r io.Reader, limit int64) ([]byte, bool, error) {
	if limit <= 0 {
		limit = defaultMaxResponseBody
	}

	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, false, fmt.Errorf("read response body: %w", err)
	}

	if int64(len(body)) > limit {
		return body[:limit], true, nil
	}

	return body, false, nil
}

// requestDescription identifies a response request for error messages.
func requestDescription(resp *http.Response) string {
	if resp.Request == nil {
		return "HTTP"
	}

	return resp.Request.Method + " " + sanitizedURL(resp.Request.URL)
}
