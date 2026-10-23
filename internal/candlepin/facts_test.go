package candlepin

import (
	"testing"

	"github.com/m-horky/elk/pkg/facts"
)

// TestNewFactsDTO verifies the facts dictionary matches Candlepin keys.
//
// Given a fully populated facts object, when it is converted to a Candlepin DTO,
// then every namespace and the derived cpu facts are present.
func TestNewFactsDTO(t *testing.T) {
	t.Parallel()

	systemFacts := &facts.Facts{
		SystemCertificateVersion:    new("3.2"),
		UnameSysname:                new("Linux"),
		UnameNodename:               new("host"),
		UnameRelease:                new("6.0"),
		UnameVersion:                new("#1 SMP"),
		UnameMachine:                new("x86_64"),
		DistributionID:              new("rhel"),
		DistributionName:            new("Red Hat Enterprise Linux"),
		DistributionVersion:         new("10.2"),
		DistributionVersionModifier: new("ga"),
		VirtIsGuest:                 new(false),
		VirtHostType:                new("Not Applicable"),
		LSCPU: map[string]string{
			"cpu(s)":             "8",
			"socket(s)":          "1",
			"core(s)_per_socket": "4",
			"thread(s)_per_core": "2",
			"cpu_op-mode(s)":     "32-bit, 64-bit",
		},
	}

	converted := NewFactsDTO(systemFacts)

	want := canonicalFactsDTO()

	if len(converted) != len(want) {
		t.Errorf("NewFactsDTO() has %d entries, want %d: %v", len(converted), len(want), converted)
	}

	for key, value := range want {
		if converted[key] != value {
			t.Errorf("NewFactsDTO()[%q] = %q, want %q", key, converted[key], value)
		}
	}
}

// canonicalFactsDTO returns the expected Candlepin dictionary for a fully populated facts model.
func canonicalFactsDTO() map[string]string {
	return map[string]string{
		"system.certificate_version":    "3.2",
		"uname.sysname":                 "Linux",
		"uname.nodename":                "host",
		"uname.release":                 "6.0",
		"uname.version":                 "#1 SMP",
		"uname.machine":                 "x86_64",
		"distribution.id":               "rhel",
		"distribution.name":             "Red Hat Enterprise Linux",
		"distribution.version":          "10.2",
		"distribution.version.modifier": "ga",
		"virt.is_guest":                 "false",
		"virt.host_type":                "Not Applicable",
		"lscpu.cpu(s)":                  "8",
		"lscpu.socket(s)":               "1",
		"lscpu.core(s)_per_socket":      "4",
		"lscpu.thread(s)_per_core":      "2",
		"lscpu.cpu_op-mode(s)":          "32-bit, 64-bit",
		"cpu.cpu(s)":                    "8",
		"cpu.cpu_socket(s)":             "1",
		"cpu.core(s)_per_socket":        "4",
		"cpu.thread(s)_per_core":        "2",
	}
}

// TestNewFactsDTOIgnoresNils verifies nil fields do not produce empty keys.
//
// Given only the certificate version, when it is converted to a Candlepin DTO,
// then no other fact keys are emitted.
func TestNewFactsDTOIgnoresNils(t *testing.T) {
	t.Parallel()

	converted := NewFactsDTO(&facts.Facts{SystemCertificateVersion: new("3.2")})

	if len(converted) != 1 {
		t.Fatalf("NewFactsDTO() = %v, want only system.certificate_version", converted)
	}

	if converted["system.certificate_version"] != "3.2" {
		t.Errorf("system.certificate_version = %q, want %q", converted["system.certificate_version"], "3.2")
	}
}

// TestNewFactsDTOHandlesNil verifies unavailable facts are omitted.
//
// Given a nil facts object, when it is converted to a Candlepin DTO,
// then the result is empty rather than panicking.
func TestNewFactsDTOHandlesNil(t *testing.T) {
	t.Parallel()

	if converted := NewFactsDTO(nil); len(converted) != 0 {
		t.Errorf("NewFactsDTO(nil) = %v, want empty map", converted)
	}
}
