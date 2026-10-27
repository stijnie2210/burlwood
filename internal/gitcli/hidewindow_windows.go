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

// Unlike hideWindow it keeps the window state: SW_HIDE would also hide a GUI editor's first window.
// Editors are often .cmd shims (code.cmd), which would otherwise flash a console.
func noConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
