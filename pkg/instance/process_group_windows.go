//go:build windows

package instance

import "os/exec"

func setProcAttrs(cmd *exec.Cmd) {
	// No-op on Windows
}
