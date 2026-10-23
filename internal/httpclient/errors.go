package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxErrorBody = 4 << 10

type HTTPError struct {
	Method, URL, Status string
	StatusCode          int
	Body                []byte
	BodyTruncated       bool
}

// Error returns the HTTP error as a formatted method, URL, status, and body message.
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

// readErrorBody reads at most maxErrorBody bytes and reports whether the body was truncated.
func readErrorBody(r io.Reader) ([]byte, bool) {
	body, truncated, err := readBoundedBody(r, maxErrorBody)
	if err != nil {
		return []byte("unable to read error body"), false
	}

	return body, truncated
}

// buildHTTPError assembles an *HTTPError from a response and a bounded body.
func buildHTTPError(resp *http.Response, body []byte, truncated bool) *HTTPError {
	httpErr := &HTTPError{
		Status:        resp.Status,
		StatusCode:    resp.StatusCode,
		Body:          body,
		BodyTruncated: truncated,
	}

	if resp.Request != nil {
		httpErr.Method = resp.Request.Method
		httpErr.URL = sanitizedURL(resp.Request.URL)
	}

	return httpErr
}
