package gitcli

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// Burlwood is a GUI-subsystem app, so without this every git call opens its own console window.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
