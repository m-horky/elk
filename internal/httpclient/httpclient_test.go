package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestJoinURLPreservesBasePathAndRemovesSecrets verifies safe URL joining.
//
// Given a base URL containing a service path and credentials
// When a request path is joined to it
// Then the base path is preserved and secrets are not exposed in the result.
func TestJoinURLPreservesBasePathAndRemovesSecrets(t *testing.T) {
	u, _ := url.Parse("https://user:password@example.test/api/?token=secret")

	got, err := JoinURL(u, "/")
	if err != nil {
		t.Fatal(err)
	}

	if got != "https://user:password@example.test/api?token=secret" { //nolint:gosec
		t.Fatalf("JoinURL() = %q", got)
	}

	if sanitizedURL(u) != "https://example.test/api/" {
		t.Fatalf("sanitizedURL() = %q", sanitizedURL(u))
	}
}

// TestGetAcceptsAll2xxAndClosesBody verifies successful response handling.
//
// Given an HTTP endpoint returning a 2xx status
// When the client performs a GET request
// Then the status is accepted and the response body is closed.
func TestGetAcceptsAll2xxAndClosesBody(t *testing.T) {
	for _, status := range []int{200, 204, 299} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))

		c, err := New(Config{BaseURL: ts.URL, TLSVerify: true})
		if err != nil {
			t.Fatal(err)
		}

		if got, err := c.Get(context.Background(), "/"); err != nil || got != status {
			t.Fatalf("status %d: got %d, error %v", status, got, err)
		}

		_ = c.Close()
		ts.Close()
	}
}

// TestGetReturnsBoundedSanitizedHTTPError verifies safe bounded HTTP errors.
//
// Given an HTTP endpoint returning a large error body and sensitive query data
// When the client performs a GET request
// Then the error body is bounded and the rendered URL is sanitized.
func TestGetReturnsBoundedSanitizedHTTPError(t *testing.T) {
	secret := strings.Repeat("x", maxErrorBody+100)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, secret, http.StatusBadGateway) })) //nolint:lll

	defer ts.Close()

	c, err := New(Config{BaseURL: ts.URL, TLSVerify: true})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Get(context.Background(), "/")

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T", err)
	}

	if !httpErr.BodyTruncated || len(httpErr.Body) != maxErrorBody {
		t.Fatalf("bounded error = %d, truncated %v", len(httpErr.Body), httpErr.BodyTruncated)
	}

	if strings.Contains(httpErr.URL, "user:") {
		t.Fatalf("error leaked URL credentials: %v", httpErr)
	}

	_ = c.Close()
}
