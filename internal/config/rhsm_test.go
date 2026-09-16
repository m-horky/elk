package config

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestLegacyPort verifies that legacy ports are parsed only within the TCP port range.
//
// Given numeric and malformed legacy port values,
// when parsing runs, then valid boundary ports are returned and invalid values fail.
func TestLegacyPort(t *testing.T) {
	t.Parallel()

	//nolint:wsl_v5
	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{name: "minimum", value: "1", want: 1},
		{name: "maximum", value: "65535", want: 65535},
		{name: "zero", value: "0", wantErr: true},
		{name: "too large", value: "65536", wantErr: true},
		{name: "not a number", value: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:wsl_v5
			got, err := parseRHSMPort(tt.value)
			//nolint:wsl_v5
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRHSMPort(%q) error = %v, want error: %v", tt.value, err, tt.wantErr)
			}
			//nolint:wsl_v5
			if got != tt.want {
				t.Errorf("parseRHSMPort(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

// TestLegacyBool verifies supported legacy boolean spellings and presence reporting.
//
// Given empty, recognized, and unrecognized boolean values,
// when parsing runs, then values and presence are reported correctly.
func TestLegacyBool(t *testing.T) {
	t.Parallel()

	//nolint:wsl_v5
	tests := []struct {
		value       string
		wantValue   bool
		wantPresent bool
		wantError   bool
	}{
		{value: "", wantValue: false, wantPresent: false},
		{value: " YES ", wantValue: true, wantPresent: true},
		{value: "on", wantValue: true, wantPresent: true},
		{value: "0", wantValue: false, wantPresent: true},
		{value: "OFF", wantValue: false, wantPresent: true},
		{value: "maybe", wantPresent: true, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			value, present, err := parseRHSMBoolean(tt.value)
			//nolint:wsl_v5
			if (err != nil) != tt.wantError {
				t.Fatalf("parseRHSMBoolean(%q) error = %v, want error: %v", tt.value, err, tt.wantError)
			}
			//nolint:wsl_v5
			if value != tt.wantValue || present != tt.wantPresent {
				t.Errorf("parseRHSMBoolean(%q) = (%v, %v), want (%v, %v)", tt.value, value, present, tt.wantValue, tt.wantPresent)
			}
		})
	}
}

// TestValidateLegacyHost verifies that URL-ambiguous host values are rejected.
//
// Given host values with whitespace, URL delimiters, or ambiguous colons,
// when validation runs, then only safe hosts are accepted.
func TestValidateLegacyHost(t *testing.T) {
	t.Parallel()

	//nolint:wsl_v5
	tests := []struct {
		value   string
		wantErr bool
	}{
		{value: "example.com"},
		{value: "127.0.0.1"},
		{value: "2001:db8::1"},
		{value: "", wantErr: true},
		{value: "host name", wantErr: true},
		{value: "https://example.com", wantErr: true},
		{value: "host:port", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			//nolint:wsl_v5
			got, err := parseRHSMHost(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRHSMHost(%q) error = %v, want error: %v", tt.value, err, tt.wantErr)
			}

			if err == nil && got != tt.value {
				t.Errorf("parseRHSMHost(%q) = %q, want %q", tt.value, got, tt.value)
			}
		})
	}
}

// TestValidateLegacyURL verifies that only usable HTTPS URLs are accepted.
//
// Given HTTPS and non-HTTPS URL variants,
// when validation runs, then only credential-free URLs without queries or fragments are accepted.
func TestValidateLegacyURL(t *testing.T) {
	t.Parallel()

	//nolint:wsl_v5
	tests := []struct {
		value   string
		wantErr bool
	}{
		{value: "https://example.com/path"},
		{value: "http://example.com", wantErr: true},
		{value: "https://user:pass@example.com", wantErr: true}, //nolint:gosec // Credentials are the validation fixture.
		{value: "https://example.com/path?query", wantErr: true},
		{value: "https://example.com/path#fragment", wantErr: true},
		{value: "https:///path", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			//nolint:wsl_v5
			got, err := parseRHSMURL(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRHSMURL(%q) error = %v, want error: %v", tt.value, err, tt.wantErr)
			}

			if err == nil && got != tt.value {
				t.Errorf("parseRHSMURL(%q) = %q, want %q", tt.value, got, tt.value)
			}
		})
	}
}

