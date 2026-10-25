package repo

import (
	"fmt"
	"strings"

	"burlwood/internal/gitcli"
)

type SwitchResult struct {
	Success       bool     `json:"success"`
	StashUsed     bool     `json:"stashUsed"`
	ConflictFiles []string `json:"conflictFiles"`
	Message       string   `json:"message"`
}

func (m *Manager) SwitchBranch(repoPath, target string, trackRemote bool) (SwitchResult, error) {
	var checkoutArgs []string
	if trackRemote {
		checkoutArgs = []string{"checkout", "--track", target}
	} else {
		checkoutArgs = []string{"checkout", target}
	}

	localName := target
	if trackRemote {
		if _, after, ok := strings.Cut(target, "/"); ok {
			localName = after
		}
	}

	// First attempt: direct checkout (succeeds when changes don't conflict).
	if _, err := gitcli.Run(repoPath, checkoutArgs...); err == nil {
		return SwitchResult{Success: true, Message: "Switched to " + localName}, nil
	} else {
		exitErr, ok := err.(*gitcli.ExitError)
		if !ok {
			return SwitchResult{}, err
		}
		// Only auto-stash when git explicitly refuses due to uncommitted changes.
		stderr := exitErr.Stderr
		if !strings.Contains(stderr, "would be overwritten by checkout") &&
			!strings.Contains(stderr, "Your local changes to the following files") {
			return SwitchResult{}, fmt.Errorf("%s", strings.TrimSpace(stderr))
		}
	}

	stashMsg := "burlwood: auto-stash before switching to " + target
	if err := gitcli.StashPushNamed(repoPath, stashMsg); err != nil {
		return SwitchResult{}, fmt.Errorf("stash failed: %w", err)
	}

	if _, err := gitcli.Run(repoPath, checkoutArgs...); err != nil {
		// Checkout still failed; restore stash and surface the real error.
		_ = gitcli.StashPop(repoPath)
		return SwitchResult{}, fmt.Errorf("checkout failed after stash: %w", err)
	}

	if err := gitcli.StashPop(repoPath); err == nil {
		return SwitchResult{
			Success:   true,
			StashUsed: true,
			Message:   "Switched to " + localName + " (stash re-applied)",
		}, nil
	}

	// Stash pop produced conflicts — collect affected paths.
	conflicts, _ := switchConflictFiles(repoPath)
	return SwitchResult{
		Success:       true,
		StashUsed:     true,
		ConflictFiles: conflicts,
		Message:       "Switched to " + localName + " — stash pop has conflicts",
	}, nil
}

func switchConflictFiles(repoPath string) ([]string, error) {
	result, err := gitcli.Run(repoPath, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	var files []string
	for line := range strings.SplitSeq(result.Stdout, "\n") {
		if len(line) < 4 {
			continue
		}
		x, y := line[0], line[1]
		if IsConflictedStatus(string(x), string(y)) {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files, nil
}
