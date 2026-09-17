package config

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"

	elkfs "github.com/m-horky/elk/internal/fs"
	"gopkg.in/ini.v1"
)

// rhsmConfiguration is the DTO for the supported portions of rhsm.conf.
// It is deliberately private because rhsm.conf is an input format, not part of
// Elk's public configuration model.
type rhsmConfiguration struct {
	Server rhsmServer `ini:"server"`
	RHSM   rhsmRHSM   `ini:"rhsm"`
	Proxy  rhsmProxy  `ini:"proxy"`
}

type rhsmServer struct {
	Hostname string `ini:"hostname"`
	Port     string `ini:"port"`
	Prefix   string `ini:"prefix"`
	Insecure string `ini:"insecure"`
}

type rhsmRHSM struct {
	BaseURL    string `ini:"baseurl"`
	RepoCACert string `ini:"repo_ca_cert"`
}

type rhsmProxy struct {
	Hostname string `ini:"proxy_hostname"`
	Port     string `ini:"proxy_port"`
	User     string `ini:"proxy_user"`
	Password string `ini:"proxy_password"`
	NoProxy  string `ini:"no_proxy"`
}

// LoadRHSM reads rhsm.conf into its DTO and translates the supported settings
// into Elk's partial configuration format.
func LoadRHSM(filesystem elkfs.FS, path string) (Partial, error) {
	data, err := filesystem.Read(path)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return Partial{}, nil
		}

		return Partial{}, fmt.Errorf("cannot read legacy configuration %s: %w", path, err)
	}

	slog.Debug("legacy configuration file read", "path", path, "bytes", len(data))
	file, err := ini.Load(data)
	if err != nil {
		return Partial{}, fmt.Errorf("cannot parse legacy configuration %s: %w", path, errors.New("invalid INI syntax"))
	}

	var legacy rhsmConfiguration
	if err := file.MapTo(&legacy); err != nil {
		return Partial{}, fmt.Errorf("cannot map legacy configuration %s: %w", path, err)
	}

	slog.Debug("legacy configuration parsed", "path", path)
	return mapRHSMToPartial(legacy)
}

// mapRHSMToPartial maps supported rhsm.conf sections to an Elk Partial configuration.
func mapRHSMToPartial(legacy rhsmConfiguration) (Partial, error) {
	insecure, insecurePresent, err := parseRHSMBoolean(legacy.Server.Insecure)
	if err != nil {
		return Partial{}, rhsmError("server.insecure", err)
	}

	subscription, hasSubscription, err := mapRHSMCandlepin(
		legacy.Server, insecure, insecurePresent,
	)
	if err != nil {
		return Partial{}, err
	}

	repository, hasRepository, err := mapRHSMContent(
		legacy.RHSM, insecure, insecurePresent,
	)
	if err != nil {
		return Partial{}, err
	}

	proxy, hasProxy, err := mapRHSMProxy(legacy.Proxy)
	if err != nil {
		return Partial{}, err
	}

	insights, err := mapRHSMInsights(legacy.Server, legacy.RHSM.RepoCACert, insecure, insecurePresent)
	if err != nil {
		return Partial{}, err
	}

	var p Partial
	if hasSubscription || hasRepository || insights != nil {
		p.API = new(PartialAPI)
		if hasSubscription {
			p.API.Subscriptions = new(subscription)
		}

		if hasRepository {
			p.API.Content = &PartialAPIContent{RPM: new(repository)}
		}

		if insights != nil {
			p.API.Insights = insights
		}
	}

	if hasRepository && legacy.RHSM.RepoCACert != "" && hasSubscription {
		p.API.Subscriptions.CAPath = new(legacy.RHSM.RepoCACert)
	}

	if hasProxy {
		p.HTTP = &PartialHTTP{Proxy: new(proxy)}
	}

	return p, nil
}

