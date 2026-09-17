// Package facts provides the public system facts model used by Elk.
//
// Facts are represented independently of any service-specific wire format.
// Applications can collect the current facts with Collect and convert them to
// the format required by a downstream service.
//
// Usage:
//
//	systemFacts, err := facts.Collect()
package facts
