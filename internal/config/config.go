package config

import (
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/m-horky/elk/data/etc"
)

var (
	configOnce   sync.Once
	cachedConfig Config
	errConfig    error
)

// Get returns a copy of the configuration compiled into the binary. The
// configuration is decoded at most once and cached for the lifetime of the
// process. If decoding fails, the same error is returned on subsequent calls.
func Get() (Config, error) {
	configOnce.Do(func() {
		cachedConfig, errConfig = loadDefaultConfig()
	})
	if errConfig != nil {
		return Config{}, errConfig
	}
	return cachedConfig, nil
}

func loadDefaultConfig() (Config, error) {
	var p Partial
	if _, err := toml.Decode(etc.ElkDefaultToml, &p); err != nil {
		return Config{}, fmt.Errorf("decode embedded default configuration: %w", err)
	}

	cfg := (Config{}).Update(p)
	return cfg, nil
}

// Update returns a copy of cfg with values present in p applied. Pointer
// fields are used by Partial so that false, zero, and empty string values
// remain distinguishable from fields that were not supplied.
func (cfg Config) Update(p Partial) Config {
	if p.Compatibility != nil && p.Compatibility.InterpretLegacyConfigurations != nil {
		cfg.Compatibility.InterpretLegacy = *p.Compatibility.InterpretLegacyConfigurations
	}

	if p.HTTP != nil {
		if p.HTTP.Timeout != nil {
			if p.HTTP.Timeout.Connect != nil {
				cfg.HTTP.Timeout.Connect = *p.HTTP.Timeout.Connect
			}
			if p.HTTP.Timeout.Idle != nil {
				cfg.HTTP.Timeout.Idle = *p.HTTP.Timeout.Idle
			}
		}
		if p.HTTP.Proxy != nil {
			if p.HTTP.Proxy.URI != nil {
				cfg.HTTP.Proxy.URI = *p.HTTP.Proxy.URI
			}
			if p.HTTP.Proxy.User != nil {
				cfg.HTTP.Proxy.User = *p.HTTP.Proxy.User
			}
			if p.HTTP.Proxy.Password != nil {
				cfg.HTTP.Proxy.Password = *p.HTTP.Proxy.Password
			}
			if p.HTTP.Proxy.NoProxy != nil {
				cfg.HTTP.Proxy.NoProxy = *p.HTTP.Proxy.NoProxy
			}
		}
	}

	if p.API == nil {
		return cfg
	}
	if p.API.Subscriptions != nil {
		cfg.API.Subscriptions = cfg.API.Subscriptions.Update(*p.API.Subscriptions)
	}
	if p.API.Content != nil && p.API.Content.RPM != nil {
		cfg.API.Content.RPM = cfg.API.Content.RPM.Update(*p.API.Content.RPM)
	}
	if p.API.Insights != nil {
		if p.API.Insights.Ingress != nil {
			cfg.API.Insights.Ingress = cfg.API.Insights.Ingress.Update(*p.API.Insights.Ingress)
		}
		if p.API.Insights.Inventory != nil {
			cfg.API.Insights.Inventory = cfg.API.Insights.Inventory.Update(*p.API.Insights.Inventory)
		}
	}
	return cfg
}

// Update returns a copy of s with values present in p applied.
func (s Subscriptions) Update(p PartialAPISubscriptions) Subscriptions {
	if p.URI != nil {
		s.URI = *p.URI
	}
	if p.TLSVerify != nil {
		s.TLSVerify = *p.TLSVerify
	}
	if p.CAPath != nil {
		s.CAPath = *p.CAPath
	}
	return s
}

// Update returns a copy of e with values present in p applied.
func (e Endpoint) Update(p PartialEndpoint) Endpoint {
	if p.URI != nil {
		e.URI = *p.URI
	}
	if p.TLSVerify != nil {
		e.TLSVerify = *p.TLSVerify
	}
	if p.CAPath != nil {
		e.CAPath = *p.CAPath
	}
	return e
}
