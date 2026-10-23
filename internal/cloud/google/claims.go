package google

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Claims are the claims of the Google instance identity JWT.
type Claims struct {
	Audience string `json:"aud"`
	Email    string `json:"email"`
	// ExpiresAt decodes from the numeric exp claim via custom unmarshaling.
	ExpiresAt time.Time `json:"exp"`
	// IssuedAt decodes from the numeric iat claim via custom unmarshaling.
	IssuedAt time.Time    `json:"iat"`
	Issuer   string       `json:"iss"`
	Google   GoogleClaims `json:"google"`
}

// GoogleClaims contains the Google-specific claims of the identity JWT.
type GoogleClaims struct {
	ComputeEngine ComputeEngine `json:"compute_engine"`
}

// ComputeEngine contains the Compute Engine identity claims.
type ComputeEngine struct {
	// InstanceID and ProjectNumber decode from numeric claims via custom
	// unmarshaling, preserving every digit.
	InstanceID    string   `json:"instance_id"`
	InstanceName  string   `json:"instance_name"`
	ProjectID     string   `json:"project_id"`
	ProjectNumber string   `json:"project_number"`
	Zone          string   `json:"zone"`
	LicenseIDs    []string `json:"license_id"`
}

// claimsPayload carries raw numeric claims until they are converted without
// passing through floating point.
type claimsPayload struct {
	Audience  string          `json:"aud"`
	Email     string          `json:"email"`
	ExpiresAt json.RawMessage `json:"exp"`
	IssuedAt  json.RawMessage `json:"iat"`
	Issuer    string          `json:"iss"`
	Google    GoogleClaims    `json:"google"`
}

// claimsOutput carries numeric time claims as exact JSON numbers when
// encoding.
type claimsOutput struct {
	Audience  string       `json:"aud,omitempty"`
	Email     string       `json:"email,omitempty"`
	ExpiresAt json.Number  `json:"exp,omitempty"`
	IssuedAt  json.Number  `json:"iat,omitempty"`
	Issuer    string       `json:"iss,omitempty"`
	Google    GoogleClaims `json:"google"`
}

// computeEnginePayload carries raw numeric claims until they are converted
// without passing through floating point.
type computeEnginePayload struct {
	InstanceID    json.RawMessage `json:"instance_id"`
	InstanceName  string          `json:"instance_name"`
	ProjectID     string          `json:"project_id"`
	ProjectNumber json.RawMessage `json:"project_number"`
	Zone          string          `json:"zone"`
	LicenseIDs    []string        `json:"license_id"`
}

// UnmarshalJSON decodes the identity JWT claims, converting numeric time to time.Time.
func (c *Claims) UnmarshalJSON(data []byte) error {
	var payload claimsPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode JWT claims JSON: %w", err)
	}

	expiresAt, err := timeClaim(payload.ExpiresAt, "exp")
	if err != nil {
		return err
	}

	issuedAt, err := timeClaim(payload.IssuedAt, "iat")
	if err != nil {
		return err
	}

	c.Audience = payload.Audience
	c.Email = payload.Email
	c.ExpiresAt = expiresAt
	c.IssuedAt = issuedAt
	c.Issuer = payload.Issuer
	c.Google = payload.Google

	return nil
}

// MarshalJSON encodes the identity JWT claims so that they round-trip through
// encoding/json, including the numeric time claims.
func (c Claims) MarshalJSON() ([]byte, error) {
	payload := claimsOutput{
		Audience: c.Audience,
		Email:    c.Email,
		Issuer:   c.Issuer,
		Google:   c.Google,
	}

	payload.ExpiresAt = numberClaim(c.ExpiresAt)
	payload.IssuedAt = numberClaim(c.IssuedAt)

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode JWT claims JSON: %w", err)
	}

	return encoded, nil
}

// UnmarshalJSON decodes the Compute Engine identity claims, preserving every
// digit of numeric claims in their string fields.
func (c *ComputeEngine) UnmarshalJSON(data []byte) error {
	var payload computeEnginePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode compute engine claims JSON: %w", err)
	}

	instanceID, err := numericClaim(payload.InstanceID, "instance_id")
	if err != nil {
		return err
	}

	projectNumber, err := numericClaim(payload.ProjectNumber, "project_number")
	if err != nil {
		return err
	}

	c.InstanceID = instanceID
	c.InstanceName = payload.InstanceName
	c.ProjectID = payload.ProjectID
	c.ProjectNumber = projectNumber
	c.Zone = payload.Zone
	c.LicenseIDs = payload.LicenseIDs

	return nil
}

// numericClaim returns a claim as its exact digits. Both JSON numbers and
// quoted numbers are accepted, because Google has returned instance_id in
// both forms.
func numericClaim(raw json.RawMessage, name string) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return "", nil
	}

	if strings.HasPrefix(trimmed, `"`) {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", fmt.Errorf("decode %s claim: %w", name, err)
		}

		return value, nil
	}

	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return "", fmt.Errorf("decode %s claim: %w", name, err)
	}

	return number.String(), nil
}

// timeClaim converts an exact numeric claim into a time.Time value.
func timeClaim(raw json.RawMessage, name string) (time.Time, error) {
	value, err := numericClaim(raw, name)
	if err != nil {
		return time.Time{}, err
	}

	if value == "" {
		return time.Time{}, nil
	}

	seconds, err := json.Number(value).Int64()
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s claim as Unix time: %w", name, err)
	}

	return time.Unix(seconds, 0).UTC(), nil
}

// numberClaim converts a time.Time value into an exact numeric JSON claim. A
// zero time yields an empty claim, which is omitted when encoding.
func numberClaim(value time.Time) json.Number {
	if value.IsZero() {
		return ""
	}

	return json.Number(strconv.FormatInt(value.Unix(), 10))
}

// validateComputeEngine verifies that the required Compute Engine claims are
// present in the decoded identity.
func validateComputeEngine(engine ComputeEngine) error {
	switch {
	case engine.InstanceID == "":
		return errors.New("identity JWT has no compute_engine.instance_id claim")
	case engine.ProjectID == "":
		return errors.New("identity JWT has no compute_engine.project_id claim")
	}

	return nil
}