// TestTranslateLegacySubscriptions verifies subscription endpoint construction and TLS translation.
//
// Given a legacy server section,
// when it is translated, then hostname, port, prefix, HTTPS scheme, and insecure TLS semantics are preserved.
func TestTranslateLegacySubscriptions(t *testing.T) {
	t.Parallel()

	server := rhsmServer{Hostname: " subscription.example.com ", Port: "8443", Prefix: "/subscription", Insecure: "true"}
	//nolint:wsl_v5
	got, present, err := mapRHSMCandlepin(server, true, true)
	//nolint:wsl_v5
	if err != nil || !present {
		t.Fatalf("mapRHSMServerToSubscriptions() = (%+v, %v, %v), want present without error", got, present, err)
	}
	//nolint:wsl_v5
	if got.URI == nil || *got.URI != "https://subscription.example.com:8443/subscription" {
		t.Errorf("URI = %v, want subscription URL", got.URI)
	}
	//nolint:wsl_v5
	if got.TLSVerify == nil || *got.TLSVerify {
		t.Errorf("TLSVerify = %v, want false", got.TLSVerify)
	}
}

// TestTranslateLegacyInsights verifies Insights endpoint derivation for staging and Satellite hosts.
//
// Given legacy server hostnames,
// when Insights endpoints are translated, then staging uses the production console API,
// Satellite uses its local Insights paths, and production RHSM leaves endpoints absent.
func TestTranslateLegacyInsights(t *testing.T) {
	t.Parallel()

	//nolint:wsl_v5
	tests := []struct {
		name          string
		hostname      string
		port          string
		caPath        string
		wantIngress   string
		wantInventory string
		wantPresent   bool
	}{
		{
			name:          "staging",
			hostname:      "subscription.rhsm.stage.redhat.com",
			wantIngress:   "https://cert.console.stage.redhat.com/api/ingress/v1",
			wantInventory: "https://cert.console.stage.redhat.com/api/inventory/v1",
			wantPresent:   true,
		},
		{
			name:          "satellite",
			hostname:      "satellite.example.com",
			port:          "8443",
			caPath:        " /etc/pki/satellite-ca.pem ",
			wantIngress:   "https://satellite.example.com:8443/redhat_access/r/insights/platform/ingress/v1",
			wantInventory: "https://satellite.example.com:8443/redhat_access/r/insights/platform/inventory/v1",
			wantPresent:   true,
		},
		{name: "production rhsm", hostname: "subscription.rhsm.redhat.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapRHSMInsights(rhsmServer{Hostname: tt.hostname, Port: tt.port}, tt.caPath, false, false)
			if err != nil {
				t.Fatalf("mapRHSMInsights() error = %v", err)
			}

			present := got != nil

			if present != tt.wantPresent {
				t.Fatalf("mapRHSMServerToInsights() present = %v, want %v", present, tt.wantPresent)
			}

			if !present {
				return
			}

			if got == nil || got.Ingress == nil || got.Inventory == nil ||
				got.Ingress.URI == nil || *got.Ingress.URI != tt.wantIngress ||
				got.Inventory.URI == nil || *got.Inventory.URI != tt.wantInventory {
				t.Errorf("mapRHSMServerToInsights() = %+v, want ingress %q and inventory %q", got, tt.wantIngress, tt.wantInventory)
			}

			if tt.caPath != "" {
				if got.Ingress.CAPath == nil || *got.Ingress.CAPath != "/etc/pki/satellite-ca.pem" ||
					got.Inventory.CAPath == nil || *got.Inventory.CAPath != "/etc/pki/satellite-ca.pem" {
					t.Errorf("CA paths = (%v, %v), want %q", got.Ingress.CAPath, got.Inventory.CAPath, "/etc/pki/satellite-ca.pem")
				}
			}
		})
	}
}

// TestTranslateLegacySubscriptionsValidation verifies subscription validation errors identify their source keys.
//
// Given invalid server fields,
// when subscription translation runs, then it returns an error containing the corresponding legacy key.
func TestTranslateLegacySubscriptionsValidation(t *testing.T) {
	t.Parallel()

	//nolint:wsl_v5
	tests := []struct {
		name   string
		server rhsmServer
		key    string
	}{
		{name: "missing hostname", server: rhsmServer{Port: "443"}, key: "server.hostname"},
		{name: "missing port", server: rhsmServer{Hostname: "example.com"}, key: "server.port"},
		{name: "invalid host", server: rhsmServer{Hostname: "bad host", Port: "443"}, key: "server.hostname"},
		{name: "invalid port", server: rhsmServer{Hostname: "example.com", Port: "0"}, key: "server.port"},
		//nolint:lll,wsl_v5
		{name: "invalid prefix", server: rhsmServer{Hostname: "example.com", Port: "443", Prefix: "subscription?x"}, key: "server.prefix"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := mapRHSMCandlepin(tt.server, false, false)
			//nolint:wsl_v5
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("error = %v, want key %q", err, tt.key)
			}
		})
	}
}

