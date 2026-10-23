package constants

import "os"

type addressID string

const (
	AWSIMDSAddress    addressID = "ELK_IMDS_AWS"
	AzureIMDSAddress  addressID = "ELK_IMDS_AZURE"
	GoogleIMDSAddress addressID = "ELK_IMDS_GOOGLE"
)

var addresses = map[addressID]string{ //nolint:gochecknoglobals
	AWSIMDSAddress:    "http://169.254.169.254",
	AzureIMDSAddress:  "http://169.254.169.254",
	GoogleIMDSAddress: "http://metadata.google.internal",
}

// Address returns the IMDS base URL for name, preferring a non-empty environment override.
func Address(name addressID) string {
	if address := os.Getenv(string(name)); address != "" {
		return address
	}

	return addresses[name]
}
