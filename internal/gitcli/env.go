package gitcli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const pathMarker = "__BURLWOOD_PATH__"

var (
	shellPathOnce  sync.Once
	shellPathValue string
)

// macOS launches GUI apps with launchd's bare PATH (/usr/bin:/bin:/usr/sbin:/sbin),
// so subprocesses can't see Homebrew/nvm/docker paths a repo's git hooks rely on.
// Asking the user's login shell for its PATH matches what a terminal-launched git would see.
func shellPath() string {
	shellPathOnce.Do(func() {
		shellPathValue = os.Getenv("PATH")
		if runtime.GOOS == "windows" {
			return
		}

		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/zsh"
		}

		script := "printf '%s%s%s' '" + pathMarker + "' \"$PATH\" '" + pathMarker + "'"
		// Every git call waits on this probe, so a hanging shell rc file would freeze the app.
		// WaitDelay covers daemons the rc file starts that keep our stdout pipe open.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, shell, "-ilc", script)
		cmd.WaitDelay = time.Second
		detachFromTerminal(cmd)
		out, err := cmd.Output()
		if err != nil && !errors.Is(err, exec.ErrWaitDelay) {
			return
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		last := lines[len(lines)-1]
		start := strings.Index(last, pathMarker)
		if start == -1 {
			return
		}
		rest := last[start+len(pathMarker):]
		end := strings.Index(rest, pathMarker)
		if end == -1 {
			return
		}
		shellPathValue = rest[:end]
	})
	return shellPathValue
}

func mergedEnv() []string {
	base := os.Environ()
	result := make([]string, 0, len(base)+1)
	for _, kv := range base {
		if !strings.HasPrefix(kv, "PATH=") {
			result = append(result, kv)
		}
	}
	return append(result, "PATH="+shellPath())
}
