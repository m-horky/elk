package candlepin

// Owner represents a Candlepin owner or organization associated with a user.
type Owner struct {
	// ID is Candlepin's internal identifier for the owner.
	ID string `json:"id"`
	// DisplayName is the human-readable name of the owner.
	DisplayName string `json:"displayName"`
	// Key is the stable organization key used when referring to the owner.
	Key string `json:"key"`
	// ContentPrefix is the prefix used for the owner's content.
	ContentPrefix string `json:"contentPrefix"`
	// Anonymous reports whether the owner is the anonymous organization.
	Anonymous bool `json:"anonymous"`
	// Claimed reports whether the owner has been claimed.
	Claimed bool `json:"claimed"`
}

// IdentityCertificate contains the identity certificate returned for a consumer.
type IdentityCertificate struct {
	// ID is Candlepin's internal identifier for the certificate.
	ID string `json:"id"`
	// Key is the PEM-encoded private key associated with the certificate.
	Key string `json:"key"`
	// Cert is the PEM-encoded certificate.
	Cert string `json:"cert"`
	// Serial is Candlepin's certificate serial value.
	Serial any `json:"serial"`
}

// ConsumerType identifies the kind of system represented by a consumer.
type ConsumerType struct {
	// ID is Candlepin's internal identifier for the consumer type.
	ID string `json:"id,omitempty"`
	// Label is the human-readable label for the consumer type.
	Label string `json:"label,omitempty"`
	// Manifest reports whether this consumer type represents a manifest consumer.
	Manifest bool `json:"manifest,omitempty"`
}

// Consumer represents a registered system in Candlepin.
type Consumer struct {
	// ID is Candlepin's internal identifier for the consumer.
	ID string `json:"id"`
	// UUID is the globally unique identifier assigned to the consumer.
	UUID string `json:"uuid"`
	// Name is the consumer's display name.
	Name string `json:"name"`
	// Username is the user associated with the consumer.
	Username string `json:"username"`
	// Owner is the organization that owns the consumer.
	Owner *Owner `json:"owner"`
	// Type describes the kind of consumer.
	Type *ConsumerType `json:"type"`
	// IDCert is the identity certificate issued to the consumer.
	IDCert *IdentityCertificate `json:"idCert"`
	// Facts contains system facts reported by the consumer.
	Facts map[string]string `json:"facts"`
}
