package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/m-horky/elk/internal/httpclient"
)

const validClaims = `{
	"aud": "https://subscription.rhsm.redhat.com:443/subscription",
	"email": "161958465613-compute@developer.gserviceaccount.com",
	"exp": 1616599823,
	"iat": 1616596223,
	"iss": "https://accounts.google.com",
	"google": {
		"compute_engine": {
			"instance_id": "2589221140676718026",
			"instance_name": "instance-1",
			"project_id": "fair-kingdom-308514",
			"project_number": 9007199254740993,
			"zone": "us-east1-b",
			"license_id": ["5731035067256925298"]
		}
	}
}`

// newTestClient constructs a client pointed at a fake IMDS server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	client, err := NewClient(Config{HTTP: httpclient.Config{BaseURL: server.URL}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	return client
}

// newToken assembles an unsigned JWT with the supplied claims JSON.
func newToken(t *testing.T, claims string) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(claims))
	signature := base64.RawURLEncoding.EncodeToString([]byte("signature"))

	return header + "." + payload + "." + signature
}

// tokenServer returns a fake IMDS that answers identity requests with token.
func tokenServer(t *testing.T, token string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, token)
	}))
	t.Cleanup(server.Close)

	return server
}

// TestIdentitySendsMetadataRequest verifies the identity request contract.
//
// Given a fake Google IMDS that returns a valid identity token
// When Identity is called
// Then the request carries the flavor header and the audience, format and
// licenses query parameters, and the decoded identity is returned.
func TestIdentitySendsMetadataRequest(t *testing.T) {
	t.Parallel()

	token := newToken(t, validClaims)

	var received *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Clone(r.Context())
		_, _ = fmt.Fprint(w, token)
	}))
	t.Cleanup(server.Close)

	identity, err := newTestClient(t, server).Identity(context.Background())
	if err != nil {
		t.Fatalf("Identity() error = %v", err)
	}

	if received.Method != http.MethodGet {
		t.Errorf("request method = %s, want GET", received.Method)
	}

	if received.URL.Path != identityPath {
		t.Errorf("request path = %s, want %s", received.URL.Path, identityPath)
	}

	if flavor := received.Header.Get("Metadata-Flavor"); flavor != "Google" {
		t.Errorf("Metadata-Flavor = %q, want Google", flavor)
	}

	query := received.URL.Query()
	if got := query.Get("audience"); got != audience {
		t.Errorf("audience query = %q, want %q", got, audience)
	}

	if got := query.Get("format"); got != "full" {
		t.Errorf("format query = %q, want full", got)
	}

	if got := query.Get("licenses"); got != "TRUE" {
		t.Errorf("licenses query = %q, want TRUE", got)
	}

	if identity == nil {
		t.Fatal("Identity() = nil, want identity")
	}
}

// TestIdentityPreservesNumericClaims verifies numeric claims are not rounded.
//
// Given an identity token with a project_number larger than 2^53 and a numeric
// instance_id
// When Identity is called
// Then the decoded claims contain every original digit.
func TestIdentityPreservesNumericClaims(t *testing.T) {
	t.Parallel()

	claims := `{
		"exp": 1616599823,
		"iat": 1616596223,
		"google": {
			"compute_engine": {
				"instance_id": 2589221140676718026,
				"instance_name": "instance-1",
				"project_id": "fair-kingdom-308514",
				"project_number": 9007199254740993,
				"zone": "us-east1-b"
			}
		}
	}`

	server := tokenServer(t, newToken(t, claims))

	identity, err := newTestClient(t, server).Identity(context.Background())
	if err != nil {
		t.Fatalf("Identity() error = %v", err)
	}

	engine := identity.Claims.Google.ComputeEngine
	if engine.InstanceID != "2589221140676718026" {
		t.Errorf("InstanceID = %q, want 2589221140676718026", engine.InstanceID)
	}

	if engine.ProjectNumber != "9007199254740993" {
		t.Errorf("ProjectNumber = %q, want 9007199254740993", engine.ProjectNumber)
	}
}

// TestIdentityReturnsIdentity verifies the populated identity contract.
//
// Given a fake Google IMDS that returns a valid identity token
// When Identity is called
// Then a non-nil identity is returned together with a nil error.
func TestIdentityReturnsIdentity(t *testing.T) {
	t.Parallel()

	server := tokenServer(t, newToken(t, validClaims))

	identity, err := newTestClient(t, server).Identity(context.Background())
	if err != nil {
		t.Fatalf("Identity() error = %v", err)
	}

	if identity == nil {
		t.Fatal("Identity() = nil, want identity")
	}

	if identity.Claims.Audience != audience {
		t.Errorf("Claims.Audience = %q, want %q", identity.Claims.Audience, audience)
	}

	if identity.Claims.Google.ComputeEngine.ProjectID != "fair-kingdom-308514" {
		t.Errorf("ProjectID = %q, want fair-kingdom-308514", identity.Claims.Google.ComputeEngine.ProjectID)
	}

	if strings.Contains(identity.Token, "=") {
		t.Errorf("Identity.Token = %q, want unpadded base64url segments", identity.Token)
	}

	wantExpiry := time.Unix(1616599823, 0).UTC()
	if !identity.Claims.ExpiresAt.Equal(wantExpiry) {
		t.Errorf("Claims.ExpiresAt = %s, want %s", identity.Claims.ExpiresAt, wantExpiry)
	}

	wantIssued := time.Unix(1616596223, 0).UTC()
	if !identity.Claims.IssuedAt.Equal(wantIssued) {
		t.Errorf("Claims.IssuedAt = %s, want %s", identity.Claims.IssuedAt, wantIssued)
	}

	wantLicenses := []string{"5731035067256925298"}
	if len(identity.Claims.Google.ComputeEngine.LicenseIDs) != len(wantLicenses) ||
		identity.Claims.Google.ComputeEngine.LicenseIDs[0] != wantLicenses[0] {
		t.Errorf("LicenseIDs = %v, want %v", identity.Claims.Google.ComputeEngine.LicenseIDs, wantLicenses)
	}
}

