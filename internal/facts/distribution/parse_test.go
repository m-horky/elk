package distribution

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestParseRHELFixture verifies that a normal RHEL os-release is mapped.
//
// Given the RHEL os-release fixture, when parse parses its contents, then all
// distribution fields are populated from their os-release keys.
func TestParseRHELFixture(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/rhel-os-release")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	want := Distribution{
		ID:              new("rhel"),
		Name:            new("Red Hat Enterprise Linux"),
		Version:         new("9.4"),
		VersionModifier: new("ga"),
	}

	got, err := parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Parse() mismatch (-want +got):\n%s", diff)
	}
}

// TestParseQuotedFixture verifies quoted, unquoted, commented, and escaped values.
//
// Given an os-release fixture with comments, single and double quotes, and
// escaped characters, when parse parses its contents, then the unescaped
// values are mapped to distribution fields.
func TestParseQuotedFixture(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/quoted-os-release")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	want := Distribution{
		ID:              new("fedora"),
		Name:            new(`Fedora Linux "Workstation"`),
		Version:         new("40"),
		VersionModifier: new("beta"),
	}

	got, err := parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Parse() mismatch (-want +got):\n%s", diff)
	}
}

// TestParseMissingFields verifies that absent keys produce nil fields.
//
// Given os-release data containing only an ID key, when parse parses it, then
// the name, version, and version modifier are nil.
func TestParseMissingFields(t *testing.T) {
	t.Parallel()

	want := Distribution{ID: new("rhel")}

	got, err := parse([]byte("ID=rhel\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Parse() mismatch (-want +got):\n%s", diff)
	}
}

// TestParseMissingCPE verifies that os-release data without CPE_NAME has no modifier.
//
// Given os-release data without a CPE_NAME key, when parse parses it, then
// VersionModifier is nil.
func TestParseMissingCPE(t *testing.T) {
	t.Parallel()

	got, err := parse([]byte("ID=rhel\nNAME=\"Red Hat Enterprise Linux\"\nVERSION_ID=9.4\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got.VersionModifier != nil {
		t.Fatalf("VersionModifier = %q, want nil", *got.VersionModifier)
	}
}

// TestParseVersionModifier verifies that valid CPE names produce a modifier.
//
// Given os-release data with a valid operating system CPE_NAME value, when
// parse parses it, then it returns the lower-cased and unescaped component 5
// modifier.
func TestParseVersionModifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cpe  string
		want string
	}{
		{
			name: "ga modifier",
			cpe:  "cpe:/o:redhat:enterprise_linux:9:ga",
			want: "ga",
		},
		{
			name: "escaped colon is unescaped and lower-cased",
			cpe:  `cpe:/o:vendor:product:9:Modifier\:Suffix`,
			want: "modifier:suffix",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parse([]byte("CPE_NAME=" + test.cpe + "\n"))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if got.VersionModifier == nil {
				t.Fatalf("VersionModifier = nil, want %q", test.want)
			}

			if *got.VersionModifier != test.want {
				t.Fatalf("VersionModifier = %q, want %q", *got.VersionModifier, test.want)
			}
		})
	}
}

// TestParseVersionModifierIgnoresInvalidCPE verifies that bad CPE data yields no modifier.
//
// Given os-release data with an unsupported or malformed CPE_NAME value, when
// parse parses it, then VersionModifier is nil.
func TestParseVersionModifierIgnoresInvalidCPE(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cpe  string
	}{
		{name: "too short", cpe: "cpe:/o:vendor:product:9"},
		{name: "empty component", cpe: "cpe:/o:vendor:product:9::update"},
		{name: "non-cpe value", cpe: "not-a-cpe-name"},
		{name: "cpe 2.3 form", cpe: "cpe:2.3:o:vendor:product:9:ga:*:*:*:*:*:*"},
		{name: "application part", cpe: "cpe:/a:vendor:product:9:update"},
		{name: "dangling escape", cpe: `cpe:/o:vendor:product:9:ga\`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parse([]byte("CPE_NAME=" + test.cpe + "\n"))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if got.VersionModifier != nil {
				t.Fatalf("VersionModifier = %q, want nil", *got.VersionModifier)
			}
		})
	}
}

// TestParseMalformedInput verifies that invalid key/value data returns an error.
//
// Given a line without a key/value separator, when parse parses it, then it
// returns an error.
func TestParseMalformedInput(t *testing.T) {
	t.Parallel()

	if _, err := parse([]byte("this is not valid\n")); err == nil {
		t.Fatal("Parse() error = nil, want an error")
	}
}
