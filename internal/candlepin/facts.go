package candlepin

import "github.com/m-horky/elk/pkg/facts"

// NewFactsDTO converts Elk's public facts model to Candlepin's fact format.
// Unavailable facts are omitted from the result.
func NewFactsDTO(systemFacts *facts.Facts) map[string]string {
	consumerFacts := make(map[string]string)

	if systemFacts.SystemCertificateVersion != nil {
		consumerFacts["system.certificate_version"] = *systemFacts.SystemCertificateVersion
	}

	return consumerFacts
}
