package gitcli

import "os/exec"

func openCommand(path string) *exec.Cmd {
	return exec.Command("open", path)
}

func revealCommand(path string) *exec.Cmd {
	return exec.Command("open", "-R", path)
}

func terminalCommand(dir string) *exec.Cmd {
	return exec.Command("open", "-a", "Terminal", dir)
}
