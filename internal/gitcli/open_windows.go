package gitcli

import (
	"os/exec"
	"syscall"
)

func openCommand(path string) *exec.Cmd {
	return exec.Command("explorer.exe", path)
}

func revealCommand(path string) *exec.Cmd {
	cmd := exec.Command("explorer.exe")
	// explorer parses /select itself and ignores it when the whole argument is quoted, as Go would pass it.
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + path + `"`}
	return cmd
}

func terminalCommand(dir string) *exec.Cmd {
	if _, err := exec.LookPath("wt.exe"); err == nil {
		return exec.Command("wt.exe", "-d", dir)
	}
	cmd := exec.Command("powershell.exe", "-NoExit")
	cmd.Dir = dir
	return cmd
}
