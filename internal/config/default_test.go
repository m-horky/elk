package config

import "testing"

// TestDefaultHasNoNils verifies the embedded defaults resolve without missing values.
//
// Given the embedded default configuration, when it is decoded into a partial,
// then every supported key has a value.
func TestDefaultHasNoNils(t *testing.T) {
	t.Parallel()

	partial, err := newPartialFromDefaults()
	if err != nil {
		t.Fatal(err)
	}

	checks := []struct {
		name string
		ok   func() bool
	}{
		{
			"compatibility.interpret-legacy",
			func() bool { return partial.Compatibility.InterpretLegacy != nil },
		},
		{"http.timeout.connect", func() bool { return partial.HTTP.Timeout.Connect != nil }},
		{"http.timeout.request", func() bool { return partial.HTTP.Timeout.Request != nil }},
		{"http.timeout.idle", func() bool { return partial.HTTP.Timeout.Idle != nil }},
		{"http.proxy.uri", func() bool { return partial.HTTP.Proxy.URI != nil }},
		{"http.proxy.user", func() bool { return partial.HTTP.Proxy.User != nil }},
		{"http.proxy.password", func() bool { return partial.HTTP.Proxy.Password != nil }},
		{"http.proxy.no-proxy", func() bool { return partial.HTTP.Proxy.NoProxy != nil }},
		{"api.subscriptions.uri", func() bool { return partial.API.Subscriptions.URI != nil }},
		{"api.subscriptions.ca-path", func() bool { return partial.API.Subscriptions.CAPath != nil }},
		{"api.subscriptions.tls-verify", func() bool { return partial.API.Subscriptions.TLSVerify != nil }},
		{"api.content.rpm.uri", func() bool { return partial.API.Content.RPM.URI != nil }},
		{"api.content.rpm.ca-path", func() bool { return partial.API.Content.RPM.CAPath != nil }},
		{"api.content.rpm.tls-verify", func() bool { return partial.API.Content.RPM.TLSVerify != nil }},
		{"api.insights.ingress.uri", func() bool { return partial.API.Insights.Ingress.URI != nil }},
		{"api.insights.inventory.uri", func() bool { return partial.API.Insights.Inventory.URI != nil }},
	}

	for _, check := range checks {
		if !check.ok() {
			t.Errorf("%s is nil in the default configuration", check.name)
		}
	}
}
