package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/m-horky/elk/internal/cloud/imds"
	"github.com/m-horky/elk/internal/httpclient"
)

const (
	// audience is the token for verifying the instance identity.
	audience = "https://subscription.rhsm.redhat.com:443/subscription"
	// identityPath is the Compute Engine service account identity endpoint.
	identityPath = "/computeMetadata/v1/instance/service-accounts/default/identity"
)

// Config configures the Google IMDS client.
type Config struct {
	// HTTP is the shared HTTP client configuration, including the IMDS base URL.
	HTTP httpclient.Config
}

// Client is a Google Compute Engine instance metadata client.
type Client struct{ http *httpclient.Client }

// Metadata is the Google instance identity token and its decoded claims.
type Metadata struct {
	// Token is the raw JWT returned by the identity endpoint.
	Token string
	// Claims are the decoded claims of the token.
	Claims Claims
}

// NewClient constructs a Google IMDS client from the supplied configuration.
// The HTTP client is configured with the shared IMDS defaults: 30 second
// connect and request timeouts, a 512 KiB response cap, and no proxy, so
// metadata traffic stays on the link-local path.
func NewClient(cfg Config) (*Client, error) {
	slog.Debug("constructing Google IMDS client")

	client, err := imds.NewClient(cfg.HTTP)
	if err != nil {
		return nil, fmt.Errorf("construct Google IMDS client: %w", err)
	}

	return &Client{http: client}, nil
}

// Collect returns the Google instance identity. It returns an error when
// the IMDS does not answer with a valid Google identity token.
func (c *Client) Collect(ctx context.Context) (*Metadata, error) {
	query := url.Values{
		"audience": {audience},
		"format":   {"full"},
		"licenses": {"TRUE"},
	}
	headers := http.Header{"Metadata-Flavor": {"Google"}}

	resp, err := c.http.Do(ctx, httpclient.Request{
		Method:  http.MethodGet,
		Path:    identityPath,
		Query:   query,
		Headers: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("request Google instance identity: %w", err)
	}

	metadata, err := decodeMetadata(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode Google instance metadata: %w", err)
	}

	return metadata, nil
}

// Close releases the resources held by the Google IMDS client.
func (c *Client) Close() error {
	if err := c.http.Close(); err != nil {
		return fmt.Errorf("close Google IMDS HTTP client: %w", err)
	}

	return nil
}

// decodeMetadata decodes the JWT from the identity endpoint into Metadata.
func decodeMetadata(body []byte) (*Metadata, error) {
	token := strings.TrimSpace(string(body))

	segments := strings.Split(token, ".")
	if len(segments) < 3 {
		return nil, fmt.Errorf("identity token has %d segments, want at least 3", len(segments))
	}

	encodedClaims, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return nil, fmt.Errorf("decode identity token claims: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(encodedClaims, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal identity token claims: %w", err)
	}

	if err := validateComputeEngine(claims.Google.ComputeEngine); err != nil {
		return nil, err
	}

	return &Metadata{Token: token, Claims: claims}, nil
}
