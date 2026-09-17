package candlepin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/m-horky/elk/internal/httpclient"
	"github.com/m-horky/elk/pkg/facts"
)

// TestBasicClientListsOwnersAndCreatesConsumer verifies the BASIC-authenticated consumer flow.
//
// Given a Candlepin server that accepts valid credentials
// When owners are listed and a consumer is created
// Then both operations succeed and return the decoded API objects.
func TestBasicClientListsOwnersAndCreatesConsumer(t *testing.T) { //nolint:funlen
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "user" || password != "password" {
			t.Fatalf("basic auth = %q, %q, %v", username, password, ok)
		}

		if r.URL.Path == "/users/alice/owners" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"key":"org-1","displayName":"Organization 1"}]`))

			return
		}

		if r.URL.Path == "/consumers" {
			if r.Method != http.MethodPost || r.URL.Query().Get("identity_cert_creation") != "true" {
				t.Errorf("consumer request = %s %s", r.Method, r.URL.String())
			}

			var request CreateConsumerRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}

			if request.Facts["system.certificate_version"] != "3.2" {
				t.Errorf("system certificate version = %q, want %q", request.Facts["system.certificate_version"], "3.2")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"uuid":"consumer-1","idCert":{"cert":"CERT","key":"KEY"}}`))

			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	//nolint:lll
	client, err := NewBasicClient(Config{HTTP: httpclient.Config{BaseURL: server.URL, TLSVerify: true}}, BasicCredentials{Username: "user", Password: "password"}) //nolint:lll
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close() //nolint:errcheck

	owners, err := client.ListUserOwners(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}

	if len(owners) != 1 || owners[0].Key != "org-1" {
		t.Fatalf("owners = %+v", owners)
	}
	//nolint:lll
	consumer, err := client.CreateConsumer(context.Background(), CreateConsumerRequest{
		Name:  "test",
		Type:  ConsumerType{Label: "system"},
		Facts: NewFactsDTO(&facts.Facts{SystemCertificateVersion: new("3.2")}),
	}, CreateConsumerOptions{Owner: "org-1"})
	if err != nil {
		t.Fatal(err)
	}

	if consumer.UUID != "consumer-1" || consumer.IDCert == nil {
		t.Fatalf("consumer = %+v", consumer)
	}
}

// TestListUserOwnersRejectsEmptyUsername verifies that an empty username is rejected locally.
//
// Given a Candlepin client
// When owners are requested for an empty username
// Then the client returns a validation error without making a request.
func TestListUserOwnersRejectsEmptyUsername(t *testing.T) {
	client, err := NewProbeClient(Config{HTTP: httpclient.Config{BaseURL: "https://example.test", TLSVerify: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close() //nolint:errcheck

	_, err = client.ListUserOwners(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "username") {
		t.Fatalf("error = %v", err)
	}
}
