package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/m-horky/elk/internal/cloud/imds"
	"github.com/m-horky/elk/internal/httpclient"
)

const (
	// tokenTTLSeconds is the lifetime requested for IMDSv2 session tokens.
	tokenTTLSeconds = 3600
	// apiPath is the IMDSv2 session token endpoint.
	apiPath = "/latest/api/token"
	// documentPath is the EC2 instance identity document endpoint.
	documentPath = "/latest/dynamic/instance-identity/document"
	// signaturePath is the EC2 instance identity signature endpoint.
	signaturePath = "/latest/dynamic/instance-identity/rsa2048"
	// sessionHeader carries an IMDSv2 session token on metadata requests.
	sessionHeader = "X-aws-ec2-metadata-token"
	// sessionTTLHeader requests the lifetime of a new IMDSv2 session token.
	sessionTTLHeader = "X-aws-ec2-metadata-token-ttl-seconds"
)

// Config configures the AWS IMDS client.
type Config struct {
	// HTTP is the shared HTTP client configuration, including the IMDS base URL.
	HTTP httpclient.Config
}

// Client is an AWS IMDSv2 client.
type Client struct {
	http  *httpclient.Client
	mu    sync.Mutex
	token string
}

// Metadata is the EC2 instance identity document and its signature.
type Metadata struct {
	// Document is the parsed instance identity document.
	Document IdentityDocument
	// Signature is the raw base64 rsa2048 signature of the document.
	Signature string
}

// IdentityDocument mirrors the EC2 instance identity document.
type IdentityDocument struct {
	AccountID               string    `json:"accountId"`
	Architecture            string    `json:"architecture"`
	AvailabilityZone        string    `json:"availabilityZone"`
	BillingProducts         []string  `json:"billingProducts"`
	DevpayProductCodes      []string  `json:"devpayProductCodes"`
	ImageID                 string    `json:"imageId"`
	InstanceID              string    `json:"instanceId"`
	InstanceType            string    `json:"instanceType"`
	KernelID                string    `json:"kernelId"`
	MarketplaceProductCodes []string  `json:"marketplaceProductCodes"`
	PendingTime             time.Time `json:"pendingTime"`
	PrivateIP               string    `json:"privateIp"`
	RamdiskID               string    `json:"ramdiskId"`
	Region                  string    `json:"region"`
	Version                 string    `json:"version"`
}

// NewClient constructs an AWS IMDS client. The HTTP client is configured
// with the shared IMDS defaults: 30 second connect and request timeouts, a
// 512 KiB response cap, and no proxy, so metadata traffic stays on the
// link-local path.
func NewClient(cfg Config) (*Client, error) {
	slog.Debug("constructing AWS IMDS client")

	client, err := imds.NewClient(cfg.HTTP)
	if err != nil {
		return nil, fmt.Errorf("construct AWS IMDS client: %w", err)
	}

	return &Client{http: client}, nil
}

// Close releases the resources held by the AWS IMDS client.
func (c *Client) Close() error {
	if err := c.http.Close(); err != nil {
		return fmt.Errorf("close AWS IMDS client: %w", err)
	}

	return nil
}

// Collect returns the EC2 instance identity with its signature. It returns
// an error when the IMDS does not answer with a valid AWS instance identity
// document.
func (c *Client) Collect(ctx context.Context) (*Metadata, error) {
	document, token, err := c.document(ctx)
	if err != nil {
		return nil, err
	}

	signature, err := c.signature(ctx, token)
	if err != nil {
		return nil, err
	}

	return &Metadata{Document: *document, Signature: signature}, nil
}

// TODO: refactor the token refresh-retry flow so that the signature fetch
//  shares it and the token-rejection path is covered by tests.

// document fetches and validates the instance identity document, refreshing
// the session token once when the IMDS rejects the held one.
func (c *Client) document(ctx context.Context) (*IdentityDocument, string, error) {
	token, err := c.sessionToken(ctx, false)
	if err != nil {
		return nil, "", err
	}

	document, err := c.fetchDocument(ctx, token)
	if err == nil {
		return document, token, nil
	}

	httpErr, ok := errors.AsType[*httpclient.HTTPError](err)
	if !ok || !isAuthStatus(httpErr.StatusCode) {
		return nil, "", err
	}

	slog.Debug("AWS IMDS rejected instance identity token, refreshing", "status", httpErr.StatusCode)

	c.discardToken(token)

	token, err = c.sessionToken(ctx, true)
	if err != nil {
		return nil, "", err
	}

	document, err = c.fetchDocument(ctx, token)
	if err != nil {
		return nil, "", fmt.Errorf("request AWS instance identity document after token refresh: %w", err)
	}

	return document, token, nil
}

// fetchDocument requests and decodes the instance identity document with token.
func (c *Client) fetchDocument(ctx context.Context, token string) (*IdentityDocument, error) {
	resp, err := c.http.Do(ctx, httpclient.Request{
		Method:  http.MethodGet,
		Path:    documentPath,
		Headers: sessionHeaders(token),
	})
	if err != nil {
		return nil, fmt.Errorf("request AWS instance identity document: %w", err)
	}

	return decodeDocument(resp.Body)
}

// signature fetches the rsa2048 signature of the instance identity document.
func (c *Client) signature(ctx context.Context, token string) (string, error) {
	resp, err := c.http.Do(ctx, httpclient.Request{
		Method:  http.MethodGet,
		Path:    signaturePath,
		Headers: sessionHeaders(token),
	})
	if err != nil {
		return "", fmt.Errorf("request AWS instance identity signature: %w", err)
	}

	return strings.TrimSpace(string(resp.Body)), nil
}

// sessionToken returns the held IMDSv2 token or fetches a new one. A forced
// refresh fetches a new token even when one is already held. The mutex is
// held across the fetch so concurrent callers reuse a single token.
func (c *Client) sessionToken(ctx context.Context, force bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && !force {
		return c.token, nil
	}

	resp, err := c.http.Do(ctx, httpclient.Request{
		Method: http.MethodPut,
		Path:   apiPath,
		Headers: http.Header{
			sessionTTLHeader: []string{strconv.Itoa(tokenTTLSeconds)},
		},
	})
	if err != nil {
		return "", fmt.Errorf("request AWS IMDSv2 session token: %w", err)
	}

	token := string(resp.Body)
	if token == "" {
		return "", errors.New("request AWS IMDSv2 session token: empty token")
	}

	c.token = token

	return token, nil
}

// discardToken forgets the held token when it is still the one that was rejected.
func (c *Client) discardToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token == token {
		c.token = ""
	}
}

// sessionHeaders builds the request headers carrying an IMDSv2 session token.
func sessionHeaders(token string) http.Header {
	return http.Header{sessionHeader: []string{token}}
}

// isAuthStatus reports whether a status means the IMDS rejected the session token.
func isAuthStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

// decodeDocument parses and validates an instance identity document.
func decodeDocument(body []byte) (*IdentityDocument, error) {
	document := new(IdentityDocument)
	if err := json.Unmarshal(body, document); err != nil {
		return nil, fmt.Errorf("decode AWS instance identity document: %w", err)
	}

	if document.InstanceID == "" {
		return nil, errors.New("AWS instance identity document has no instanceId")
	}

	if document.Region == "" {
		return nil, errors.New("AWS instance identity document has no region")
	}

	return document, nil
}
