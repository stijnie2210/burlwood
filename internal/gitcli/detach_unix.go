//go:build !windows

package gitcli

import (
	"os/exec"
	"syscall"
)

// An interactive shell takes over the controlling terminal; when Burlwood runs as a background job of one
// (`burlwood &`), that stops Burlwood's whole process group, so the probe runs in a session of its own.
func detachFromTerminal(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
