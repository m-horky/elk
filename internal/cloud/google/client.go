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
	"time"

	"github.com/m-horky/elk/internal/httpclient"
)

const (
	// audience is the token for verifying the instance identity.
	audience = "https://subscription.rhsm.redhat.com:443/subscription"
	// identityPath is the Compute Engine service account identity endpoint.
	identityPath = "/computeMetadata/v1/instance/service-accounts/default/identity"
	// defaultConnectTimeout bounds establishing a connection to the IMDS.
	defaultConnectTimeout = time.Second
	// defaultRequestTimeout bounds a complete request to the IMDS.
	defaultRequestTimeout = time.Second
	// maxResponseBody caps every IMDS response; identity tokens are tiny.
	maxResponseBody = 512 << 10
)

type (
	// Config configures the Google IMDS client.
	Config struct{ HTTP httpclient.Config }
)

// Client is a Google Compute Engine instance metadata client.
type Client struct{ http *httpclient.Client }

// NewClient constructs a Google IMDS client from the supplied configuration.
// Responses are capped and configured proxies are ignored so
// metadata traffic stays on the link-local path.
func NewClient(cfg Config) (*Client, error) {
	slog.Debug("constructing Google IMDS client")

	if cfg.HTTP.ConnectTimeout <= 0 {
		cfg.HTTP.ConnectTimeout = defaultConnectTimeout
	}

	if cfg.HTTP.RequestTimeout <= 0 {
		cfg.HTTP.RequestTimeout = defaultRequestTimeout
	}

	if cfg.HTTP.MaxResponseBody <= 0 || cfg.HTTP.MaxResponseBody > maxResponseBody {
		cfg.HTTP.MaxResponseBody = maxResponseBody
	}

	// Ignore proxy, if set; IMDS needs to be accessed locally.
	cfg.HTTP.Proxy.URI = ""

	client, err := httpclient.New(cfg.HTTP)
	if err != nil {
		return nil, fmt.Errorf("construct Google IMDS HTTP client: %w", err)
	}

	return &Client{http: client}, nil
}

// Identity returns the Google instance identity. It returns an error when the
// IMDS does not answer with a valid Google identity token.
func (c *Client) Identity(ctx context.Context) (*Identity, error) {
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

	identity, err := decodeIdentity(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode Google instance identity: %w", err)
	}

	return identity, nil
}

// Close releases the resources held by the Google IMDS client.
func (c *Client) Close() error {
	if err := c.http.Close(); err != nil {
		return fmt.Errorf("close Google IMDS HTTP client: %w", err)
	}

	return nil
}

// Identity is the Google instance identity token and its decoded claims.
type Identity struct {
	// Token is the raw JWT returned by the identity endpoint.
	Token string
	// Claims are the decoded claims of the token.
	Claims Claims
}

// decodeIdentity decodes the JWT from the identity endpoint into an Identity.
func decodeIdentity(body []byte) (*Identity, error) {
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

	return &Identity{Token: token, Claims: claims}, nil
}
