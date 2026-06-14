//go:build !windows

package verify

import "os/exec"

func hideWindow(cmd *exec.Cmd) {
	// No-op: console windows don't exist on Linux/macOS.
}
