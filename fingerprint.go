package verify

import (
	"context"
	"fmt"

	"github.com/slashdevops/machineid"
)

// DeviceFingerprint generates a hardware-bound device identifier
// using CPU information, motherboard serial, and MAC addresses.
//
// This produces a consistent fingerprint for the same machine,
// suitable for use with Client.Activate() and Client.Heartbeat().
//
// The fingerprint is computed as SHA-256(CPU + Motherboard + MAC).
func DeviceFingerprint(ctx context.Context) (string, error) {
	id, err := machineid.New().
		WithCPU().
		WithMotherboard().
		WithMAC().
		ID(ctx)
	if err != nil {
		return "", fmt.Errorf("verify: device fingerprint failed: %w", err)
	}
	return id, nil
}

// MustDeviceFingerprint is like DeviceFingerprint but panics on error.
// Useful for one-liners in application setup code.
func MustDeviceFingerprint(ctx context.Context) string {
	id, err := DeviceFingerprint(ctx)
	if err != nil {
		panic(err)
	}
	return id
}
