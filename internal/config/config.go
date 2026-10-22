package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config contains resolved native configuration.
type Config struct {
	HTTP HTTP
	API  API
}

// HTTP contains resolved outgoing HTTP settings.
type HTTP struct {
	Timeout Timeout
	Proxy   Proxy
}

// Timeout contains resolved transport timeouts.
type Timeout struct {
	Connect time.Duration
	Request time.Duration
	Idle    time.Duration
}

// Proxy contains a proxy URL (including credentials) and bypass rules.
type Proxy struct {
	URI     url.URL
	NoProxy string
}

// API contains remote endpoints.
type API struct {
	Subscriptions Endpoint
	Content       Content
	Insights      Insights
}

// Content contains content-delivery endpoints.
type Content struct {
	RPM Endpoint
}

// Insights contains Insights endpoints.
type Insights struct {
	Ingress   Endpoint
	Inventory Endpoint
}

// Endpoint contains an endpoint URL and TLS settings.
type Endpoint struct {
	URI       url.URL
	TLSVerify bool
	CAPath    string
}

// resolve converts merged native settings into complete resolved sections.
func (partial *dtoConfig) resolve() (Config, error) {
	subscriptions, err := resolveEndpoint("api.subscriptions", partial.API.Subscriptions)
	if err != nil {
		return Config{}, err
	}

	rpm, err := resolveEndpoint("api.content.rpm", partial.API.Content.RPM)
	if err != nil {
		return Config{}, err
	}

	ingress, err := resolveEndpoint("api.insights.ingress", partial.API.Insights.Ingress)
	if err != nil {
		return Config{}, err
	}

	inventory, err := resolveEndpoint("api.insights.inventory", partial.API.Insights.Inventory)
	if err != nil {
		return Config{}, err
	}

	proxy, err := resolveProxy(partial.HTTP.Proxy)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP: HTTP{
			Timeout: Timeout{
				Connect: time.Duration(*partial.HTTP.Timeout.Connect) * time.Second,
				Request: time.Duration(*partial.HTTP.Timeout.Request) * time.Second,
				Idle:    time.Duration(*partial.HTTP.Timeout.Idle) * time.Second,
			},
			Proxy: proxy,
		},
		API: API{
			Subscriptions: subscriptions,
			Content:       Content{RPM: rpm},
			Insights:      Insights{Ingress: ingress, Inventory: inventory},
		},
	}, nil
}

// resolveEndpoint parses the configured endpoint and resolves its TLS settings.
func resolveEndpoint(name string, partial dtoEndpoint) (Endpoint, error) {
	if partial.URI == nil || *partial.URI == "" || partial.TLSVerify == nil || partial.CAPath == nil {
		return Endpoint{}, fmt.Errorf("resolve %s: required setting is empty or missing", name)
	}

	if err := isValidURI(*partial.URI); err != nil {
		return Endpoint{}, fmt.Errorf("resolve %s.uri: %w", name, err)
	}

	uri, err := url.Parse(*partial.URI)
	if err != nil {
		return Endpoint{}, fmt.Errorf("resolve %s.uri: %w", name, err)
	}

	return Endpoint{URI: *uri, TLSVerify: *partial.TLSVerify, CAPath: *partial.CAPath}, nil
}

// resolveProxy parses the configured proxy URI, resolves optional credentials, and bypass rules.
func resolveProxy(partial dtoProxy) (Proxy, error) {
	if partial.URI == nil || *partial.URI == "" {
		return Proxy{}, nil
	}

	if err := isValidURI(*partial.URI); err != nil {
		return Proxy{}, fmt.Errorf("resolve http.proxy.uri: %w", err)
	}

	uri, err := url.Parse(*partial.URI)
	if err != nil {
		return Proxy{}, fmt.Errorf("resolve http.proxy.uri: %w", err)
	}

	user, password := "", ""
	if partial.User != nil {
		user = *partial.User
	}

	if partial.Password != nil {
		password = *partial.Password
	}

	if user != "" || password != "" {
		uri.User = url.UserPassword(user, password)
	}

	noProxy := ""
	if partial.NoProxy != nil {
		noProxy = strings.TrimSpace(*partial.NoProxy)
	}

	return Proxy{
		URI:     *uri,
		NoProxy: noProxy,
	}, nil
}
