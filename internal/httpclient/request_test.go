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

// TestGetAccepts2xx verifies successful GET response handling.
//
// Given an HTTP endpoint returning a 2xx status, when the client performs
// a GET request, then the status is accepted for each 2xx response tested.
func TestGetAccepts2xx(t *testing.T) {
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

			response, err := client.Get(context.Background(), "/")
			if err != nil {
				t.Fatalf("status %d: error %v", status, err)
			}

			if response.StatusCode != status {
				t.Fatalf("status %d: got %d", status, response.StatusCode)
			}
		})
	}
}

// TestGetReturnsBodyAndHeaders verifies GET responses expose their body and headers.
//
// Given an HTTP endpoint returning a body and a custom header, when the client performs
// a GET request, then the returned response contains the body and a header.
func TestGetReturnsBodyAndHeaders(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "value")
		_, _ = w.Write([]byte("body"))
	}))
	defer ts.Close()

	client, err := New(Config{BaseURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	response, err := client.Get(context.Background(), "/")
	if err != nil {
		t.Fatal(err)
	}

	if string(response.Body) != "body" {
		t.Fatalf("body = %q, want body", response.Body)
	}

	if got := response.Header.Get("X-Test"); got != "value" {
		t.Fatalf("X-Test = %q, want value", got)
	}
}

// TestNewJSONRequest verifies JSON request construction.
//
// Given a JSON request body and query values, when NewJSONRequest builds a request,
// then it sets the JSON content type, preserves the query, and returns the raw response.
func TestNewJSONRequest(t *testing.T) {
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

	query := map[string][]string{"mode": {"test"}}

	httpRequest, err := NewJSONRequest(http.MethodPost, "/resource", query, payload{Name: "elk"})
	if err != nil {
		t.Fatal(err)
	}

	response, err := client.Do(context.Background(), httpRequest)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK || string(response.Body) != `{"name":"response"}` {
		t.Fatalf("Do() = (%d, %s), want (200, response)", response.StatusCode, response.Body)
	}
}

// TestNewJSONWithoutBody verifies nil bodies are omitted.
//
// Given a nil JSON body, when NewJSONRequest builds a request,
// then the request has no body and no JSON content type.
func TestNewJSONWithoutBody(t *testing.T) {
	t.Parallel()

	request, err := NewJSONRequest(http.MethodGet, "/", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if request.Body != nil {
		t.Fatalf("body = %v, want nil", request.Body)
	}

	if got := request.Headers.Get("Content-Type"); got != "" {
		t.Fatalf("Content-Type = %q, want empty", got)
	}
}

// TestNewJSONRequestRejectsUnencodableBody verifies JSON encoding failures are returned.
//
// Given a request body containing an unsupported value, when NewJSONRequest attempts
// to encode it, then it returns an encoding error.
func TestNewJSONRequestRejectsUnencodableBody(t *testing.T) {
	t.Parallel()

	_, err := NewJSONRequest(http.MethodPost, "/", nil, func() {})
	if err == nil || !strings.Contains(err.Error(), "encode JSON request") {
		t.Fatalf("NewJSONRequest() error = %v, want encoding error", err)
	}
}

// TestGetReturnsBoundedSanitizedHTTPError verifies safe bounded HTTP errors.
//
// Given an HTTP endpoint returning a large error body and sensitive URL data,
// when the client performs a GET request, then the error body is bounded and
// the rendered URL is sanitized.
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

	httpErr, ok := errors.AsType[*HTTPError](err)
	if !ok {
		t.Fatalf("error type = %T", err)
	}

	if !httpErr.BodyTruncated || len(httpErr.Body) != maxErrorBody {
		t.Fatalf("bounded error = %d, truncated %v", len(httpErr.Body), httpErr.BodyTruncated)
	}
}

// TestMaxResponseBody verifies response bodies are capped at MaxResponseBody.
//
// Given a response body exactly at the configured limit and one larger than it,
// when Do reads them, then the bounded body succeeds and the oversized body
// returns a truncated response together with an error.
func TestMaxResponseBody(t *testing.T) {
	t.Parallel()

	const limit = 32

	for _, tt := range []struct {
		name          string
		size          int
		wantTruncated bool
	}{
		{name: "at limit", size: limit},
		{name: "over limit", size: limit + 1, wantTruncated: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(strings.Repeat("x", tt.size)))
			}))
			defer ts.Close()

			client, err := New(Config{BaseURL: ts.URL, MaxResponseBody: limit})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = client.Close() }()

			resp, err := client.Do(context.Background(), Request{Method: http.MethodGet, Path: "/"})
			if tt.wantTruncated {
				if err == nil || !strings.Contains(err.Error(), "exceeds") {
					t.Fatalf("Do() error = %v, want body limit error", err)
				}

				if resp == nil || len(resp.Body) != limit || !resp.BodyTruncated {
					t.Fatalf("Do() response = %+v, want %d truncated bytes", resp, limit)
				}

				if !strings.Contains(err.Error(), http.MethodGet+" "+ts.URL+"/") {
					t.Fatalf("Do() error = %v, want method and sanitized URL", err)
				}

				return
			}

			if err != nil || len(resp.Body) != tt.size || resp.BodyTruncated {
				t.Fatalf("Do() = (%d, %v), want %d untruncated bytes", len(resp.Body), err, tt.size)
			}
		})
	}
}

// TestMaxResponseBodyErrors verifies the configured limit also bounds error bodies.
//
// Given an error response larger than MaxResponseBody, when Do reads it,
// then the HTTPError carries a body truncated at MaxResponseBody.
func TestMaxResponseBodyErrors(t *testing.T) {
	t.Parallel()

	const limit = 16

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, strings.Repeat("x", limit+32), http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, err := New(Config{BaseURL: ts.URL, MaxResponseBody: limit})
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = client.Close() }()

	_, err = client.Do(context.Background(), Request{Method: http.MethodGet, Path: "/"})

	httpErr, ok := errors.AsType[*HTTPError](err)
	if !ok {
		t.Fatalf("error = %v, want *HTTPError", err)
	}

	if len(httpErr.Body) != limit || !httpErr.BodyTruncated {
		t.Fatalf("HTTPError body = %d bytes, truncated %v; want %d bytes", len(httpErr.Body), httpErr.BodyTruncated, limit)
	}
}

// TestDoForNon2xx verifies non-2xx responses surface an HTTP error.
//
// Given an endpoint returning a non-2xx status with a body, when Do sends a request,
// then it returns an *HTTPError carrying the status code and bounded body.
func TestDoForNon2xx(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "teapot", http.StatusTeapot)
	}))
	defer ts.Close()

	client, err := New(Config{BaseURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.Do(context.Background(), Request{Method: http.MethodGet, Path: "/"})

	httpErr, ok := errors.AsType[*HTTPError](err)
	if !ok {
		t.Fatalf("error = %v, want *HTTPError", err)
	}

	if resp != nil {
		t.Fatalf("response = %+v, want nil", resp)
	}

	if httpErr.StatusCode != http.StatusTeapot {
		t.Fatalf("HTTPError status = %d, want %d", httpErr.StatusCode, http.StatusTeapot)
	}

	if !strings.Contains(string(httpErr.Body), "teapot") {
		t.Fatalf("body = %q, want teapot", httpErr.Body)
	}
}
