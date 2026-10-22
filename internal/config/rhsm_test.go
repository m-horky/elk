package config

import (
	"strings"
	"testing"

	"github.com/m-horky/elk/internal/config/rhsm"
)

// TestMapRHSMSubscriptions verifies the server endpoint derivation.
//
// Given stage, production, and Satellite Candlepin hosts, when the
// subscriptions endpoint is derived, then the port 443 is omitted and other
// ports are joined to the host.
func TestMapRHSMSubscriptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host string
		port uint16
		want string
	}{
		{"subscription.rhsm.redhat.com", 443, "https://subscription.rhsm.redhat.com"},
		{"satellite.example.org", 8443, "https://satellite.example.org:8443"},
		{"203.0.113.3", 8443, "https://203.0.113.3:8443"},
		{"2001:db8::1", 8443, "https://[2001:db8::1]:8443"},
	}

	for _, test := range tests {
		t.Run(test.host, func(t *testing.T) {
			source := rhsm.Config{Server: rhsm.Server{Hostname: test.host, Port: test.port}}

			got, err := mapRHSMSubscriptions(source)
			if err != nil {
				t.Fatalf("mapRHSMSubscriptions() error = %v", err)
			}

			if *got.URI != test.want {
				t.Errorf("mapRHSMSubscriptions() = %q, want %q", *got.URI, test.want)
			}
		})
	}
}

// TestMapRHSMInsights verifies Insights endpoints are derived from the server target.
//
// Given stage, production, and Satellite Candlepin hosts, when the Insights
// endpoints are derived, then only matching domain-label suffixes map to Red
// Hat services.
func TestMapRHSMInsights(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host  string
		ing   string
		inven string
	}{
		{
			"foo.rhsm.stage.redhat.com",
			"https://cert.console.stage.redhat.com/api/ingress/v1",
			"https://cert.console.stage.redhat.com/api/inventory/v1",
		},
		{"subscription.rhsm.redhat.com", "", ""},
		{"SUBSCRIPTION.RHSM.REDHAT.COM", "", ""},
		{
			"satellite.example.com",
			"https://satellite.example.com/redhat_access/r/insights/platform/ingress/v1",
			"https://satellite.example.com/redhat_access/r/insights/platform/inventory/v1",
		},
	}

	for _, test := range tests {
		t.Run(test.host, func(t *testing.T) {
			source := rhsm.Config{Server: rhsm.Server{Hostname: test.host, Port: 443}}

			got, err := mapRHSMInsights(source)
			if err != nil {
				t.Fatalf("mapRHSMInsights() error = %v", err)
			}

			if test.ing == "" {
				if got.Ingress.URI != nil || got.Inventory.URI != nil {
					t.Errorf("mapRHSMInsights() = %+v, want untouched endpoints", got)
				}

				return
			}

			if *got.Ingress.URI != test.ing {
				t.Errorf("mapRHSMInsights() ingress = %q, want %q", *got.Ingress.URI, test.ing)
			}

			if *got.Inventory.URI != test.inven {
				t.Errorf("mapRHSMInsights() inventory = %q, want %q", *got.Inventory.URI, test.inven)
			}
		})
	}
}

// TestMapRHSMContent verifies the content endpoint derivation.
//
// Given RHSM settings with and without a base URL, when the content endpoint
// is derived, then an absent URL leaves the endpoint untouched and a present
// URL is applied.
func TestMapRHSMContent(t *testing.T) {
	t.Parallel()

	source := rhsm.Config{RHSM: rhsm.RHSM{}}

	got, err := mapRHSMContent(source)
	if err != nil {
		t.Fatalf("mapRHSMContent() error = %v", err)
	}

	if got.URI != nil {
		t.Errorf("mapRHSMContent() = %+v, want untouched endpoint", got)
	}

	source.RHSM.BaseURL = "https://cdn.example.org"

	got, err = mapRHSMContent(source)
	if err != nil {
		t.Fatalf("mapRHSMContent() error = %v", err)
	}

	if *got.URI != "https://cdn.example.org" {
		t.Errorf("mapRHSMContent() = %q, want %q", *got.URI, "https://cdn.example.org")
	}
}

// TestMapRHSMContentRejectsInvalidURL verifies malformed base URLs are rejected.
//
// Given RHSM settings with a non-HTTPS base URL, when the content endpoint is
// derived, then an error is returned.
func TestMapRHSMContentRejectsInvalidURL(t *testing.T) {
	t.Parallel()

	source := rhsm.Config{RHSM: rhsm.RHSM{BaseURL: "http://cdn.example.org"}}

	if _, err := mapRHSMContent(source); err == nil {
		t.Fatal("mapRHSMContent() error = nil, want invalid URL error")
	}
}

// TestMapRHSMProxy verifies proxy derivation from RHSM settings.
//
// Given RHSM proxy settings with and without explicit values, when the proxy
// is derived, then defaults are applied for the missing scheme and port.
func TestMapRHSMProxy(t *testing.T) {
	t.Parallel()

	source := rhsm.Config{Server: rhsm.Server{ProxyHostname: "rhsm.example.org"}}

	got := mapRHSMProxy(source)
	if *got.URI != "http://rhsm.example.org:3128" {
		t.Errorf("mapRHSMProxy() = %q, want %q", *got.URI, "http://rhsm.example.org:3128")
	}

	source.Server.ProxyScheme = "https"
	source.Server.ProxyPort = 8888
	source.Server.ProxyUser = "user"
	source.Server.ProxyPassword = "secret"
	source.Server.NoProxy = "localhost"

	got = mapRHSMProxy(source)
	if *got.URI != "https://rhsm.example.org:8888" {
		t.Errorf("mapRHSMProxy() = %q, want %q", *got.URI, "https://rhsm.example.org:8888")
	}

	if *got.User != "user" || *got.Password != "secret" {
		t.Errorf("mapRHSMProxy() credentials = %q/%q, want user/secret", *got.User, *got.Password)
	}

	if *got.NoProxy != "localhost" {
		t.Errorf("mapRHSMProxy() no-proxy = %q, want %q", *got.NoProxy, "localhost")
	}
}

// TestMapRHSMProxyWithoutHost verifies a missing proxy hostname leaves the proxy untouched.
//
// Given RHSM settings without a proxy hostname, when the proxy is derived,
// then no proxy fields are set.
func TestMapRHSMProxyWithoutHost(t *testing.T) {
	t.Parallel()

	source := rhsm.Config{Server: rhsm.Server{ProxyPort: 3128}}

	got := mapRHSMProxy(source)
	if got.URI != nil || got.User != nil || got.Password != nil || got.NoProxy != nil {
		t.Errorf("mapRHSMProxy() = %+v, want untouched proxy", got)
	}
}

// TestHasLabelSuffix verifies we correctly match hostname suffix.
//
// Given a FQDN segment, when we compare it to another segment,
// 'true' is returned when they share the suffix.
func TestHasLabelSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefix []string
		ok     bool
	}{
		{[]string{"a"}, false},
		{[]string{"b", "a"}, false},
		{[]string{"c", "b", "a"}, true},
		{[]string{"d", "c", "b", "a"}, true},
		{[]string{"c", "d", "b", "a"}, false},
		{[]string{"c", "b", "a", "x"}, false},
	}

	prefix := []string{"c", "b", "a"}

	for _, test := range tests {
		t.Run(strings.Join(test.prefix, " "), func(t *testing.T) {
			ok := hasLabelSuffix(test.prefix, prefix)

			if ok != test.ok {
				t.Errorf("hasLabelSuffix() = %v, want %v", ok, test.ok)
			}
		})
	}
}
