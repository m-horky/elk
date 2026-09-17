// Command test-elk-canelepin is an exploratory Candlepin registration client.
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

// TODO Stop importing internal packages

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(); err != nil {
		if !errors.Is(err, errUsage) {
			slog.Error("command failed", "err", err)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error { //nolint:funlen
	username := flag.String("username", "", "Candlepin username")
	password := flag.String("password", "", "Candlepin password")

	flag.Parse()

	if *username == "" || *password == "" {
		return fmt.Errorf("%w: --username and --password are required", errUsage)
	}

	slog.Info("loading configuration")

	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	endpoint := cfg.API.Subscriptions
	if endpoint.URI == "" {
		//nolint:lll
		return errors.New("candlepin subscriptions endpoint is not configured") //nolint:err113
	}

	slog.Info("connecting to Candlepin")

	client, err := candlepin.NewBasicClient(candlepin.Config{
		HTTP: httpclient.Config{
			BaseURL:        endpoint.URI,
			CAFile:         endpoint.CAPath,
			TLSVerify:      endpoint.TLSVerify,
			InsecureTLS:    !endpoint.TLSVerify,
			ConnectTimeout: cfg.HTTP.Timeout.Connect,
			RequestTimeout: cfg.HTTP.Timeout.Request,
			IdleTimeout:    cfg.HTTP.Timeout.Idle,
			Proxy: httpclient.ProxyConfig{
				URI:      cfg.HTTP.Proxy.URI,
				Username: cfg.HTTP.Proxy.User,
				Password: cfg.HTTP.Proxy.Password,
				NoProxy:  cfg.HTTP.Proxy.NoProxy,
			},
			AllowRedirects: true,
		},
	}, candlepin.BasicCredentials{Username: *username, Password: *password})
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	slog.Info("fetching Candlepin organizations")
	owners, err := client.ListUserOwners(ctx, *username)
	if err != nil {
		return err
	}

	if len(owners) == 0 {
		//nolint:lll
		return errors.New("candlepin returned no organizations for the user") //nolint:err113
	}

	if len(owners) != 1 {
		//nolint:lll
		message := fmt.Sprintf("candlepin returned %d organizations; selecting one automatically is unsafe", len(owners))
		return errors.New(message) //nolint:err113
	}

	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		slog.Warn("hostname unavailable, using fallback consumer name", "name", "elk-test-identity", "err", err)
		name = "elk-test-identity"
	}

	systemFacts, err := facts.Collect()
	if err != nil {
		message := fmt.Sprintf("failed to collect system facts: %v", err)
		return errors.New(message)
	}

	slog.Info("creating Candlepin consumer", "organization", owners[0].Key)

	consumer, err := client.CreateConsumer(ctx, candlepin.CreateConsumerRequest{
		Name:  name,
		Type:  candlepin.ConsumerType{Label: "system"},
		Facts: candlepin.NewFactsDTO(systemFacts),
	}, candlepin.CreateConsumerOptions{Owner: owners[0].Key})
	if err != nil {
		return err
	}

	if consumer.UUID == "" || consumer.IDCert == nil || consumer.IDCert.Cert == "" || consumer.IDCert.Key == "" {
		//nolint:lll
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

var errUsage = errors.New("invalid command usage")
