package uname

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/sys/unix"
)

// TestCollect verifies that the current Linux system reports every uname value.
//
// Given a Linux host, when Collect reads the system identification values,
// then all fields are populated and the system name is Linux.
func TestCollect(t *testing.T) {
	t.Parallel()

	got, err := Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if got.Sysname == nil || *got.Sysname != "Linux" {
		t.Errorf("Collect() sysname = %v, want Linux", got.Sysname)
	}

	if got.Nodename == nil || *got.Nodename == "" {
		t.Errorf("Collect() nodename = %v, want a non-empty value", got.Nodename)
	}

	if got.Release == nil || *got.Release == "" {
		t.Errorf("Collect() release = %v, want a non-empty value", got.Release)
	}

	if got.Version == nil || *got.Version == "" {
		t.Errorf("Collect() version = %v, want a non-empty value", got.Version)
	}

	if got.Machine == nil || *got.Machine == "" {
		t.Errorf("Collect() machine = %v, want a non-empty value", got.Machine)
	}
}

// TestFromUtsname verifies that every Utsname field is converted.
//
// Given a unix.Utsname with constructed values, when fromUtsname converts it,
// then each value appears in the corresponding Uname field.
func TestFromUtsname(t *testing.T) {
	t.Parallel()

	var info unix.Utsname

	copy(info.Sysname[:], "Linux")
	copy(info.Nodename[:], "workstation")
	copy(info.Release[:], "6.12.0-1")
	copy(info.Version[:], "#1 SMP PREEMPT")
	copy(info.Machine[:], "x86_64")

	want := Uname{
		Sysname:  new("Linux"),
		Nodename: new("workstation"),
		Release:  new("6.12.0-1"),
		Version:  new("#1 SMP PREEMPT"),
		Machine:  new("x86_64"),
	}

	got := fromUtsname(info)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("FromUtsname() mismatch (-want +got):\n%s", diff)
	}
}
