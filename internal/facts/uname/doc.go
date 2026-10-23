// Package uname collects system identification values from the Linux uname(2) system call.
//
// Collect reads the current kernel identification values, while fromUtsname
// converts a caller-provided unix.Utsname into the package result type. Values
// that are empty or unavailable are represented as nil pointers; there is no
// separate valid-empty state.
//
//	info, _ := uname.Collect()
//	fmt.Println(*info.Sysname)
package uname
