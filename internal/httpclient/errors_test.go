package httpclient

import (
	"errors"
	"strings"
	"testing"
)

// TestHTTPErrorErrorFormatsBody verifies HTTP errors include status and bounded body details.
//
// Given an HTTP error with a response body
// When it is formatted
// Then the method, URL, status, body, and truncation marker are rendered.
func TestHTTPErrorErrorFormatsBody(t *testing.T) {
	t.Parallel()

	err := &HTTPError{
		Method:        "GET",
		URL:           "https://example.test/resource",
		Status:        "502 Bad Gateway",
		Body:          []byte(" upstream failure "),
		BodyTruncated: true,
	}

	got := err.Error()

	want := "GET https://example.test/resource: 502 Bad Gateway: upstream failure (truncated)"
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

// TestHTTPErrorErrorOmitsEmptyBody verifies empty HTTP error bodies are omitted.
//
// Given an HTTP error without a response body
// When it is formatted
// Then the result contains only the request and status information.
func TestHTTPErrorErrorOmitsEmptyBody(t *testing.T) {
	t.Parallel()

	err := (&HTTPError{Method: "GET", URL: "https://example.test", Status: "404 Not Found"}).Error()
	if err != "GET https://example.test: 404 Not Found" {
		t.Fatalf("Error() = %q", err)
	}
}

// TestReadErrorBodyBoundsInput verifies error bodies are limited to maxErrorBody.
//
// Given a body larger than the configured error limit
// When it is read
// Then only maxErrorBody bytes are returned and truncation is reported.
func TestReadErrorBodyBoundsInput(t *testing.T) {
	t.Parallel()

	body, truncated := readErrorBody(strings.NewReader(strings.Repeat("x", maxErrorBody+1)))
	if !truncated || len(body) != maxErrorBody {
		t.Fatalf("readErrorBody() = (%d, %v), want (%d, true)", len(body), truncated, maxErrorBody)
	}
}

// TestReadErrorBodyReportsReadFailure verifies read failures produce a safe fallback body.
//
// Given a reader that fails while being read
// When the error body is read
// Then a safe fallback message is returned without reporting truncation.
func TestReadErrorBodyReportsReadFailure(t *testing.T) {
	t.Parallel()

	body, truncated := readErrorBody(errorReader{})
	if truncated || string(body) != "unable to read error body" {
		t.Fatalf("readErrorBody() = (%q, %v)", body, truncated)
	}
}

type errorReader struct{}

// Read always returns an error for testing read-failure handling.
func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
