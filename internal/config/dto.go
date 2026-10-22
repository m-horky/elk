package config

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

// dtoConfig contains one partial native configuration layer.
type dtoConfig struct {
	Compatibility dtoCompatibility `toml:"compatibility"`
	HTTP          dtoHTTP          `toml:"http"`
	API           dtoAPI           `toml:"api"`
}

// dtoCompatibility contains partial compatibility preferences.
type dtoCompatibility struct {
	InterpretLegacy *bool `toml:"interpret-legacy-configurations"`
}

// dtoHTTP contains partial outgoing HTTP settings.
type dtoHTTP struct {
	Timeout dtoTimeout `toml:"timeout"`
	Proxy   dtoProxy   `toml:"proxy"`
}

// dtoTimeout contains optional timeout values measured in seconds.
type dtoTimeout struct {
	Connect *int64 `toml:"connect"`
	Request *int64 `toml:"request"`
	Idle    *int64 `toml:"idle"`
}

// dtoProxy contains optional proxy settings, including credentials and bypass rules.
type dtoProxy struct {
	URI      *string `toml:"uri"`
	User     *string `toml:"user"`
	Password *string `toml:"password"`
	NoProxy  *string `toml:"no-proxy"`
}

// dtoAPI contains partial API endpoint settings.
type dtoAPI struct {
	Subscriptions dtoEndpoint `toml:"subscriptions"`
	Content       dtoContent  `toml:"content"`
	Insights      dtoInsights `toml:"insights"`
}

// dtoContent contains partial content-delivery endpoint settings.
type dtoContent struct {
	RPM dtoEndpoint `toml:"rpm"`
}

// dtoInsights contains partial Insights endpoint settings.
type dtoInsights struct {
	Ingress   dtoEndpoint `toml:"ingress"`
	Inventory dtoEndpoint `toml:"inventory"`
}

// dtoEndpoint contains optional endpoint and TLS settings.
type dtoEndpoint struct {
	URI       *string `toml:"uri"`
	TLSVerify *bool   `toml:"tls-verify"`
	CAPath    *string `toml:"ca-path"`
}

// update validates and merges the supplied partial configuration into the current dtoConfig.
func (partial *dtoConfig) update(next *dtoConfig) error {
	if next == nil {
		return nil
	}

	if next.Compatibility.InterpretLegacy != nil {
		partial.Compatibility.InterpretLegacy = next.Compatibility.InterpretLegacy
	}

	if err := partial.HTTP.Timeout.update(next.HTTP.Timeout); err != nil {
		return fmt.Errorf("invalid http.timeout: %w", err)
	}

	if err := partial.HTTP.Proxy.update(next.HTTP.Proxy); err != nil {
		return fmt.Errorf("invalid http.proxy: %w", err)
	}

	if err := partial.API.Subscriptions.update(next.API.Subscriptions); err != nil {
		return fmt.Errorf("invalid api.subscriptions: %w", err)
	}

	if err := partial.API.Content.RPM.update(next.API.Content.RPM); err != nil {
		return fmt.Errorf("invalid api.content.rpm: %w", err)
	}

	if err := partial.API.Insights.Ingress.update(next.API.Insights.Ingress); err != nil {
		return fmt.Errorf("invalid api.insights.ingress: %w", err)
	}

	if err := partial.API.Insights.Inventory.update(next.API.Insights.Inventory); err != nil {
		return fmt.Errorf("invalid api.insights.inventory: %w", err)
	}

	return nil
}

// update validates and merges HTTP timeout preferences.
func (partial *dtoTimeout) update(next dtoTimeout) error {
	if next.Connect != nil {
		if !isTimeDuration(*next.Connect) {
			return fmt.Errorf("invalid connect duration: %d", *next.Connect)
		}

		partial.Connect = next.Connect
	}

	if next.Request != nil {
		if !isTimeDuration(*next.Request) {
			return fmt.Errorf("invalid request duration: %d", *next.Request)
		}

		partial.Request = next.Request
	}

	if next.Idle != nil {
		if !isTimeDuration(*next.Idle) {
			return fmt.Errorf("invalid idle duration: %d", *next.Idle)
		}

		partial.Idle = next.Idle
	}

	return nil
}

// update validates and merges proxy URLs, credentials, and bypass rules.
func (partial *dtoProxy) update(next dtoProxy) error {
	if next.URI != nil {
		// An explicit empty URI disables an inherited proxy.
		if *next.URI != "" {
			if err := isValidURI(*next.URI); err != nil {
				return err
			}
		}

		partial.URI = new(*next.URI)
	}

	if next.User != nil {
		partial.User = new(*next.User)
	}

	if next.Password != nil {
		partial.Password = new(*next.Password)
	}

	if next.NoProxy != nil && *next.NoProxy != "" {
		partial.NoProxy = new(*next.NoProxy)
	}

	return nil
}

// update validates and merges endpoint URL and TLS settings.
func (partial *dtoEndpoint) update(next dtoEndpoint) error {
	if next.URI != nil {
		if err := isValidURI(*next.URI); err != nil {
			return err
		}

		partial.URI = new(*next.URI)
	}

	if next.TLSVerify != nil {
		partial.TLSVerify = next.TLSVerify
	}

	if next.CAPath != nil {
		partial.CAPath = new(strings.TrimSpace(*next.CAPath))
	}

	return nil
}

// isValidURI validates a supplied URI.
func isValidURI(value string) error {
	if s := strings.TrimSpace(value); s == "" {
		return errors.New("no content")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid URI: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("scheme must be http or https")
	}

	if parsed.Hostname() == "" {
		return errors.New("no hostname")
	}

	return nil
}

// isTimeDuration validates the input can be represented as a time.Duration.
func isTimeDuration(seconds int64) bool {
	return seconds >= 0 && seconds <= math.MaxInt64/int64(time.Second)
}
