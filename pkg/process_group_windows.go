//go:build windows

package llamactl

import "os/exec"

func setProcAttrs(cmd *exec.Cmd) {
	// No-op on Windows
}
