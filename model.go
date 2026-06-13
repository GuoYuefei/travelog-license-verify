package verify

import (
	"sort"
	"time"
)

// LicenseType represents the type/category of a license.
type LicenseType string

const (
	LicenseTypeTrial     LicenseType = "trial"
	LicenseTypeStandard  LicenseType = "standard"
	LicenseTypeEnterprise LicenseType = "enterprise"
)

// License represents a verified Travelog license.
// It is populated from the JSON payload inside a .lic file
// after Ed25519 signature verification passes.
type License struct {
	// ID is the unique license identifier.
	ID string `json:"id"`

	// Product identifies the product this license is for (e.g. "travelog").
	Product string `json:"product"`

	// LicenseType is the type of license (trial, standard, enterprise).
	LicenseType LicenseType `json:"license_type"`

	// CustomerID is the unique identifier of the licensed customer.
	CustomerID string `json:"customer_id"`

	// CustomerName is the display name of the customer.
	CustomerName string `json:"customer_name"`

	// IssuedAt is the Unix timestamp when the license was issued.
	IssuedAt int64 `json:"issued_at"`

	// ExpiresAt is the Unix timestamp when the license expires.
	// A value of 0 means the license never expires.
	ExpiresAt int64 `json:"expires_at"`

	// MaxDevices is the maximum number of devices that can be activated.
	// A value of 0 means unlimited.
	MaxDevices int `json:"max_devices"`

	// Features is a map of boolean feature flags.
	Features map[string]bool `json:"features"`

	// Capabilities holds arbitrary key-value configuration data.
	Capabilities map[string]any `json:"capabilities,omitempty"`

	// Metadata holds optional string-keyed metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// RevokedAt is the Unix timestamp when the license was revoked.
	// A value of 0 means the license has not been revoked.
	RevokedAt int64 `json:"revoked_at,omitempty"`
}

// IsExpired returns true if the license has an expiration timestamp
// and the current time is past that timestamp.
// A license with ExpiresAt == 0 never expires.
func (l *License) IsExpired() bool {
	return l.ExpiresAt > 0 && l.ExpiresAt < time.Now().Unix()
}

// IsRevoked returns true if the license has been revoked.
func (l *License) IsRevoked() bool {
	return l.RevokedAt > 0
}

// IsValid returns true if the license is neither expired nor revoked.
// Note: this does NOT re-verify the cryptographic signature.
// Use Verify() to check signature integrity.
func (l *License) IsValid() bool {
	return !l.IsExpired() && !l.IsRevoked()
}

// IsFeatureEnabled returns true if the named feature exists and is enabled.
func (l *License) IsFeatureEnabled(name string) bool {
	if l.Features == nil {
		return false
	}
	enabled, ok := l.Features[name]
	return ok && enabled
}

// GetCapability returns the capability value for the given key, or nil if not set.
func (l *License) GetCapability(key string) any {
	if l.Capabilities == nil {
		return nil
	}
	return l.Capabilities[key]
}

// CapabilityKeys returns all capability key names in sorted order.
// Use this to discover what capabilities are available instead of guessing keys.
func (l *License) CapabilityKeys() []string {
	if l.Capabilities == nil {
		return nil
	}
	keys := make([]string, 0, len(l.Capabilities))
	for k := range l.Capabilities {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// RangeCapabilities calls fn for each capability key-value pair in sorted key order.
func (l *License) RangeCapabilities(fn func(key string, val any)) {
	for _, k := range l.CapabilityKeys() {
		fn(k, l.Capabilities[k])
	}
}
