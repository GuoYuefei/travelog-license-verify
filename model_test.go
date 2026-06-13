package verify

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLicense_IsExpired_NotExpired(t *testing.T) {
	lic := makeValidLicense()
	require.False(t, lic.IsExpired())
}

func TestLicense_IsExpired_Expired(t *testing.T) {
	lic := makeExpiredLicense()
	require.True(t, lic.IsExpired())
}

func TestLicense_IsExpired_NeverExpires(t *testing.T) {
	lic := makeNeverExpireLicense()
	require.False(t, lic.IsExpired())
}

func TestLicense_IsRevoked(t *testing.T) {
	lic := makeValidLicense()
	require.False(t, lic.IsRevoked())

	lic.RevokedAt = time.Now().Unix()
	require.True(t, lic.IsRevoked())
}

func TestLicense_IsValid(t *testing.T) {
	// Valid license
	lic := makeValidLicense()
	require.True(t, lic.IsValid())

	// Expired
	lic = makeExpiredLicense()
	require.False(t, lic.IsValid())

	// Revoked (but not expired)
	lic = makeValidLicense()
	lic.RevokedAt = time.Now().Unix()
	require.False(t, lic.IsValid())
}

func TestLicense_IsFeatureEnabled(t *testing.T) {
	lic := makeValidLicense()

	// Feature enabled
	require.True(t, lic.IsFeatureEnabled("export_pdf"))
	require.True(t, lic.IsFeatureEnabled("export_csv"))

	// Feature disabled
	require.False(t, lic.IsFeatureEnabled("bulk_import"))

	// Feature not present
	require.False(t, lic.IsFeatureEnabled("nonexistent_feature"))

	// Nil features
	lic.Features = nil
	require.False(t, lic.IsFeatureEnabled("anything"))
}

func TestLicense_GetCapability(t *testing.T) {
	lic := makeValidLicense()

	// String capability
	tier := lic.GetCapability("tier")
	require.Equal(t, "professional", tier)

	// Numeric capability
	storage := lic.GetCapability("storage_gb")
	require.Equal(t, float64(100), storage)

	// Bool capability
	audit := lic.GetCapability("audit_log")
	require.Equal(t, true, audit)

	// Not present
	require.Nil(t, lic.GetCapability("nonexistent"))

	// Nil capabilities
	lic.Capabilities = nil
	require.Nil(t, lic.GetCapability("anything"))
}

func TestLicense_CapabilityKeys(t *testing.T) {
	lic := makeValidLicense()
	keys := lic.CapabilityKeys()
	require.ElementsMatch(t, []string{"audit_log", "storage_gb", "tier"}, keys)

	// Nil capabilities
	lic.Capabilities = nil
	require.Nil(t, lic.CapabilityKeys())
}

func TestLicense_RangeCapabilities(t *testing.T) {
	lic := makeValidLicense()
	var keys []string
	lic.RangeCapabilities(func(k string, v any) {
		keys = append(keys, k)
	})
	require.Equal(t, []string{"audit_log", "storage_gb", "tier"}, keys)

	// Nil capabilities
	lic.Capabilities = nil
	count := 0
	lic.RangeCapabilities(func(k string, v any) {
		count++
	})
	require.Equal(t, 0, count)
}

func TestLicenseType_Constants(t *testing.T) {
	require.Equal(t, LicenseType("trial"), LicenseTypeTrial)
	require.Equal(t, LicenseType("standard"), LicenseTypeStandard)
	require.Equal(t, LicenseType("enterprise"), LicenseTypeEnterprise)
}
