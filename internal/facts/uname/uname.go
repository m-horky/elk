package uname

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Uname contains the system identification values reported by uname(2).
// A nil field means that the corresponding value is unavailable or empty.
type Uname struct {
	// Sysname is the operating system kernel name, such as "Linux".
	Sysname *string
	// Nodename is the network node hostname.
	Nodename *string
	// Release is the operating system kernel release.
	Release *string
	// Version is the operating system kernel version.
	Version *string
	// Machine is the hardware identifier, such as "x86_64".
	Machine *string
}

// Collect reads the current system identification values through uname(2).
func Collect() (Uname, error) {
	var info unix.Utsname

	if err := unix.Uname(&info); err != nil {
		return Uname{}, fmt.Errorf("collect uname: %w", err)
	}

	return fromUtsname(info), nil
}

// fromUtsname converts a unix.Utsname into an Uname.
// Each byte array is decoded up to its first NUL byte, and empty values become
// nil pointers.
func fromUtsname(info unix.Utsname) Uname {
	return Uname{
		Sysname:  stringPointer(info.Sysname[:]),
		Nodename: stringPointer(info.Nodename[:]),
		Release:  stringPointer(info.Release[:]),
		Version:  stringPointer(info.Version[:]),
		Machine:  stringPointer(info.Machine[:]),
	}
}

// stringPointer decodes a fixed-size byte array up to its first NUL byte and
// returns nil for an empty result.
func stringPointer(value []byte) *string {
	decoded := unix.ByteSliceToString(value)
	if decoded == "" {
		return nil
	}

	return &decoded
}