// TestTranslateLegacyRepository verifies repository URL, CA path, and TLS translation.
//
// Given a legacy repository section,
// when it is translated, then baseurl, repo_ca_cert, and insecure TLS semantics populate the RPM host.
func TestTranslateLegacyRepository(t *testing.T) {
	t.Parallel()

	//nolint:lll,wsl_v5
	got, present, err := mapRHSMContent(rhsmRHSM{BaseURL: " https://content.example.com/pulp ", RepoCACert: " /etc/pki/ca.pem "}, true, true) //nolint:wsl_v5
	//nolint:wsl_v5
	if err != nil || !present {
		t.Fatalf("mapRHSMRepository() = (%+v, %v, %v), want present without error", got, present, err)
	}
	//nolint:lll,wsl_v5
	if got.URI == nil || *got.URI != "https://content.example.com/pulp" || got.CAPath == nil || *got.CAPath != "/etc/pki/ca.pem" { //nolint:wsl_v5
		t.Errorf("repository = %+v, want trimmed URL and CA path", got)
	}
	//nolint:wsl_v5
	if got.TLSVerify == nil || *got.TLSVerify {
		t.Errorf("TLSVerify = %v, want false", got.TLSVerify)
	}
}

// TestTranslateLegacyRepositoryValidation verifies invalid repository URLs are attributed to baseurl.
//
// Given a non-HTTPS repository URL, when repository translation runs, then the error identifies rhsm.baseurl.
func TestTranslateLegacyRepositoryValidation(t *testing.T) {
	t.Parallel()

	_, _, err := mapRHSMContent(rhsmRHSM{BaseURL: "http://content.example.com"}, false, false)
	//nolint:wsl_v5
	if err == nil || !strings.Contains(err.Error(), "rhsm.baseurl") {
		t.Fatalf("error = %v, want rhsm.baseurl", err)
	}
}

// TestTranslateLegacyProxy verifies proxy endpoint and optional credential translation.
//
// Given a complete legacy proxy section,
// when it is translated, then hostname, port, credentials, and no_proxy populate the HTTP proxy.
func TestTranslateLegacyProxy(t *testing.T) {
	t.Parallel()

	//nolint:lll,wsl_v5
	got, present, err := mapRHSMProxy(rhsmProxy{Hostname: "proxy.example.com", Port: "3128", User: " user ", Password: "secret", NoProxy: " localhost,example.com "}) //nolint:wsl_v5
	//nolint:wsl_v5
	if err != nil || !present {
		t.Fatalf("mapRHSMProxy() = (%+v, %v, %v), want present without error", got, present, err)
	}
	//nolint:lll,wsl_v5
	if got.URI == nil || *got.URI != "https://proxy.example.com:3128" || got.User == nil || *got.User != "user" || got.Password == nil || *got.Password != "secret" || got.NoProxy == nil || *got.NoProxy != "localhost,example.com" { //nolint:wsl_v5
		t.Errorf("proxy = %+v, want translated fields", got)
	}
}

// TestTranslateLegacyProxyValidation verifies proxy validation errors identify the relevant legacy key.
//
// Given incomplete or invalid proxy endpoint fields,
// when proxy translation runs, then the error contains the relevant proxy key.
func TestTranslateLegacyProxyValidation(t *testing.T) {
	t.Parallel()

	//nolint:wsl_v5
	tests := []struct {
		name  string
		proxy rhsmProxy
		key   string
	}{
		{name: "missing hostname", proxy: rhsmProxy{Port: "3128"}, key: "proxy.proxy_hostname"},
		{name: "missing port", proxy: rhsmProxy{Hostname: "proxy.example.com"}, key: "proxy.proxy_port"},
		{name: "invalid host", proxy: rhsmProxy{Hostname: "bad host", Port: "3128"}, key: "proxy.proxy_hostname"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := mapRHSMProxy(tt.proxy)
			//nolint:wsl_v5
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("error = %v, want key %q", err, tt.key)
			}
		})
	}
}

// TestTranslateLegacyEmptySections verifies empty DTO sections do not create partial fields.
//
// Given an empty legacy configuration, when it is translated, then the returned partial configuration is empty.
func TestTranslateLegacyEmptySections(t *testing.T) {
	t.Parallel()

	//nolint:wsl_v5
	got, err := mapRHSMToPartial(rhsmConfiguration{})
	//nolint:wsl_v5
	if err != nil {
		t.Fatalf("mapRHSMToPartial() error = %v", err)
	}
	//nolint:wsl_v5
	if !reflect.DeepEqual(got, Partial{}) {
		t.Fatalf("mapRHSMToPartial() = %+v, want empty partial", got)
	}
}

