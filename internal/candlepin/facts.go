package candlepin

import (
	"strconv"

	"github.com/m-horky/elk/pkg/facts"
)

// NewFactsDTO converts Elk's public facts model to the dictionary sent to
// Candlepin when registering a consumer.
//
// Unavailable facts are omitted. Valid empty values and false booleans are
// preserved.
//
// Candlepin uses system.certificate_version, uname.machine, virt.is_guest,
// cpu.cpu_socket(s), and cpu.core(s)_per_socket directly. The result mirrors
// subscription-manager by sending the complete lscpu namespace and deriving the
// cpu.* facts.
func NewFactsDTO(systemFacts *facts.Facts) map[string]string {
	result := make(map[string]string)
	if systemFacts == nil {
		return result
	}

	setFact(result, "system.certificate_version", systemFacts.SystemCertificateVersion)
	setFact(result, "uname.sysname", systemFacts.UnameSysname)
	setFact(result, "uname.nodename", systemFacts.UnameNodename)
	setFact(result, "uname.release", systemFacts.UnameRelease)
	setFact(result, "uname.version", systemFacts.UnameVersion)
	setFact(result, "uname.machine", systemFacts.UnameMachine)
	setFact(result, "distribution.id", systemFacts.DistributionID)
	setFact(result, "distribution.name", systemFacts.DistributionName)
	setFact(result, "distribution.version", systemFacts.DistributionVersion)
	setFact(result, "distribution.version.modifier", systemFacts.DistributionVersionModifier)

	if systemFacts.VirtIsGuest != nil {
		result["virt.is_guest"] = strconv.FormatBool(*systemFacts.VirtIsGuest)
	}

	setFact(result, "virt.host_type", systemFacts.VirtHostType)

	for key, value := range systemFacts.LSCPU {
		result["lscpu."+key] = value
	}

	setFact(result, "cpu.cpu(s)", copyFact(systemFacts.LSCPU, "cpu(s)"))
	setFact(result, "cpu.cpu_socket(s)", copyFact(systemFacts.LSCPU, "socket(s)"))
	setFact(result, "cpu.core(s)_per_socket", copyFact(systemFacts.LSCPU, "core(s)_per_socket"))
	setFact(result, "cpu.thread(s)_per_core", copyFact(systemFacts.LSCPU, "thread(s)_per_core"))

	return result
}

// copyFact returns a pointer to the value of key when it exists.
func copyFact(values map[string]string, key string) *string {
	value, ok := values[key]
	if !ok {
		return nil
	}

	return &value
}

// setFact stores value under key when the value is available.
func setFact(target map[string]string, key string, value *string) {
	if value == nil {
		return
	}

	target[key] = *value
}
