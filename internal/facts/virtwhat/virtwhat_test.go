package virtwhat

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestParseKVM verifies a single guest type is reported as a guest.
//
// Given virt-what reports kvm, when parse reads the output,
// then IsGuest is true and HostType is kvm.
func TestParseKVM(t *testing.T) {
	t.Parallel()

	got := parse([]byte("kvm\n"))

	want := Virt{IsGuest: new(true), HostType: new("kvm")}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Parse() mismatch (-want +got): %s", diff)
	}
}

// TestParseXenHVM verifies multiple guest types are joined into one host type.
//
// Given virt-what reports xen and xen-hvm on separate lines, when parse reads
// the output, then the values are joined with a comma and a space.
func TestParseXenHVM(t *testing.T) {
	t.Parallel()

	got := parse([]byte("xen\nxen-hvm\n"))

	want := Virt{IsGuest: new(true), HostType: new("xen, xen-hvm")}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Parse() mismatch (-want +got): %s", diff)
	}
}

// TestParseXenDom0 verifies a dom0 host is not reported as a guest.
//
// Given virt-what reports xen-dom0, when parse reads the output,
// then IsGuest is false and HostType keeps the reported value.
func TestParseXenDom0(t *testing.T) {
	t.Parallel()

	got := parse([]byte("xen-dom0\n"))

	want := Virt{IsGuest: new(false), HostType: new("xen-dom0")}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Parse() mismatch (-want +got): %s", diff)
	}
}

// TestParseEmpty verifies empty output reports a non-guest host.
//
// Given virt-what produces no output, when parse reads it,
// then IsGuest points to false and HostType points to Not Applicable.
func TestParseEmpty(t *testing.T) {
	t.Parallel()

	got := parse(nil)

	if got.IsGuest == nil {
		t.Fatal("Parse().IsGuest is nil")
	}

	if *got.IsGuest {
		t.Error("Parse().IsGuest = true, want false")
	}

	if got.HostType == nil {
		t.Fatal("Parse().HostType is nil")
	}

	if *got.HostType != "" {
		t.Errorf("Parse().HostType = %q, want %q", *got.HostType, "")
	}
}
