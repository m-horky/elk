package httpclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Given caller metadata, when formatting a user agent, then it includes the caller details.
func TestUserAgentValue(t *testing.T) {
	newCaller := func(name, callerVersion string) *triggeredBy {
		return &triggeredBy{name: name, version: callerVersion}
	}

	tests := []struct {
		name        string
		triggeredBy *triggeredBy
		want        string
	}{
		{name: "without caller", want: "elk/dev rhel/10.2"},
		{
			name: "name and version", triggeredBy: newCaller("cockpit", "312"),
			want: "elk/dev (triggered-by: cockpit/312) rhel/10.2",
		},
		{name: "name only", triggeredBy: newCaller("cockpit", ""), want: "elk/dev (triggered-by: cockpit) rhel/10.2"},
		{name: "version only", triggeredBy: newCaller("", "312"), want: "elk/dev (triggered-by: 312) rhel/10.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := userAgent{
				applicationName:    applicationName,
				applicationVersion: "dev",
				osName:             "rhel",
				osVersion:          "10.2",
			}
			if tt.triggeredBy != nil {
				u.caller = *tt.triggeredBy
			}

			if got := u.String(); got != tt.want {
				t.Fatalf("userAgent.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Given a caller-supplied User-Agent header, when the request is sent, then the transport overwrites it.
func TestUserAgentTransportOverwritesCallerHeader(t *testing.T) {
	var got string

	transport := userAgentTransport{
		next: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			got = req.Header.Get("User-Agent")

			return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
		}),
		userAgent: userAgent{applicationVersion: "dev", osName: "unknown", osVersion: "unknown"},
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("User-Agent", "caller-supplied")

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}

	if got != "elk/dev unknown/unknown" {
		t.Fatalf("User-Agent = %q", got)
	}
}

// TestWithCallerRejectsInvalidCharacters validates user-provided input.
//
// Given invalid caller input, when WithTriggeredBy is called, then an error is returned.
func TestWithCallerRejectsInvalidCharacters(t *testing.T) {
	for _, tt := range []struct {
		name    string
		version string
	}{
		{name: "bad name", version: "1"},
		{name: "service", version: "1\\2"},
		{name: "empty value", version: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := WithTriggeredBy(context.Background(), tt.name, tt.version); err == nil {
				t.Fatal("WithCaller() accepted invalid metadata")
			}
		})
	}
}

// TestWithCallerAcceptsExpandedTokenCharacters verifies the expanded safe-character set.
//
// Given caller metadata containing permitted token punctuation, when WithTriggeredBy is called,
// then it returns a context without an error.
func TestWithCallerAcceptsExpandedTokenCharacters(t *testing.T) {
	if _, err := WithTriggeredBy(context.Background(), "service*:;~", "1.0+build"); err != nil {
		t.Fatalf("WithCaller() rejected permitted metadata: %v", err)
	}
}

// Given a nil context, when WithTriggeredBy is called, then it returns an error.
func TestWithCallerRejectsNilContext(t *testing.T) {
	var ctx context.Context
	if _, err := WithTriggeredBy(ctx, "service", "1"); err == nil {
		t.Fatal("WithCaller() accepted a nil context")
	}
}

// Given caller metadata is already set, when WithTriggeredBy is called again, then it returns an error.
func TestWithCallerRejectsExistingMetadata(t *testing.T) {
	ctx, err := WithTriggeredBy(context.Background(), "first", "1")
	if err != nil {
		t.Fatalf("first WithCaller() returned an error: %v", err)
	}

	if _, err := WithTriggeredBy(ctx, "second", "2"); err == nil {
		t.Fatal("WithCaller() accepted metadata when caller metadata was already set")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip invokes the wrapped round-trip function.
func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
