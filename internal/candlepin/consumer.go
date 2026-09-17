package candlepin

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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

	_, err := c.http.DoJSON(ctx, http.MethodPost, "/consumers", query, request, &consumer)
	if err != nil {
		return Consumer{}, fmt.Errorf("create consumer: %w", err)
	}

	slog.Debug("Candlepin consumer received", "uuid_present", consumer.UUID != "", "identity_certificate_present", consumer.IDCert != nil)
	return consumer, nil
}
