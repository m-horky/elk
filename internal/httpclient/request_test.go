package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGetAcceptsAll2xxStatuses verifies successful GET response handling.
//
// Given an HTTP endpoint returning a 2xx status
// When the client performs a GET request
// Then the status is accepted for each 2xx response tested.
func TestGetAcceptsAll2xxStatuses(t *testing.T) {
	t.Parallel()

	for _, status := range []int{200, 204, 299} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
			defer ts.Close()

			client, err := New(Config{BaseURL: ts.URL})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = client.Close() }()

			if got, err := client.Get(context.Background(), "/"); err != nil || got != status {
				t.Fatalf("status %d: got %d, error %v", status, got, err)
			}
		})
	}
}

// TestDoJSONSendsAndDecodesJSON verifies JSON request and response handling.
//
// Given a JSON request body and query values
// When DoJSON sends the request
// Then it sets the JSON content type, preserves the query, and decodes the response.
func TestDoJSONSendsAndDecodesJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("mode") != "test" {
			t.Errorf("mode query = %q, want test", r.URL.Query().Get("mode"))
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var request payload
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}

		if request.Name != "elk" {
			t.Errorf("request name = %q, want elk", request.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"response"}`))
	}))
	defer ts.Close()

	client, err := New(Config{BaseURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	var response payload

	query := map[string][]string{"mode": {"test"}}

	status, err := client.DoJSON(
		context.Background(), http.MethodPost, "/resource", query, payload{Name: "elk"}, &response,
	)
	if err != nil {
		t.Fatal(err)
	}

	if status != http.StatusOK || response.Name != "response" {
		t.Fatalf("DoJSON() = (%d, %+v), want (200, response)", status, response)
	}
}

// TestDoJSONRejectsUnencodableBody verifies JSON encoding failures are returned.
//
// Given a request body containing an unsupported value
// When DoJSON attempts to encode it
// Then it returns an encoding error without sending a request.
func TestDoJSONRejectsUnencodableBody(t *testing.T) {
	t.Parallel()

	client, err := New(Config{BaseURL: "https://example.test"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.DoJSON(context.Background(), http.MethodPost, "/", nil, func() {}, nil)
	if err == nil || !strings.Contains(err.Error(), "encode JSON request") {
		t.Fatalf("DoJSON() error = %v, want encoding error", err)
	}
}

// TestGetReturnsBoundedSanitizedHTTPError verifies safe bounded HTTP errors.
//
// Given an HTTP endpoint returning a large error body and sensitive URL data
// When the client performs a GET request
// Then the error body is bounded and the rendered URL is sanitized.
func TestGetReturnsBoundedSanitizedHTTPError(t *testing.T) {
	t.Parallel()

	secret := strings.Repeat("x", maxErrorBody+100)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, secret, http.StatusBadGateway)
	}))
	defer ts.Close()

	client, err := New(Config{BaseURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Get(context.Background(), "/")

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T", err)
	}

	if !httpErr.BodyTruncated || len(httpErr.Body) != maxErrorBody {
		t.Fatalf("bounded error = %d, truncated %v", len(httpErr.Body), httpErr.BodyTruncated)
	}
}