// mapRHSMCandlepin maps the rhsm.conf [server] section to the
// subscriptions API endpoint configuration.
func mapRHSMCandlepin(
	server rhsmServer,
	insecure bool,
	insecurePresent bool,
) (PartialAPIHost, bool, error) {
	host := strings.TrimSpace(server.Hostname)
	if host == "" {
		if server.Port != "" || server.Prefix != "" || server.Insecure != "" {
			return PartialAPIHost{}, false, rhsmError("server.hostname", errors.New("missing hostname"))
		}

		return PartialAPIHost{}, false, nil
	}

	if _, err := parseRHSMHost(host); err != nil {
		return PartialAPIHost{}, false, rhsmError("server.hostname", err)
	}

	port, err := parseRHSMPort(server.Port)
	if err != nil {
		return PartialAPIHost{}, false, rhsmError("server.port", err)
	}

	prefix := strings.TrimSpace(server.Prefix)
	if prefix != "" && (prefix[0] != '/' || strings.ContainsAny(prefix, "?#")) {
		return PartialAPIHost{}, false, rhsmError("server.prefix", errors.New("invalid path prefix"))
	}

	uri := "https://" + net.JoinHostPort(host, strconv.Itoa(port)) + prefix
	if _, err := parseRHSMURL(uri); err != nil {
		return PartialAPIHost{}, false, rhsmError("server", err)
	}

	result := &PartialAPIHost{URI: new(uri)}
	if insecurePresent {
		result.TLSVerify = new(!insecure)
	}

	slog.Debug("mapped legacy subscription configuration", "host", host, "port", port)
	return *result, true, nil
}

type rhsmTarget int

const (
	rhsmTargetRHSM rhsmTarget = iota
	rhsmTargetStage
	rhsmTargetSatellite
)

func classifyRHSMTarget(host string) rhsmTarget {
	hostname := strings.ToLower(strings.TrimSpace(host))
	switch {
	case strings.HasSuffix(hostname, ".rhsm.stage.redhat.com"):
		return rhsmTargetStage
	case strings.HasSuffix(hostname, ".rhsm.redhat.com"):
		return rhsmTargetRHSM
	default:
		return rhsmTargetSatellite
	}
}

// mapRHSMInsights maps the rhsm.conf [server] section to Insights endpoint
// configuration according to the target system type.
func mapRHSMInsights(
	server rhsmServer,
	caPath string,
	insecure bool,
	insecurePresent bool,
) (*PartialAPIInsights, error) {
	host := strings.TrimSpace(server.Hostname)
	if host == "" {
		return nil, nil //nolint:nilnil // No server section means no Insights endpoint.
	}

	if _, err := parseRHSMHost(host); err != nil {
		return nil, rhsmError("server.hostname", err)
	}

	var result *PartialAPIInsights

	var err error

	switch classifyRHSMTarget(host) {
	case rhsmTargetStage:
		result, _ = mapRHSMStageInsights()
	case rhsmTargetSatellite:
		result, err = mapRHSMSatelliteInsights(server, caPath)
		if err != nil {
			return nil, err
		}
	case rhsmTargetRHSM:
		return nil, nil //nolint:nilnil // Production RHSM has no Insights endpoint.
	}

	if insecurePresent {
		result.Ingress.TLSVerify = new(!insecure)
		result.Inventory.TLSVerify = new(!insecure)
	}

	return result, nil
}

func mapRHSMStageInsights() (*PartialAPIInsights, error) {
	base := "https://cert.console.stage.redhat.com/api"

	return &PartialAPIInsights{
		Ingress:   &PartialAPIHost{URI: new(base + "/ingress/v1")},
		Inventory: &PartialAPIHost{URI: new(base + "/inventory/v1")},
	}, nil
}

func mapRHSMSatelliteInsights(server rhsmServer, caPath string) (*PartialAPIInsights, error) {
	host := strings.TrimSpace(server.Hostname)
	if host == "" {
		return nil, rhsmError("server.hostname", errors.New("missing hostname"))
	}

	if _, err := parseRHSMHost(host); err != nil {
		return nil, rhsmError("server.hostname", err)
	}

	port, err := parseRHSMPort(server.Port)
	if err != nil {
		return nil, rhsmError("server.port", err)
	}

	base := "https://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/redhat_access/r/insights/platform"

	result := &PartialAPIInsights{
		Ingress:   &PartialAPIHost{URI: new(base + "/ingress/v1")},
		Inventory: &PartialAPIHost{URI: new(base + "/inventory/v1")},
	}
	if ca := strings.TrimSpace(caPath); ca != "" {
		result.Ingress.CAPath = new(ca)
		result.Inventory.CAPath = new(ca)
	}

	return result, nil
}

