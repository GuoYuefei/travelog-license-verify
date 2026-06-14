package verify

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/slashdevops/machineid"
)

const defaultCommandTimeout = 5 * time.Second

// hideWindowExecutor implements machineid.CommandExecutor with HideWindow
// on Windows to prevent console windows from flashing when wmic/PowerShell
// commands are run for hardware fingerprinting.
type hideWindowExecutor struct{}

func (e *hideWindowExecutor) Execute(ctx context.Context, name string, args ...string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, name, args...)
	hideWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("command %s: %w", name, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// DeviceFingerprint generates a hardware-bound device identifier
// using CPU information, motherboard serial, and MAC addresses.
//
// This produces a consistent fingerprint for the same machine,
// suitable for use with Client.Activate() and Client.Heartbeat().
//
// The fingerprint is computed as SHA-256(CPU + Motherboard + MAC).
func DeviceFingerprint(ctx context.Context) (string, error) {
	id, err := machineid.New().
		WithExecutor(&hideWindowExecutor{}).
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
