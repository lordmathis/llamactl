//go:build windows

package instance

import "os/exec"

func setProcAttrs(cmd *exec.Cmd) {
	// No-op on Windows. (os.Interrupt/SIGINT are no-ops for child processes
	// here, and GenerateConsoleCtrlEvent could not be delivered reliably to
	// the child in a service session, so graceful stop falls back to the
	// force-kill after a short grace in process.go.)
}

// signalStop is a no-op on Windows: SIGINT does not reach child processes
// and the console Ctrl-C path proved unreliable, so the caller relies on
// the force-kill after the grace period.
func signalStop(cmd *exec.Cmd) {
	// No-op on Windows — see setProcAttrs.
}
