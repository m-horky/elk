package config

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/m-horky/elk/internal/config/rhsm"
)

// newPartialFromRHSM translates resolved RHSM settings into a partial native layer.
func newPartialFromRHSM(source rhsm.Config) (dtoConfig, error) {
	subscriptions, err := mapRHSMSubscriptions(source)
	if err != nil {
		return dtoConfig{}, err
	}

	content, err := mapRHSMContent(source)
	if err != nil {
		return dtoConfig{}, err
	}

	insights, err := mapRHSMInsights(source)
	if err != nil {
		return dtoConfig{}, err
	}

	return dtoConfig{
		HTTP: dtoHTTP{Proxy: mapRHSMProxy(source)},
		API: dtoAPI{
			Subscriptions: subscriptions,
			Content:       dtoContent{RPM: content},
			Insights:      insights,
		},
	}, nil
}

// mapRHSMSubscriptions maps the RHSM server settings to the subscriptions endpoint.
func mapRHSMSubscriptions(source rhsm.Config) (dtoEndpoint, error) {
	uri := url.URL{
		Scheme: "https",
		Host:   buildURIHost(source.Server.Hostname, source.Server.Port),
		Path:   source.Server.Prefix,
	}

	return dtoEndpoint{
		URI:       new(uri.String()),
		TLSVerify: new(source.Server.TLSVerify),
		CAPath:    new(source.RHSM.RepoCACert),
	}, nil
}

// mapRHSMContent maps the RHSM content settings to the RPM endpoint.
// An empty base URL leaves the endpoint untouched.
func mapRHSMContent(source rhsm.Config) (dtoEndpoint, error) {
	baseURL := strings.TrimSpace(source.RHSM.BaseURL)
	if baseURL == "" {
		return dtoEndpoint{}, nil
	}

	if err := isValidHTTPSURI(baseURL); err != nil {
		return dtoEndpoint{}, errors.New("invalid rhsm baseurl")
	}

	result := dtoEndpoint{
		URI:       new(baseURL),
		TLSVerify: new(source.Server.TLSVerify),
	}

	if ca := strings.TrimSpace(source.RHSM.RepoCACert); ca != "" {
		result.CAPath = new(ca)
	}

	return result, nil
}

// mapRHSMInsights maps the RHSM server settings to Insights endpoints
// according to the target system type. Production RHSM keeps the default
// endpoints.
func mapRHSMInsights(source rhsm.Config) (dtoInsights, error) {
	if hasLabelSuffix(strings.Split(source.Server.Hostname, "."), []string{"rhsm", "stage", "redhat", "com"}) {
		base := "https://cert.console.stage.redhat.com/api"

		return dtoInsights{
			Ingress:   dtoEndpoint{URI: new(base + "/ingress/v1"), TLSVerify: new(source.Server.TLSVerify)},
			Inventory: dtoEndpoint{URI: new(base + "/inventory/v1"), TLSVerify: new(source.Server.TLSVerify)},
		}, nil
	}

	if hasLabelSuffix(strings.Split(source.Server.Hostname, "."), []string{"rhsm", "redhat", "com"}) {
		return dtoInsights{}, nil
	}

	base := url.URL{
		Scheme: "https",
		Host:   buildURIHost(source.Server.Hostname, source.Server.Port),
		Path:   "/redhat_access/r/insights/platform",
	}

	ingress := dtoEndpoint{URI: new(base.String() + "/ingress/v1"), TLSVerify: new(source.Server.TLSVerify)}
	inventory := dtoEndpoint{URI: new(base.String() + "/inventory/v1"), TLSVerify: new(source.Server.TLSVerify)}

	if ca := strings.TrimSpace(source.RHSM.RepoCACert); ca != "" {
		ingress.CAPath = new(ca)
		inventory.CAPath = new(ca)
	}

	return dtoInsights{Ingress: ingress, Inventory: inventory}, nil
}

// mapRHSMProxy maps the RHSM proxy settings to HTTP proxy configuration.
// A missing proxy hostname leaves the proxy untouched.
func mapRHSMProxy(source rhsm.Config) dtoProxy {
	host := strings.TrimSpace(source.Server.ProxyHostname)
	if host == "" {
		return dtoProxy{}
	}

	port := source.Server.ProxyPort
	if port == 0 {
		port = 3128
	}

	scheme := strings.ToLower(strings.TrimSpace(source.Server.ProxyScheme))
	// Only https is honored; any other value, including the default, falls back to http.
	if scheme != "https" {
		scheme = "http"
	}

	proxy := dtoProxy{
		URI: new((&url.URL{
			Scheme: scheme,
			Host:   net.JoinHostPort(host, strconv.Itoa(int(port))),
		}).String()),
	}

	if user := source.Server.ProxyUser; user != "" {
		proxy.User = new(user)
	}

	if password := source.Server.ProxyPassword; password != "" {
		proxy.Password = new(password)
	}

	if noProxy := strings.TrimSpace(source.Server.NoProxy); noProxy != "" {
		proxy.NoProxy = new(noProxy)
	}

	return proxy
}

// buildURIHost omits :443 in a 'host:port' string.
func buildURIHost(hostname string, port uint16) string {
	if port == 443 {
		return hostname
	}

	return net.JoinHostPort(hostname, strconv.Itoa(int(port)))
}

// isValidHTTPSURI accepts HTTPS URLs without credentials, queries, or fragments.
func isValidHTTPSURI(value string) error {
	u, err := url.ParseRequestURI(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil ||
		strings.Contains(value, "#") || u.Fragment != "" || u.RawQuery != "" {
		return errors.New("invalid HTTPS URL")
	}

	return nil
}

// hasLabelSuffix compares DNS labels against a provided DNS label suffix.
func hasLabelSuffix(host, suffix []string) bool {
	if len(host) < len(suffix) {
		return false
	}

	offset := len(host) - len(suffix)
	for index, label := range suffix {
		if !strings.EqualFold(host[offset+index], label) {
			return false
		}
	}

	return true
}
