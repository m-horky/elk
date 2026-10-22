// Package httpclient provides a shared HTTP client with configurable TLS,
// proxy, timeout, and redirect policies.
//
// Construct a client from a configuration and use it for requests against the
// configured base URL:
//
//	client, err := httpclient.New(httpclient.Config{
//		BaseURL:     "https://example.test",
//		CAFile:      "/etc/rhsm/ca/redhat-uep.pem",
//		Proxy:       httpclient.ProxyConfig{URI: "http://proxy.example.test:3128"},
//	})
//	if err != nil {
//		return err
//	}
//	defer client.Close()
//
// Redirects are restricted to the configured origin unless AllowRedirects is
// disabled. A CAFile of "system" uses the system certificate pool. Proxy
// credentials may be supplied either through ProxyConfig fields or through
// userinfo embedded in the proxy URI; NoProxy lists bypass hosts.
package httpclient
