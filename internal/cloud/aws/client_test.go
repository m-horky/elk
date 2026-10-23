package aws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/m-horky/elk/internal/cloud/imds"
	"github.com/m-horky/elk/internal/httpclient"
)

// documentJSON is a representative instance identity document.
const documentJSON = `{
	"accountId": "123456789012",
	"architecture": "x86_64",
	"availabilityZone": "us-east-1a",
	"billingProducts": null,
	"imageId": "ami-123",
	"instanceId": "i-abc",
	"instanceType": "m5.large",
	"marketplaceProductCodes": null,
	"pendingTime": "2024-01-02T03:04:05Z",
	"privateIp": "10.0.0.5",
	"region": "us-east-1",
	"version": "2017-09-30"
}`

// newTestClient constructs a client pointed at the test server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	client, err := NewClient(Config{HTTP: httpclient.Config{BaseURL: server.URL}})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = client.Close() })

	return client
}

// TestCollect verifies the full IMDSv2 flow.
//
// Given an IMDS that issues a token and serves a valid document and signature,
// when the instance identity is requested, then the token request carries the TTL,
// the document request carries the token, and the parsed document and the
// whitespace-trimmed signature are returned.
func TestCollect(t *testing.T) {
	t.Parallel()

	var tokenRequests, documentRequests, signatureRequests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case apiPath:
			tokenRequests.Add(1)

			if r.Method != http.MethodPut {
				t.Errorf("token method = %s, want PUT", r.Method)
			}

			if got := r.Header.Get(sessionTTLHeader); got != "3600" {
				t.Errorf("token TTL header = %q, want %q", got, "3600")
			}

			_, _ = w.Write([]byte("TOKEN-ONE"))
		case documentPath:
			documentRequests.Add(1)

			if got := r.Header.Get(sessionHeader); got != "TOKEN-ONE" {
				t.Errorf("document token header = %q, want %q", got, "TOKEN-ONE")
			}

			_, _ = w.Write([]byte(documentJSON))
		case signaturePath:
			signatureRequests.Add(1)

			if got := r.Header.Get(sessionHeader); got != "TOKEN-ONE" {
				t.Errorf("signature token header = %q, want %q", got, "TOKEN-ONE")
			}

			_, _ = w.Write([]byte("U0lHTkFUVVJF\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	identity, err := newTestClient(t, server).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if identity.Signature != "U0lHTkFUVVJF" {
		t.Errorf("Signature = %q, want %q", identity.Signature, "U0lHTkFUVVJF")
	}

	assertRequestCounts(t, tokenRequests.Load(), documentRequests.Load(), signatureRequests.Load())
}

// assertRequestCounts verifies the number of requests observed per IMDS endpoint.
func assertRequestCounts(t *testing.T, tokens, documents, signatures int32) {
	t.Helper()

	if tokens != 1 || documents != 1 || signatures != 1 {
		t.Errorf("requests = token %d, document %d, signature %d", tokens, documents, signatures)
	}
}

// TestTokenIsCached verifies token reuse across calls.
//
// Given an IMDS that issues one token, when the instance identity is requested twice,
// then the token endpoint is hit once and the document endpoint twice.
func TestTokenIsCached(t *testing.T) {
	t.Parallel()

	var tokenRequests, documentRequests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case apiPath:
			tokenRequests.Add(1)

			_, _ = w.Write([]byte("TOKEN"))
		case documentPath:
			documentRequests.Add(1)

			_, _ = w.Write([]byte(documentJSON))
		case signaturePath:
			_, _ = w.Write([]byte("U0lHTkFUVVJF"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)

	for range 2 {
		if _, err := client.Collect(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if tokenRequests.Load() != 1 {
		t.Errorf("token requests = %d, want 1", tokenRequests.Load())
	}

	if documentRequests.Load() != 2 {
		t.Errorf("document requests = %d, want 2", documentRequests.Load())
	}
}

// TestCollectRejectsOversizedResponse verifies the response body cap.
//
// Given an IMDS that serves a document larger than 512 KiB,
// when the instance identity is requested,
// then collection fails with a body-limit error.
func TestCollectRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == apiPath {
			_, _ = w.Write([]byte("TOKEN"))

			return
		}

		_, _ = w.Write([]byte(strings.Repeat("x", imds.MaxResponseBody+1)))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).Collect(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error = %q, want a body-limit error", err)
	}
}
