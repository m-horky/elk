package candlepin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/m-horky/elk/internal/httpclient"
)

// CreateConsumerRequest contains the JSON body used to register a consumer.
type CreateConsumerRequest struct {
	// Name is the requested display name for the consumer.
	Name string `json:"name,omitempty"`
	// Type identifies the kind of consumer being registered.
	Type ConsumerType `json:"type"`
	// Facts contains facts to associate with the new consumer.
	Facts map[string]string `json:"facts,omitempty"`
}

// CreateConsumerOptions contains query parameters for consumer creation.
type CreateConsumerOptions struct {
	// Owner selects the organization that owns the new consumer.
	Owner string
	// ActivationKey contains an optional activation key used during registration.
	ActivationKey string
}

// CreateConsumer creates a Candlepin consumer and requests an identity certificate.
func (c *Client) CreateConsumer(
	ctx context.Context,
	request CreateConsumerRequest,
	options CreateConsumerOptions,
) (Consumer, error) {
	query := url.Values{}
	if options.Owner != "" {
		query.Set("owner", options.Owner)
	}

	if options.ActivationKey != "" {
		query.Set("activation_keys", options.ActivationKey)
	}

	query.Set("identity_cert_creation", "true")

	slog.Debug("creating Candlepin consumer", "owner", options.Owner, "facts", len(request.Facts))

	var consumer Consumer

	httpRequest, err := httpclient.NewJSONRequest(http.MethodPost, "/consumers", query, request)
	if err != nil {
		return Consumer{}, fmt.Errorf("create consumer: %w", err)
	}

	response, err := c.http.Do(ctx, httpRequest)
	if err != nil {
		return Consumer{}, fmt.Errorf("create consumer: %w", err)
	}

	if err := json.Unmarshal(response.Body, &consumer); err != nil {
		return Consumer{}, fmt.Errorf("decode consumer: %w", err)
	}

	slog.Debug("Candlepin consumer received")

	return consumer, nil
}

// DeleteConsumer removes a consumer by UUID.
func (c *Client) DeleteConsumer(ctx context.Context, uuid string) error {
	if uuid == "" {
		return fmt.Errorf("consumer UUID must not be empty")
	}

	// The configured subscriptions endpoint already includes the /subscription
	// service path, so append only the consumer resource path here.
	path := "/consumers/" + url.PathEscape(uuid)
	slog.Debug("deleting Candlepin consumer", "uuid", uuid)

	if _, err := c.http.Do(ctx, httpclient.Request{Method: http.MethodDelete, Path: path}); err != nil {
		return fmt.Errorf("delete consumer: %w", err)
	}

	return nil
}
