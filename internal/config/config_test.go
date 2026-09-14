package config

import (
	"testing"
)

// TestGetReturnsEmbeddedDefaults verifies that Get decodes the embedded TOML
// defaults.
//
// Given the compiled default configuration, when Get is called, then the
// returned configuration contains the documented default values.
func TestGetReturnsEmbeddedDefaults(t *testing.T) {
	t.Parallel()

	got, err := Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if !got.Compatibility.InterpretLegacy {
		t.Error("Get().Compatibility.InterpretLegacy = false, want true")
	}
	if got.HTTP.Timeout.Connect != 30 || got.HTTP.Timeout.Idle != 120 {
		t.Errorf("Get().HTTP.Timeout = %+v, want connect 30 and idle 120", got.HTTP.Timeout)
	}
	if got.API.Subscriptions.URI != "https://subscription.rhsm.redhat.com/subscription" {
		t.Errorf("Get().API.Subscriptions.URI = %q, want embedded default", got.API.Subscriptions.URI)
	}
}

// TestConfigUpdatePreservesAbsentValues verifies presence-aware partial updates.
//
// Given a populated configuration and a partial override containing only one
// field, when Config.Update applies it, then absent fields retain their values.
func TestConfigUpdatePreservesAbsentValues(t *testing.T) {
	t.Parallel()

	base := Config{
		HTTP: HTTP{
			Timeout: Timeout{Connect: 30, Idle: 120},
			Proxy:   Proxy{URI: "https://proxy.example", User: "user"},
		},
	}
	override := Partial{
		HTTP: &PartialHTTP{
			Timeout: &PartialHTTPTimeout{Connect: new(int)},
		},
	}

	got := base.Update(override)
	if got.HTTP.Timeout.Connect != 0 {
		t.Errorf("Connect = %d, want 0", got.HTTP.Timeout.Connect)
	}
	if got.HTTP.Timeout.Idle != 120 {
		t.Errorf("Idle = %d, want unchanged value 120", got.HTTP.Timeout.Idle)
	}
	if got.HTTP.Proxy.URI != "https://proxy.example" || got.HTTP.Proxy.User != "user" {
		t.Errorf("Proxy = %+v, want unchanged proxy", got.HTTP.Proxy)
	}
}

// TestConfigUpdateAppliesExplicitZeroValues verifies that explicit zero, false,
// and empty-string values are treated as overrides.
//
// Given non-zero, true, and non-empty base values, when an override explicitly
// supplies zero, false, and empty values, then all three values are replaced.
func TestConfigUpdateAppliesExplicitZeroValues(t *testing.T) {
	t.Parallel()

	base := Config{
		Compatibility: Compatibility{InterpretLegacy: true},
		HTTP: HTTP{
			Timeout: Timeout{Connect: 30},
			Proxy:   Proxy{URI: "https://proxy.example"},
		},
	}
	override := Partial{
		Compatibility: &PartialCompatibility{InterpretLegacyConfigurations: new(bool)},
		HTTP: &PartialHTTP{
			Timeout: &PartialHTTPTimeout{Connect: new(int)},
			Proxy:   &PartialHTTPProxy{URI: new(string)},
		},
	}

	got := base.Update(override)
	if got.Compatibility.InterpretLegacy {
		t.Error("InterpretLegacy = true, want false")
	}
	if got.HTTP.Timeout.Connect != 0 {
		t.Errorf("Connect = %d, want 0", got.HTTP.Timeout.Connect)
	}
	if got.HTTP.Proxy.URI != "" {
		t.Errorf("Proxy.URI = %q, want empty string", got.HTTP.Proxy.URI)
	}
}

// TestConfigUpdateAppliesNestedEndpoints verifies updates across API sections.
//
// Given nested subscription, RPM, ingress, and inventory overrides, when the
// configuration is updated, then each endpoint receives its supplied values.
func TestConfigUpdateAppliesNestedEndpoints(t *testing.T) {
	t.Parallel()

	subscriptionsURI := "subscriptions"
	rpmURI := "rpm"
	ingressURI := "ingress"
	inventoryURI := "inventory"
	override := Partial{
		API: &PartialAPI{
			Subscriptions: &PartialAPISubscriptions{URI: &subscriptionsURI},
			Content:       &PartialAPIContent{RPM: &PartialEndpoint{URI: &rpmURI}},
			Insights: &PartialAPIInsights{
				Ingress:   &PartialEndpoint{URI: &ingressURI},
				Inventory: &PartialEndpoint{URI: &inventoryURI},
			},
		},
	}

	got := (Config{}).Update(override)
	if got.API.Subscriptions.URI != "subscriptions" {
		t.Errorf("Subscriptions.URI = %q", got.API.Subscriptions.URI)
	}
	if got.API.Content.RPM.URI != "rpm" {
		t.Errorf("Content.RPM.URI = %q", got.API.Content.RPM.URI)
	}
	if got.API.Insights.Ingress.URI != "ingress" {
		t.Errorf("Insights.Ingress.URI = %q", got.API.Insights.Ingress.URI)
	}
	if got.API.Insights.Inventory.URI != "inventory" {
		t.Errorf("Insights.Inventory.URI = %q", got.API.Insights.Inventory.URI)
	}
}
