//go:build !windows

package gitcli

import "os/exec"

func hideWindow(*exec.Cmd) {}
