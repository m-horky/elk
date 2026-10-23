// Package constants exposes the system paths and network addresses used by
// Elk that are not configurable from its configuration file.
//
// Path and Address resolve each well-known location from the process
// environment first, falling back to the system default. Environment overrides
// are strictly a development aid and are not part of the supported
// configuration surface.
//
//	config := constants.Path(constants.ConfigPath)
//	lock := constants.Path(constants.LockPath)
//	address := constants.Address(constants.AWSIMDSAddress)
package constants
