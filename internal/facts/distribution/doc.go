// Package distribution collects distribution facts from an os-release file.
//
// Collect reads the file through an injected fs.FS and returns its contents
// as a Distribution. Missing os-release keys are represented by nil pointers.
//
// For example, collect the distribution facts from /usr/lib/os-release:
//
//	dist, _ := distribution.Collect(fs.Filesystem{}, "/usr/lib/os-release")
//	_ = dist.ID
package distribution
