package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/m-horky/elk/pkg/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(); err != nil {
		slog.Error("command failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	slog.Info("loading configuration")

	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	fmt.Println("http.timeout:")
	fmt.Printf("  connect: %s\n", cfg.HTTP.Timeout.Connect)
	fmt.Printf("  request: %s\n", cfg.HTTP.Timeout.Request)
	fmt.Printf("  idle:    %s\n", cfg.HTTP.Timeout.Idle)

	fmt.Println("http.proxy:")
	fmt.Printf("  uri:       %s\n", redactCredentials(cfg.HTTP.Proxy.URI))
	fmt.Printf("  no-proxy:  %s\n", cfg.HTTP.Proxy.NoProxy)

	fmt.Println("api.subscriptions:")
	printEndpoint(cfg.API.Subscriptions)
	fmt.Println("api.content.rpm:")
	printEndpoint(cfg.API.Content.RPM)
	fmt.Println("api.insights.ingress:")
	printEndpoint(cfg.API.Insights.Ingress)
	fmt.Println("api.insights.inventory:")
	printEndpoint(cfg.API.Insights.Inventory)

	slog.Info("configuration output written")

	return nil
}

// printEndpoint writes one resolved endpoint.
func printEndpoint(endpoint config.Endpoint) {
	fmt.Printf("  uri:       %s\n", endpoint.URI.String())
	fmt.Printf("  tls-verify: %t\n", endpoint.TLSVerify)
	fmt.Printf("  ca-path:   %s\n", endpoint.CAPath)
}

// redactCredentials removes userinfo from a URI before it is written to output.
func redactCredentials(uri url.URL) string {
	if uri.User == nil {
		return uri.String()
	}

	redacted := uri
	redacted.User = nil

	return redacted.String()
}
