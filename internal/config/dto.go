package config

import "time"

type Compatibility struct {
	InterpretLegacy bool `toml:"interpret-legacy-configurations"`
}

type HTTP struct {
	Timeout Timeout `toml:"timeout"`
	Proxy   Proxy   `toml:"proxy"`
}

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

type Timeout struct {
	Connect time.Duration `toml:"connect"`
	Request time.Duration `toml:"request"`
	Idle    time.Duration `toml:"idle"`
}

type Proxy struct {
	URI      string `toml:"uri"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	NoProxy  string `toml:"no-proxy"`
}

type API struct {
	Subscriptions Endpoint `toml:"subscriptions"`
	Content       Content  `toml:"content"`
	Insights      Insights `toml:"insights"`
}

type Content struct {
	RPM Endpoint `toml:"rpm"`
}

type Insights struct {
	Ingress   Endpoint `toml:"ingress"`
	Inventory Endpoint `toml:"inventory"`
}
