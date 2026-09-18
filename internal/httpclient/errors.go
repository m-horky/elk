package httpclient

import (
	"fmt"
	"io"
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
	b, err := io.ReadAll(io.LimitReader(r, maxErrorBody+1))
	if err != nil {
		return []byte("unable to read error body"), false
	}

	if len(b) > maxErrorBody {
		return b[:maxErrorBody], true
	}

	return b, false
}
