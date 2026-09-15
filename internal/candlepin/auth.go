package candlepin

import (
	"fmt"

	"github.com/m-horky/elk/internal/httpclient"
)

// BasicCredentials contains credentials for Candlepin HTTP Basic authentication.
type BasicCredentials struct {
	Username string
	Password string
}

// NewBasicClient constructs a Candlepin client authenticated with HTTP Basic authentication.
func NewBasicClient(cfg Config, credentials BasicCredentials) (*Client, error) {
	if credentials.Username == "" {
		return nil, fmt.Errorf("candlepin username must not be empty") //nolint:err113
	}

	cfg.HTTP.BasicAuth = &httpclient.BasicAuth{
		Username: credentials.Username,
		Password: credentials.Password,
	}

	client, err := httpclient.New(cfg.HTTP)
	if err != nil {
		return nil, fmt.Errorf("construct Candlepin BASIC client: %w", err)
	}

	return &Client{http: client}, nil
}

// TODO Cert auth
// TODO Activation key auth
