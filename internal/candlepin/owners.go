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

// ListUserOwners returns the organizations associated with a user.
func (c *Client) ListUserOwners(ctx context.Context, username string) ([]Owner, error) {
	if username == "" {
		return nil, fmt.Errorf("username must not be empty")
	}

	var owners []Owner

	slog.Debug("listing Candlepin organizations")

	httpRequest, err := httpclient.NewJSONRequest(http.MethodGet, "/users/"+url.PathEscape(username)+"/owners", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list owners for user: %w", err)
	}

	response, err := c.http.Do(ctx, httpRequest)
	if err != nil {
		return nil, fmt.Errorf("list owners for user: %w", err)
	}

	if err := json.Unmarshal(response.Body, &owners); err != nil {
		return nil, fmt.Errorf("decode owners: %w", err)
	}

	slog.Debug("Candlepin organizations received", "count", len(owners))

	return owners, nil
}
