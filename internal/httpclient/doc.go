// Package httpclient provides a shared HTTP client with configurable TLS,
// proxy, timeout, and same-origin redirect policies.
//
// # Constructing a client
//
// Construct a client from a configuration and use it for requests against the
// configured base URL:
//
//	client, _ := httpclient.New(httpclient.Config{
//		BaseURL:     "https://example.test",
//		CAFile:      "/etc/rhsm/ca/redhat-uep.pem",
//		Proxy:       httpclient.ProxyConfig{URI: "http://proxy.example.test:3128"},
//	})
//	defer client.Close()
//
// A CAFile of "system" uses the system certificate pool. Certificate and
// BasicAuth configure client authentication. InsecureTLS disables certificate
// verification and should only be used for diagnostics. ConnectTimeout,
// RequestTimeout, and IdleTimeout control the transport and request lifetimes.
// MaxResponseBody bounds response bodies; zero uses the package default.
//
// Proxy credentials may be supplied either through ProxyConfig fields or through
// userinfo embedded in the proxy URI; NoProxy lists bypass hosts. The
// client ignores proxy environment variables; proxies must be configured
// explicitly.
//
// # Executing requests
//
// Do is the request executor. It resolves Request.Path against the client's
// base URL, applies Request.Query, Request.Headers, and Request.Body, and
// returns the successful response. An empty Request.Method is treated as GET.
//
//	response, _ := client.Do(ctx, httpclient.Request{
//		Method: http.MethodGet,
//		Path:   "/status",
//	})
//	fmt.Println(string(response.Body))
//
// Get is a convenience for a GET request without a query or body:
//
//	response, _ := client.Get(ctx, "/status")
//
// NewJSONRequest builds a Request whose body is JSON encoded and whose
// Content-Type is application/json. A nil body produces a request without a
// body. The caller executes the request with Do and decodes the response body:
//
//	request, _ := httpclient.NewJSONRequest(http.MethodPost, "/consumers", nil, payload)
//	response, _ := client.Do(ctx, request)
//	var consumer Consumer
//	_ = json.Unmarshal(response.Body, &consumer)
//
// # Responses and errors
//
// Response contains the status code, headers, and raw body. Successful
// response bodies are bounded by Config.MaxResponseBody and set
// Response.BodyTruncated when the limit is exceeded; an oversized successful
// response is returned together with an error so incomplete data is never
// mistaken for success.
//
// Do and Get return an *HTTPError for non-2xx responses; the error carries
// the sanitized request URL, status, and a bounded copy of the error body.
// Non-2xx bodies are read with the same truncation logic and are additionally
// capped for diagnostics. The original response is nil in that case.
//
// # URLs and caller metadata
//
// Client.URL and JoinURL resolve paths against a base URL while preserving the
// base path. WithTriggeredBy associates caller name and version with a context
// so they appear in the User-Agent header. The client always sets a User-Agent
// identifying elk and the host system.
package httpclient
