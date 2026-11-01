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

// Checking out origin/x when a local x exists switches to x instead of failing on
// "branch already exists", and fast-forwards x when it tracks origin/x and is behind. When x
// can't simply be moved (it has its own commits, or tracks something else), the message says
// how it differs, since the user asked for origin/x and is now on something else.
func (m *Manager) CheckoutRemoteBranch(repoPath, remote, name string) (SwitchResult, error) {
	remoteRef := remote + "/" + name
	if revParse(repoPath, "refs/heads/"+name) == "" {
		return m.SwitchBranch(repoPath, remoteRef, true)
	}
	res, err := m.SwitchBranch(repoPath, name, false)
	if err != nil || len(res.ConflictFiles) > 0 {
		return res, err
	}
	switch upstream := upstreamOf(repoPath, name); upstream {
	case remoteRef:
	case "":
		res.Message += ". It doesn't track " + remoteRef + "."
		return res, nil
	default:
		res.Message += ". It tracks " + upstream + ", not " + remoteRef + "."
		return res, nil
	}

	ahead, behind, err := aheadBehind(repoPath, "HEAD", remoteRef)
	if err != nil {
		return res, nil
	}
	if ahead == 0 && behind > 0 {
		if _, err := gitcli.Run(repoPath, "merge", "--ff-only", "--quiet", remoteRef); err == nil {
			res.Message = "Switched to " + name + " and fast-forwarded it to " + remoteRef
			return res, nil
		}
	}
	switch {
	case ahead > 0 && behind > 0:
		res.Message += fmt.Sprintf(". It has %s that %s doesn't, and is %d behind it. Pull to bring those in.",
			commits(ahead), remoteRef, behind)
	case ahead > 0:
		res.Message += fmt.Sprintf(". It has %s that %s doesn't.", commits(ahead), remoteRef)
	case behind > 0:
		res.Message += fmt.Sprintf(". It's %s behind %s. Pull to bring those in.", commits(behind), remoteRef)
	}
	return res, nil
}

func aheadBehind(repoPath, ours, theirs string) (int, int, error) {
	out, err := gitcli.Run(repoPath, "rev-list", "--left-right", "--count", ours+"..."+theirs)
	if err != nil {
		return 0, 0, err
	}
	var ahead, behind int
	if _, err := fmt.Sscanf(strings.TrimSpace(out.Stdout), "%d %d", &ahead, &behind); err != nil {
		return 0, 0, err
	}
	return ahead, behind, nil
}

func commits(n int) string {
	if n == 1 {
		return "1 commit"
	}
	return fmt.Sprintf("%d commits", n)
}
