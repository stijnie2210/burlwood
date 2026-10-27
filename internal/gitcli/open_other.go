//go:build !darwin && !windows

package gitcli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

func openCommand(path string) *exec.Cmd {
	return exec.Command("xdg-open", path)
}

func revealCommand(path string) *exec.Cmd {
	return exec.Command("xdg-open", filepath.Dir(path))
}

func terminalCommand(dir string) *exec.Cmd {
	candidates := []string{os.Getenv("TERMINAL"), "x-terminal-emulator", "gnome-terminal", "konsole", "xterm"}
	for _, name := range candidates {
		if name == "" {
			continue
		}
		if _, err := exec.LookPath(name); err == nil {
			cmd := exec.Command(name)
			cmd.Dir = dir
			return cmd
		}
	}
	cmd := exec.Command("x-terminal-emulator")
	cmd.Err = errors.New("no terminal found; set $TERMINAL")
	return cmd
}