// mapRHSMContent maps the rhsm.conf [rhsm] section to the content API
// endpoint configuration.
func mapRHSMContent(
	rhsm rhsmRHSM,
	insecure bool,
	insecurePresent bool,
) (PartialAPIHost, bool, error) {
	baseURL := strings.TrimSpace(rhsm.BaseURL)
	if baseURL == "" {
		return PartialAPIHost{}, false, nil
	}

	if _, err := parseRHSMURL(baseURL); err != nil {
		return PartialAPIHost{}, false, rhsmError("rhsm.baseurl", err)
	}

	result := &PartialAPIHost{URI: new(baseURL)}
	if insecurePresent {
		result.TLSVerify = new(!insecure)
	}

	if ca := strings.TrimSpace(rhsm.RepoCACert); ca != "" {
		result.CAPath = new(ca)
	}

	return *result, true, nil
}

// mapRHSMProxy maps the rhsm.conf [proxy] section to HTTP proxy configuration.
func mapRHSMProxy(proxy rhsmProxy) (PartialHTTPProxy, bool, error) {
	host := strings.TrimSpace(proxy.Hostname)
	portValue := strings.TrimSpace(proxy.Port)
	result := new(PartialHTTPProxy)

	if host != "" || portValue != "" {
		if host == "" {
			return PartialHTTPProxy{}, false, rhsmError("proxy.proxy_hostname", errors.New("missing hostname"))
		}

		if _, err := parseRHSMHost(host); err != nil {
			return PartialHTTPProxy{}, false, rhsmError("proxy.proxy_hostname", err)
		}

		port, err := parseRHSMPort(portValue)
		if err != nil {
			return PartialHTTPProxy{}, false, rhsmError("proxy.proxy_port", err)
		}

		uri := "https://" + net.JoinHostPort(host, strconv.Itoa(port))
		if _, err := parseRHSMURL(uri); err != nil {
			return PartialHTTPProxy{}, false, rhsmError("proxy", err)
		}

		result.URI = new(uri)
	}

	if value := strings.TrimSpace(proxy.User); value != "" {
		result.User = new(value)
	}

	if proxy.Password != "" {
		result.Password = new(proxy.Password)
	}

	if value := strings.TrimSpace(proxy.NoProxy); value != "" {
		result.NoProxy = new(value)
	}

	if result.URI == nil && result.User == nil && result.Password == nil && result.NoProxy == nil {
		return PartialHTTPProxy{}, false, nil
	}

	return *result, true, nil
}

// rhsmError annotates a validation error with the source file and legacy key.
func rhsmError(key string, err error) error {
	return fmt.Errorf("error reading legacy %s: %w", key, err)
}

// parseRHSMPort parses a legacy port and rejects values outside the TCP port range.
func parseRHSMPort(value string) (int, error) {
	value = strings.TrimSpace(value)
	port, err := strconv.Atoi(value)

	if err != nil || port < 1 || port > 65535 {
		return 0, errors.New("invalid port")
	}

	return port, nil
}

// parseRHSMBoolean parses a rhsm.conf boolean value.
//
// Returns parsed value, whether it was set, and optionally an error.
func parseRHSMBoolean(value string) (bool, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, false, nil
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true, true, nil
	case "0", "false", "no", "off":
		return false, true, nil
	default:
		return false, true, errors.New("invalid boolean")
	}
}

// parseRHSMHost accepts host values that can be safely used in a URL.
func parseRHSMHost(value string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" || strings.ContainsAny(value, " /?#") {
		return "", errors.New("invalid host")
	}

	if ip := net.ParseIP(value); ip == nil && strings.Contains(value, ":") {
		return "", errors.New("invalid host")
	}

	return value, nil
}

// parseRHSMURL accepts HTTPS URLs without credentials, queries, or fragments.
func parseRHSMURL(value string) (string, error) {
	value = strings.TrimSpace(value)

	u, err := url.ParseRequestURI(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil ||
		strings.Contains(value, "#") || u.Fragment != "" || u.RawQuery != "" {
		return "", errors.New("invalid HTTPS URL")
	}

	return value, nil
}
