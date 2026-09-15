// Command test-elk-canelepin is an exploratory Candlepin registration client.
// It creates one consumer and stores the returned identity certificate in the
// configured certificate store. It is not a production credential manager.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/m-horky/elk/internal/candlepin"
	"github.com/m-horky/elk/internal/httpclient"
	"github.com/m-horky/elk/pkg/certstore"
	"github.com/m-horky/elk/pkg/config"
)

// TODO Stop importing internal packages

func main() { //nolint:funlen
	username := flag.String("username", "", "Candlepin username")
	password := flag.String("password", "", "Candlepin password")

	flag.Parse()

	if *username == "" || *password == "" {
		fatal(errors.New("--username and --password are required")) //nolint:err113
	}

	cfg, err := config.Get()
	if err != nil {
		fatal(fmt.Errorf("load configuration: %w", err))
	}

	endpoint := cfg.API.Subscriptions
	if endpoint.URI == "" {
		//nolint:lll
		fatal(errors.New("candlepin subscriptions endpoint is not configured")) //nolint:err113
	}

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
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	owners, err := client.ListUserOwners(ctx, *username)
	if err != nil {
		fatal(err)
	}

	if len(owners) == 0 {
		//nolint:lll
		fatal(errors.New("candlepin returned no organizations for the user")) //nolint:err113
	}

	if len(owners) != 1 {
		//nolint:lll
		message := fmt.Sprintf("candlepin returned %d organizations; selecting one automatically is unsafe", len(owners))
		fatal(errors.New(message)) //nolint:err113
	}

	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		name = "elk-test-identity"
	}

	consumer, err := client.CreateConsumer(ctx, candlepin.CreateConsumerRequest{
		Name: name,
		Type: candlepin.ConsumerType{Label: "system"},
	}, candlepin.CreateConsumerOptions{Owner: owners[0].Key})
	if err != nil {
		fatal(err)
	}

	if consumer.UUID == "" || consumer.IDCert == nil || consumer.IDCert.Cert == "" || consumer.IDCert.Key == "" {
		//nolint:lll
		fatal(errors.New("candlepin created the consumer without complete identity certificate material")) //nolint:err113
	}

	identityStore, err := certstore.NewIdentityStore()
	if err != nil {
		fatal(fmt.Errorf("open identity certificate store: %w", err))
	}

	if err := identityStore.Save(certstore.KeyPair{
		CertificatePEM: []byte(consumer.IDCert.Cert),
		PrivateKeyPEM:  []byte(consumer.IDCert.Key),
	}); err != nil {
		fatal(fmt.Errorf("save consumer identity certificate: %w", err))
	}

	fmt.Printf("created consumer %s for organization %s\n", consumer.UUID, owners[0].Key)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
