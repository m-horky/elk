// Package virtwhat detects virtualization by running the virt-what binary and
// parsing its output into guest status and host type facts.
//
// Collect runs virt-what through an injected exec.Runner and a caller-supplied
// context.
//
//	virt, _ := virtwhat.Collect(context.Background(), exec.OSRunner{}, "/usr/sbin/virt-what")
//	isGuest := virt.IsGuest
//	hostType := virt.HostType
package virtwhat
