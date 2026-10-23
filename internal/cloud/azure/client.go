package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/m-horky/elk/internal/cloud/imds"
	"github.com/m-horky/elk/internal/httpclient"
)

const (
	// apiVersion is the pinned Azure IMDS API version.
	apiVersion = "2023-07-01"
	// instancePath is the Azure IMDS instance document endpoint.
	instancePath = "/metadata/instance"
	// attestedPath is the Azure IMDS attested document endpoint.
	attestedPath = "/metadata/attested/document"
)

// Config configures the Azure IMDS client.
type Config struct {
	// HTTP is the shared HTTP client configuration, including the IMDS base URL.
	HTTP httpclient.Config
}

// Client is a client for the Microsoft Azure instance metadata service.
type Client struct{ http *httpclient.Client }

// Metadata is the Azure instance metadata with its attested document.
type Metadata struct {
	// Instance is the parsed /metadata/instance document.
	Instance Instance
	// Attested is the parsed /metadata/attested/document response.
	Attested Attested
}

// Instance mirrors the compute object of the Azure IMDS instance document.
type Instance struct {
	Compute Compute `json:"compute"`
}

// Compute contains the virtual machine identity fields of the instance document.
type Compute struct {
	VMID              string           `json:"vmId"`
	Name              string           `json:"name"`
	Offer             string           `json:"offer"`
	SKU               string           `json:"sku"`
	Location          string           `json:"location"`
	ResourceGroupName string           `json:"resourceGroupName"`
	SubscriptionID    string           `json:"subscriptionId"`
	ExtendedLocation  ExtendedLocation `json:"extendedLocation"`
}

// ExtendedLocation contains the optional extended location of the instance.
type ExtendedLocation struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Attested mirrors the Azure IMDS attested document.
type Attested struct {
	Encoding  string `json:"encoding"`
	Signature string `json:"signature"`
	PKCS7     string `json:"pkcs7"`
}

// NewClient constructs an Azure IMDS client. The HTTP client is configured
// with the shared IMDS defaults: 30 second connect and request timeouts, a
// 512 KiB response cap, and no proxy, so metadata traffic stays on the
// link-local path.
func NewClient(cfg Config) (*Client, error) {
	slog.Debug("constructing Azure IMDS client")

	client, err := imds.NewClient(cfg.HTTP)
	if err != nil {
		return nil, fmt.Errorf("construct Azure IMDS client: %w", err)
	}

	return &Client{http: client}, nil
}

// Close releases the resources held by the Azure IMDS client.
func (c *Client) Close() error {
	if err := c.http.Close(); err != nil {
		return fmt.Errorf("close Azure IMDS client: %w", err)
	}

	return nil
}

// Collect returns the Azure instance metadata with its attested document.
// It returns an error when the IMDS does not answer with valid Azure
// instance metadata.
func (c *Client) Collect(ctx context.Context) (*Metadata, error) {
	slog.Debug("collecting Azure instance metadata")

	instance, err := c.instance(ctx)
	if err != nil {
		return nil, err
	}

	if instance.Compute.VMID == "" {
		return nil, fmt.Errorf("azure instance metadata is missing compute.vmId")
	}

	attested, err := c.attested(ctx)
	if err != nil {
		return nil, err
	}

	return &Metadata{Instance: *instance, Attested: *attested}, nil
}

// instance requests and decodes the /metadata/instance document.
func (c *Client) instance(ctx context.Context) (*Instance, error) {
	return fetchDocument[Instance](ctx, c, instancePath, "Azure instance metadata")
}

// attested requests and decodes the /metadata/attested/document response.
func (c *Client) attested(ctx context.Context) (*Attested, error) {
	return fetchDocument[Attested](ctx, c, attestedPath, "Azure attested document")
}

// fetchDocument requests and decodes the IMDS document at path into T.
func fetchDocument[T any](ctx context.Context, c *Client, path, name string) (*T, error) {
	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", name, err)
	}

	var document T
	if err := json.Unmarshal(body, &document); err != nil {
		slog.Debug(name+" decode failed", "err", err)

		return nil, fmt.Errorf("decode %s: %w", name, err)
	}

	return &document, nil
}

// get performs a header-gated, version-pinned GET against an IMDS endpoint.
func (c *Client) get(ctx context.Context, p string) ([]byte, error) {
	query := url.Values{"api-version": []string{apiVersion}}
	headers := http.Header{"Metadata": []string{"true"}}

	resp, err := c.http.Do(ctx, httpclient.Request{
		Method:  http.MethodGet,
		Path:    p,
		Query:   query,
		Headers: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", p, err)
	}

	return resp.Body, nil
}
