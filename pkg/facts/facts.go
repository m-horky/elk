package facts

import "log/slog"

const (
	// certificateVersion is the version of the system facts certificate format.
	certificateVersion = "3.2"
)

// Facts contains the system facts that Elk can publish to a service.
//
// A nil field means that the corresponding fact could not be collected.
type Facts struct {
	// SystemCertificateVersion identifies the system facts certificate format.
	SystemCertificateVersion *string
}

// Collect returns the facts currently available from Elk's fact model.
//
// Collection of host-specific facts will be added independently. The system
// certificate version is always available and is populated by this initial
// implementation.
func Collect() (*Facts, error) {
	slog.Debug("collecting system facts")
	return &Facts{
		SystemCertificateVersion: new(certificateVersion),
	}, nil
}
