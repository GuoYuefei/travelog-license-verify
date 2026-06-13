package verify

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCapability(t *testing.T) {
	caps := map[string]any{
		"name": "test",
		"age":  float64(30),
		"flag": true,
	}

	v, ok := GetCapability(caps, "name")
	require.True(t, ok)
	require.Equal(t, "test", v)

	_, ok = GetCapability(caps, "nonexistent")
	require.False(t, ok)

	// nil map
	_, ok = GetCapability(nil, "key")
	require.False(t, ok)
}

func TestGetCapabilityString(t *testing.T) {
	caps := map[string]any{
		"name":   "hello",
		"number": float64(42),
	}

	v, ok := GetCapabilityString(caps, "name")
	require.True(t, ok)
	require.Equal(t, "hello", v)

	// wrong type
	_, ok = GetCapabilityString(caps, "number")
	require.False(t, ok)

	// missing
	_, ok = GetCapabilityString(caps, "missing")
	require.False(t, ok)
}

func TestGetCapabilityInt(t *testing.T) {
	caps := map[string]any{
		"count_float": float64(42),
		"name":        "hello",
	}

	v, ok := GetCapabilityInt(caps, "count_float")
	require.True(t, ok)
	require.Equal(t, 42, v)

	// wrong type
	_, ok = GetCapabilityInt(caps, "name")
	require.False(t, ok)

	// missing
	_, ok = GetCapabilityInt(caps, "missing")
	require.False(t, ok)
}

func TestGetCapabilityBool(t *testing.T) {
	caps := map[string]any{
		"enabled":  true,
		"disabled": false,
		"name":     "hello",
	}

	v, ok := GetCapabilityBool(caps, "enabled")
	require.True(t, ok)
	require.True(t, v)

	v, ok = GetCapabilityBool(caps, "disabled")
	require.True(t, ok)
	require.False(t, v)

	// wrong type
	_, ok = GetCapabilityBool(caps, "name")
	require.False(t, ok)

	// missing
	_, ok = GetCapabilityBool(caps, "missing")
	require.False(t, ok)
}

func TestLicenseCapabilityHelpers(t *testing.T) {
	lic := makeValidLicense()

	// Using the package-level functions
	v, ok := GetCapabilityString(lic.Capabilities, "tier")
	require.True(t, ok)
	require.Equal(t, "professional", v)

	gb, ok := GetCapabilityInt(lic.Capabilities, "storage_gb")
	require.True(t, ok)
	require.Equal(t, 100, gb)

	audit, ok := GetCapabilityBool(lic.Capabilities, "audit_log")
	require.True(t, ok)
	require.True(t, audit)
}
