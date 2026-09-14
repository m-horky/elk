package config

type Partial struct {
	Compatibility *PartialCompatibility `toml:"compatibility"`
	HTTP          *PartialHTTP          `toml:"http"`
	API           *PartialAPI           `toml:"api"`
}

type PartialCompatibility struct {
	InterpretLegacyConfigurations *bool `toml:"interpret-legacy-configurations"`
}

type PartialHTTP struct {
	Timeout *PartialHTTPTimeout `toml:"timeout"`
	Proxy   *PartialHTTPProxy   `toml:"proxy"`
}

type PartialHTTPTimeout struct {
	Connect *int `toml:"connect"`
	Idle    *int `toml:"idle"`
}

type PartialHTTPProxy struct {
	URI      *string `toml:"uri"`
	User     *string `toml:"user"`
	Password *string `toml:"password"`
	NoProxy  *string `toml:"no-proxy"`
}

type PartialAPI struct {
	Subscriptions *PartialAPISubscriptions `toml:"subscriptions"`
	Content       *PartialAPIContent       `toml:"content"`
	Insights      *PartialAPIInsights      `toml:"insights"`
}

type PartialAPIContent struct {
	RPM *PartialEndpoint `toml:"rpm"`
}

type PartialAPIInsights struct {
	Ingress   *PartialEndpoint `toml:"ingress"`
	Inventory *PartialEndpoint `toml:"inventory"`
}

type PartialEndpoint struct {
	URI       *string `toml:"uri"`
	TLSVerify *bool   `toml:"tls-verify"`
	CAPath    *string `toml:"ca-path"`
}

type PartialAPISubscriptions struct {
	URI       *string `toml:"uri"`
	TLSVerify *bool   `toml:"tls-verify"`
	CAPath    *string `toml:"ca-path"`
}