// TestIdentityRejectsInvalidResponses verifies failure responses return errors.
//
// Given a fake Google IMDS that returns a non-2xx status or an invalid token
// When Identity is called
// Then a nil identity and a descriptive error are returned.
func TestIdentityRejectsInvalidResponses(t *testing.T) {
	t.Parallel()

	invalidClaims := map[string]string{
		"fewer than three segments": "header.claims",
		"undecodable claims":        "header.@@@.signature",
		"claims not JSON":           newToken(t, "not-json"),
		"missing instance_id":       newToken(t, `{"google":{"compute_engine":{"project_id":"p"}}}`),
		"missing project_id":        newToken(t, `{"google":{"compute_engine":{"instance_id":"i"}}}`),
		"non-numeric exp": newToken(t, `{
			"exp": "soon",
			"google": {"compute_engine": {"instance_id": "i", "project_id": "p"}}
		}`),
	}

	for name, token := range invalidClaims {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := tokenServer(t, token)

			identity, err := newTestClient(t, server).Identity(context.Background())
			if err == nil {
				t.Fatal("Identity() error = nil, want error")
			}

			if identity != nil {
				t.Fatalf("Identity() = %+v, want nil", identity)
			}
		})
	}

	t.Run("non-2xx status", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "flavor header missing", http.StatusForbidden)
		}))
		t.Cleanup(server.Close)

		identity, err := newTestClient(t, server).Identity(context.Background())
		if err == nil {
			t.Fatal("Identity() error = nil, want error")
		}

		if identity != nil {
			t.Fatalf("Identity() = %+v, want nil", identity)
		}

		if !strings.Contains(err.Error(), "403") {
			t.Errorf("Identity() error = %v, want status 403", err)
		}
	})
}

// TestIdentityRejectsTransportFailure verifies transport failures are errors.
//
// Given a fake Google IMDS server that has been shut down
// When Identity is called
// Then a nil identity and a wrapped transport error are returned.
func TestIdentityRejectsTransportFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := newTestClient(t, server)
	server.Close()

	identity, err := client.Identity(context.Background())
	if err == nil {
		t.Fatal("Identity() error = nil, want error")
	}

	if identity != nil {
		t.Fatalf("Identity() = %+v, want nil", identity)
	}
}

// TestIdentityRejectsOversizedResponse verifies the response body cap.
//
// Given a Google IMDS that serves an identity token larger than 128 KiB,
// When Identity is called,
// Then collection fails with a body-limit error.
func TestIdentityRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, strings.Repeat("x", maxResponseBody+1))
	}))
	defer server.Close()

	identity, err := newTestClient(t, server).Identity(context.Background())
	if err == nil {
		t.Fatal("Identity() error = nil, want error")
	}

	if identity != nil {
		t.Fatalf("Identity() = %+v, want nil", identity)
	}

	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("Identity() error = %q, want a body-limit error", err)
	}
}

// TestClaimsRoundTrip verifies claims survive JSON encoding and decoding.
//
// Given decoded claims with large numeric identifiers and Unix timestamps
// When the claims are marshaled and unmarshaled through encoding/json
// Then the values are unchanged and numeric claims keep every digit.
func TestClaimsRoundTrip(t *testing.T) {
	t.Parallel()

	original := Claims{
		Audience:  audience,
		Email:     "compute@example.test",
		ExpiresAt: time.Unix(1616599823, 0).UTC(),
		IssuedAt:  time.Unix(1616596223, 0).UTC(),
		Issuer:    "https://accounts.google.com",
		Google: GoogleClaims{ComputeEngine: ComputeEngine{
			InstanceID:    "2589221140676718026",
			InstanceName:  "instance-1",
			ProjectID:     "fair-kingdom-308514",
			ProjectNumber: "9007199254740993",
			Zone:          "us-east1-b",
			LicenseIDs:    []string{"5731035067256925298"},
		}},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if !strings.Contains(string(encoded), `"exp":1616599823`) {
		t.Errorf("encoded claims = %s, want numeric exp claim", encoded)
	}

	var decoded Claims
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if diff := cmp.Diff(original, decoded); diff != "" {
		t.Errorf("claims round-trip mismatch (-want +got):\n%s", diff)
	}
}

// TestNewClientRejectsInvalidBaseURL verifies construction validates the address.
//
// Given a configuration with an invalid IMDS base URL
// When NewClient is called
// Then construction fails with a descriptive error.
func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Config{HTTP: httpclient.Config{BaseURL: "://invalid"}})
	if err == nil {
		t.Fatal("NewClient() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "invalid HTTP base URL") {
		t.Errorf("NewClient() error = %v, want invalid base URL error", err)
	}
}
