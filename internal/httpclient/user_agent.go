// Package httpclient contains the exploratory shared HTTP transport used by
// the draft endpoint probe. It is not the final service API.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/m-horky/elk/pkg/version"
)

const (
	applicationName    = "elk"
	unknownOSID        = "unknown"
	unknownOSVersionID = "unknown"
)

// The User-Agent implementation in this file is built as a context-aware transport
// decorator. WithCaller stores validated caller metadata in the request context.
// The HTTP client wraps its underlying transport with userAgentTransport, whose
// RoundTrip method reads that metadata, combines it with the Elk & system identity.

var userAgentSafePattern = regexp.MustCompile(`^[A-Za-z0-9._/+*:;~-]+$`)

// userAgentTransport decorates an HTTP transport with User-Agent handling.
type userAgentTransport struct {
	next      http.RoundTripper
	userAgent userAgent
}

// RoundTrip adds the configured user agent and forwards the request.
func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	metadata, _ := req.Context().Value(withTriggeredByKey{}).(triggeredBy)
	userAgent := t.userAgent
	userAgent.caller = metadata

	if req.Header == nil {
		req.Header = make(http.Header)
	}

	req.Header.Set("User-Agent", userAgent.String())

	return t.next.RoundTrip(req) //nolint:wrapcheck
}

// withUserAgent wraps transport with the default User-Agent behavior.
func withUserAgent(transport http.RoundTripper) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}

	return userAgentTransport{next: transport, userAgent: newUserAgent()}
}

// triggeredBy represents the component that triggered this HTTP action.
type triggeredBy struct {
	name    string
	version string
}

// String formats the triggered-by metadata for inclusion in a User-Agent value.
func (c *triggeredBy) String() string {
	if c == nil {
		return ""
	}

	if c.name == "" {
		return c.version
	}

	if c.version == "" {
		return c.name
	}

	return c.name + "/" + c.version
}

// withTriggeredByKey is used to associate a triggeredBy with context.Context.
type withTriggeredByKey struct{}

// WithTriggeredBy associates the name and version of the calling component with ctx.
//
// To have '(triggered-by: name/version)' inserted into the User-Agent header:
//
//	ctx, err = WithTriggeredBy(ctx, "name", "version")
//
// An error is returned when name or version contain non-printable-ASCII characters.
func WithTriggeredBy(ctx context.Context, name, version string) (context.Context, error) {
	if ctx == nil {
		return nil, errors.New("caller context must not be nil")
	}

	if ctx.Value(withTriggeredByKey{}) != nil {
		return nil, errors.New("caller metadata is already set")
	}

	if !isUserAgentSafe(name) {
		return nil, errors.New("caller name contains invalid characters")
	}

	if !isUserAgentSafe(version) {
		return nil, errors.New("caller version contains invalid characters")
	}

	return context.WithValue(ctx, withTriggeredByKey{}, triggeredBy{name: name, version: version}), nil
}

// isUserAgentSafe reports whether value contains only permitted User-Agent token characters.
func isUserAgentSafe(value string) bool {
	return userAgentSafePattern.MatchString(value)
}

// userAgent contains the application, operating-system, and optional triggeredBy
// identification used to construct an HTTP User-Agent header.
type userAgent struct {
	applicationName    string
	applicationVersion string
	osName             string
	osVersion          string
	caller             triggeredBy
}

// String formats the user agent for an HTTP request.
func (u userAgent) String() string {
	name := u.applicationName
	if name == "" {
		name = applicationName
	}

	value := fmt.Sprintf("%s/%s", name, u.applicationVersion)
	if caller := u.caller.String(); caller != "" {
		value += " (triggered-by: " + caller + ")"
	}

	return fmt.Sprintf("%s %s/%s", value, u.osName, u.osVersion)
}

// newUserAgent returns the User-Agent identity used by new HTTP clients.
func newUserAgent() userAgent {
	return userAgent{
		applicationName:    applicationName,
		applicationVersion: version.Version,
		osName:             unknownOSID,
		osVersion:          unknownOSVersionID,
	}
}
