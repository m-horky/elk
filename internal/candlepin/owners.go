package candlepin

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

// ListUserOwners returns the organizations associated with a user.
func (c *Client) ListUserOwners(ctx context.Context, username string) ([]Owner, error) {
	if username == "" {
		return nil, fmt.Errorf("username must not be empty") //nolint:err113
	}

	var owners []Owner

	slog.Debug("listing Candlepin organizations")
	_, err := c.http.DoJSON(ctx, http.MethodGet, "/users/"+url.PathEscape(username)+"/owners", nil, nil, &owners)
	if err != nil {
		return nil, fmt.Errorf("list owners for user: %w", err)
	}

	slog.Debug("Candlepin organizations received", "count", len(owners))
	return owners, nil
}
