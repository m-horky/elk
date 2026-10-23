package azure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/m-horky/elk/internal/cloud/imds"
	"github.com/m-horky/elk/internal/httpclient"
)

const (
	validInstanceJSON = `{
		"compute": {
			"vmId": "2f4a6b1c-3d5e-4f7a-8b9c-0d1e2f3a4b5c",
			"name": "elk-test",
			"offer": "RHEL",
			"sku": "9-lvm-gen2",
			"location": "eastus",
			"resourceGroupName": "elk-rg",
			"subscriptionId": "6a7b8c9d-0e1f-4a2b-8c3d-4e5f6a7b8c9d",
			"extendedLocation": {"name": "edge-site", "type": "EdgeZone"},
			"unknownField": "ignored"
		},
		"network": {"interface": []}
	}`
	validAttestedJSON = `{
		"encoding": "pkcs7",
		"signature": "c2lnbmF0dXJl",
		"pkcs7": "cGtjczc="
	}`
)

// newTestClient constructs an Azure IMDS client pointed at the supplied server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	client, err := NewClient(Config{HTTP: httpclient.Config{BaseURL: server.URL}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	t.Cleanup(func() { _ = client.Close() })

	return client
}

// instanceAndAttestedHandler serves the instance and attested endpoints,
// passing each request to the supplied callback.
func instanceAndAttestedHandler(
	t *testing.T, instance, attested string, onRequest func(*http.Request),
) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, req *http.Request) {
		if onRequest != nil {
			onRequest(req)
		}

		w.Header().Set("Content-Type", "application/json")

		switch req.URL.Path {
		case instancePath:
			_, _ = w.Write([]byte(instance))
		case attestedPath:
			_, _ = w.Write([]byte(attested))
		default:
			t.Errorf("unexpected request path %q", req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// recordedRequest is a request captured by a fake IMDS.
type recordedRequest struct {
	method string
	path   string
	header http.Header
	query  url.Values
}

// requestRecorder records requests served by a fake IMDS.
type requestRecorder struct {
	mu       sync.Mutex
	requests []recordedRequest
}

// record stores a copy of the request.
func (r *requestRecorder) record(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests = append(r.requests, recordedRequest{
		method: req.Method,
		path:   req.URL.Path,
		header: req.Header.Clone(),
		query:  req.URL.Query(),
	})
}

// snapshot returns the recorded requests.
func (r *requestRecorder) snapshot() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]recordedRequest(nil), r.requests...)
}

// TestCollect verifies both IMDS requests are header-gated and version-pinned.
//
// Given a fake IMDS, when Collect is called, then both requests are GETs carrying
// the Metadata: true header and the pinned api-version, instance before attested.
func TestCollect(t *testing.T) {
	t.Parallel()

	recorder := new(requestRecorder)

	server := httptest.NewServer(instanceAndAttestedHandler(t, validInstanceJSON, validAttestedJSON, recorder.record))
	defer server.Close()

	if _, err := newTestClient(t, server).Collect(context.Background()); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	requests := recorder.snapshot()
	if len(requests) != 2 {
		t.Fatalf("Metadata() made %d requests, want 2: %v", len(requests), requests)
	}

	for _, request := range requests {
		if request.method != http.MethodGet {
			t.Errorf("request %q method = %q, want %q", request.path, request.method, http.MethodGet)
		}

		if got := request.header.Get("Metadata"); got != "true" {
			t.Errorf("request %q Metadata header = %q, want %q", request.path, got, "true")
		}

		if got := request.query.Get("api-version"); got != apiVersion {
			t.Errorf("request %q api-version = %q, want %q", request.path, got, apiVersion)
		}
	}

	if requests[0].path != instancePath || requests[1].path != attestedPath {
		t.Errorf("Metadata() request paths = %q, %q; want %q, %q",
			requests[0].path, requests[1].path, instancePath, attestedPath)
	}
}

// TestNon2xx verifies a rejected instance request fails collection.
//
// Given a fake IMDS answering the instance request with 500, when Collect is called,
// then it returns (nil, err) and never requests the attested document.
func TestNon2xx(t *testing.T) {
	t.Parallel()

	recorder := new(requestRecorder)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		recorder.record(req)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	metadata, err := newTestClient(t, server).Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() error = nil, want error for non-2xx instance response")
	}

	if metadata != nil {
		t.Errorf("Collect() = %+v, want nil metadata", metadata)
	}

	if requests := recorder.snapshot(); len(requests) != 1 || requests[0].path != instancePath {
		t.Errorf("Metadata() requests = %v, want only %q", requests, instancePath)
	}
}

// TestNon2xxAttestation verifies attested failure after a valid instance document.
//
// Given a valid instance response and an attested request answered with 404, when
// Metadata is called, then it returns (nil, err) and does not return partial metadata.
func TestNon2xxAttestation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == attestedPath {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		_, _ = w.Write([]byte(validInstanceJSON))
	}))
	defer server.Close()

	metadata, err := newTestClient(t, server).Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() error = nil, want error for non-2xx attested response")
	}

	if metadata != nil {
		t.Errorf("Collect() = %+v, want nil metadata", metadata)
	}
}

// TestCollectRejectsOversizedResponse verifies the response body cap.
//
// Given a fake IMDS that serves an instance document larger than 512 KiB,
// when Collect is called,
// then collection fails with a body-limit error.
func TestCollectRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", imds.MaxResponseBody+1)))
	}))
	defer server.Close()

	metadata, err := newTestClient(t, server).Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() error = nil, want error for oversized response")
	}

	if metadata != nil {
		t.Errorf("Collect() = %+v, want nil metadata", metadata)
	}

	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("Collect() error = %q, want a body-limit error", err)
	}
}