// TestLoadRHSM verifies DTO mapping and complete legacy translation through the filesystem abstraction.
//
// Given a legacy rhsm.conf with server, repository, and proxy values,
// when LoadRHSM reads it, then all supported values are translated into Partial.
func TestLoadRHSM(t *testing.T) {
	t.Parallel()

	path := "/etc/rhsm.conf"
	filesystem := configTestFS{files: map[string][]byte{path: []byte(`[server]
hostname = subscription.example.com
port = 443
prefix = /subscription
insecure = yes

[rhsm]
baseurl = https://content.example.com/pulp
repo_ca_cert = /etc/pki/ca.pem

[proxy]
proxy_hostname = proxy.example.com
proxy_port = 3128
proxy_user = user
proxy_password = password
no_proxy = localhost
`)}}
	//nolint:wsl_v5
	got, err := LoadRHSM(filesystem, path)
	//nolint:wsl_v5
	if err != nil {
		t.Fatalf("LoadRHSM() error = %v", err)
	}
	//nolint:lll,wsl_v5
	if got.API == nil || got.API.Subscriptions == nil || got.API.Content == nil || got.API.Content.RPM == nil || got.HTTP == nil || got.HTTP.Proxy == nil { //nolint:wsl_v5
		t.Fatalf("LoadRHSM() = %+v, want all translated sections", got)
	}
	//nolint:lll,wsl_v5
	if *got.API.Subscriptions.URI != "https://subscription.example.com:443/subscription" || *got.API.Subscriptions.TLSVerify { //nolint:wsl_v5
		t.Errorf("subscriptions = %+v, want URL and TLSVerify false", got.API.Subscriptions)
	}
	//nolint:wsl_v5
	if *got.API.Content.RPM.URI != "https://content.example.com/pulp" || *got.API.Content.RPM.CAPath != "/etc/pki/ca.pem" {
		t.Errorf("RPM = %+v, want URL and CA path", got.API.Content.RPM)
	}
	//nolint:wsl_v5
	if *got.API.Subscriptions.CAPath != "/etc/pki/ca.pem" {
		t.Errorf("subscription CAPath = %q, want repository CA path", *got.API.Subscriptions.CAPath)
	}
}

// TestLoadRHSMErrors verifies missing files, read failures, and malformed INI errors.
//
// Given representative filesystem and parsing failures,
// when LoadRHSM runs, then missing files are ignored and other errors are wrapped safely with the path.
func TestLoadRHSMErrors(t *testing.T) {
	t.Parallel()

	path := "/etc/rhsm.conf"
	//nolint:wsl_v5
	tests := []struct {
		name       string
		filesystem rhsmTestFS
		wantErr    bool
		wantText   string
		forbidden  string
	}{
		{name: "missing", filesystem: rhsmTestFS{configTestFS: configTestFS{}}, wantText: ""},
		{name: "read failure", filesystem: rhsmTestFS{
			configTestFS: configTestFS{}, readErr: errRHSMRead,
		}, wantErr: true, wantText: path},
		{name: "invalid INI", filesystem: rhsmTestFS{configTestFS: configTestFS{
			files: map[string][]byte{path: []byte("[server\nhostname = parser detail")},
		}}, wantErr: true, wantText: "invalid INI syntax", forbidden: "parser detail"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:wsl_v5
			got, err := LoadRHSM(tt.filesystem, path)
			//nolint:wsl_v5
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadRHSM() = (%+v, %v), want error: %v", got, err, tt.wantErr)
			}
			//nolint:wsl_v5
			if tt.wantText != "" && !strings.Contains(err.Error(), tt.wantText) {
				t.Errorf("error = %v, want %q", err, tt.wantText)
			}
			//nolint:wsl_v5
			if tt.forbidden != "" && strings.Contains(err.Error(), tt.forbidden) {
				t.Errorf("error = %v, must not expose parser contents %q", err, tt.forbidden)
			}
			//nolint:wsl_v5
			if !tt.wantErr && !reflect.DeepEqual(got, Partial{}) {
				t.Errorf("missing file result = %+v, want empty partial", got)
			}
		})
	}
}

var errRHSMRead = errors.New("permission denied") //nolint:gochecknoglobals,err113 // Test filesystem failure.

type rhsmTestFS struct {
	configTestFS

	readErr error
}

func (f rhsmTestFS) Read(path string) ([]byte, error) {
	//nolint:wsl_v5
	if f.readErr != nil {
		return nil, f.readErr
	}

	return f.configTestFS.Read(path)
}
