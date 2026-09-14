package config

type (
	Compatibility struct {
		InterpretLegacy bool `toml:"interpret-legacy-configurations"`
	}
	HTTP struct {
		Timeout Timeout `toml:"timeout"`
		Proxy   Proxy   `toml:"proxy"`
	}
)

type Config struct {
	Compatibility Compatibility `toml:"compatibility"`
	HTTP          HTTP          `toml:"http"`
	API           API           `toml:"api"`
}

type Endpoint struct {
	URI       string `toml:"uri"`
	TLSVerify bool   `toml:"tls-verify"`
	CAPath    string `toml:"ca-path"`
}

type (
	Timeout struct {
		Connect int `toml:"connect"`
		Idle    int `toml:"idle"`
	}
	Proxy struct {
		URI      string `toml:"uri"`
		User     string `toml:"user"`
		Password string `toml:"password"`
		NoProxy  string `toml:"no-proxy"`
	}
	API struct {
		Subscriptions Subscriptions `toml:"subscriptions"`
		Content       Content       `toml:"content"`
		Insights      Insights      `toml:"insights"`
	}
)

type Subscriptions struct {
	URI       string `toml:"uri"`
	TLSVerify bool   `toml:"tls-verify"`
	CAPath    string `toml:"ca-path"`
}

type (
	Content struct {
		RPM Endpoint `toml:"rpm"`
	}
	Insights struct {
		Ingress   Endpoint `toml:"ingress"`
		Inventory Endpoint `toml:"inventory"`
	}
)
