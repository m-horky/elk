// Command elk-identity is an exploratory Candlepin registration client.
// It creates one consumer and stores the returned identity certificate in the
// configured certificate store. It is not a production credential manager.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/m-horky/elk/internal/candlepin"
	"github.com/m-horky/elk/internal/httpclient"
	"github.com/m-horky/elk/pkg/certstore"
	"github.com/m-horky/elk/pkg/config"
	"github.com/m-horky/elk/pkg/facts"
)

var errUsage = errors.New("invalid command usage")

func main() {
	slog.SetDefault(
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})),
	)

	if err := run(); err != nil {
		if !errors.Is(err, errUsage) {
			slog.Error("command failed", "err", err)
		}
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("%w: a subcommand is required (register or unregister)", errUsage)
	}

	switch os.Args[1] {
	case "register":
		return runRegister(os.Args[2:])
	case "unregister":
		return runUnregister(os.Args[2:])
	default:
		return fmt.Errorf("%w: unknown subcommand %q", errUsage, os.Args[1])
	}
}

func runRegister(args []string) error { //nolint:funlen
	flags := flag.NewFlagSet("register", flag.ContinueOnError)
	username := flags.String("username", "", "Candlepin username")
	password := flags.String("password", "", "Candlepin password")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	if *username == "" || *password == "" {
		return fmt.Errorf("%w: --username and --password are required", errUsage)
	}

	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	slog.Info("connecting to Candlepin")
	client, err := candlepin.NewBasicClient(candlepinConfig(cfg, cfg.API.Subscriptions), candlepin.BasicCredentials{
		Username: *username,
		Password: *password,
	})
	if err != nil {
		return fmt.Errorf("construct Candlepin client: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	slog.Info("fetching Candlepin organizations")
	owners, err := client.ListUserOwners(ctx, *username)
	if err != nil {
		return fmt.Errorf("list Candlepin organizations: %w", err)
	}

	if len(owners) == 0 {
		return errors.New("candlepin returned no organizations for the user") //nolint:err113
	}
	if len(owners) != 1 {
		return fmt.Errorf("candlepin returned %d organizations; selecting one automatically is unsafe", len(owners))
	}

	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		slog.Warn("hostname unavailable, using fallback consumer name", "name", "elk-test-identity", "err", err)
		name = "elk-test-identity"
	}
	systemFacts, err := facts.Collect()
	if err != nil {
		return fmt.Errorf("failed to collect system facts: %w", err)
	}

	slog.Info("creating Candlepin consumer", "organization", owners[0].Key)
	consumer, err := client.CreateConsumer(ctx, candlepin.CreateConsumerRequest{
		Name: name, Type: candlepin.ConsumerType{Label: "system"}, Facts: candlepin.NewFactsDTO(systemFacts),
	}, candlepin.CreateConsumerOptions{Owner: owners[0].Key})
	if err != nil {
		return fmt.Errorf("create Candlepin consumer: %w", err)
	}

	if consumer.UUID == "" || consumer.IDCert == nil || consumer.IDCert.Cert == "" || consumer.IDCert.Key == "" {
		return errors.New("candlepin created the consumer without complete identity certificate material") //nolint:err113
	}

	identityStore, err := certstore.NewIdentityStore()
	if err != nil {
		return fmt.Errorf("open identity certificate store: %w", err)
	}
	if err := identityStore.Save(certstore.KeyPair{
		CertificatePEM: []byte(consumer.IDCert.Cert),
		PrivateKeyPEM:  []byte(consumer.IDCert.Key),
	}); err != nil {
		return fmt.Errorf("save consumer identity certificate: %w", err)
	}

	slog.Info("consumer identity saved", "organization", owners[0].Key)
	fmt.Printf("created consumer for organization %s\n", owners[0].Key)

	return nil
}

func runUnregister(args []string) error {
	flags := flag.NewFlagSet("unregister", flag.ContinueOnError)
	// TODO: obtain the UUID from the stored certificate once certificate-reading infrastructure exists.
	uuid := flags.String("uuid", "", "Candlepin consumer UUID")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	if *uuid == "" {
		return fmt.Errorf("%w: --uuid is required", errUsage)
	}

	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	identityStore, err := certstore.NewIdentityStore()
	if err != nil {
		return fmt.Errorf("open identity certificate store: %w", err)
	}
	certificate, err := identityStore.LoadTLS()
	if err != nil {
		return fmt.Errorf("load consumer identity certificate: %w", err)
	}

	client, err := candlepin.NewClient(candlepinConfig(cfg, cfg.API.Subscriptions), certificate)
	if err != nil {
		return fmt.Errorf("construct Candlepin client: %w", err)
	}

	defer func() { _ = client.Close() }()

	if err := client.DeleteConsumer(context.Background(), *uuid); err != nil {
		return fmt.Errorf("delete Candlepin consumer: %w", err)
	}
	if err := identityStore.Delete(); err != nil {
		return fmt.Errorf("delete local consumer identity: %w", err)
	}

	fmt.Println("Host is no longer registered.")

	return nil
}

func candlepinConfig(cfg config.Config, endpoint config.Endpoint) candlepin.Config {
	return candlepin.Config{HTTP: httpclient.Config{
		BaseURL: endpoint.URI, CAFile: endpoint.CAPath, TLSVerify: endpoint.TLSVerify,
		InsecureTLS: !endpoint.TLSVerify, ConnectTimeout: cfg.HTTP.Timeout.Connect,
		RequestTimeout: cfg.HTTP.Timeout.Request, IdleTimeout: cfg.HTTP.Timeout.Idle,
		Proxy: httpclient.ProxyConfig{
			URI: cfg.HTTP.Proxy.URI, Username: cfg.HTTP.Proxy.User,
			Password: cfg.HTTP.Proxy.Password, NoProxy: cfg.HTTP.Proxy.NoProxy,
		}, AllowRedirects: true,
	}}
}
