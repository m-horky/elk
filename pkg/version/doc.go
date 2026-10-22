// Package version provides Elk's build version information.
//
// The version defaults to "dev" and may be replaced at build time:
//
//	go build -ldflags "-X github.com/m-horky/elk/pkg/version.Version=1.2.3"
//
// Read it directly:
//
//	fmt.Println(version.Version)
package version
