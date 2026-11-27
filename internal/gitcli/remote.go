package gitcli

func FetchAll(repoPath string, prune bool) error {
	args := []string{"fetch", "--all"}
	if prune {
		args = append(args, "--prune")
	}
	_, err := Run(repoPath, args...)
	return err
}

func FetchRemote(repoPath, remote string, prune bool) error {
	args := []string{"fetch", remote}
	if prune {
		args = append(args, "--prune")
	}
	_, err := Run(repoPath, args...)
	return err
}

// PullMode values mirror the toolbar's pull-mode picker.
const (
	PullModeFastForward     = "ff"      // no explicit flag: honors the repo's pull.rebase/pull.ff config, defaulting to fast-forward-or-merge
	PullModeFastForwardOnly = "ff-only" // fails rather than creating a merge commit or rebasing
	PullModeRebase          = "rebase"
)

func Pull(repoPath, mode string) error {
	args := []string{"pull"}
	switch mode {
	case PullModeFastForwardOnly:
		args = append(args, "--ff-only")
	case PullModeRebase:
		args = append(args, "--rebase")
	}
	_, err := Run(repoPath, args...)
	return err
}

func PushTag(repoPath, tagName string) error {
	_, err := Run(repoPath, "push", "origin", "refs/tags/"+tagName)
	return err
}

func ForcePushTag(repoPath, tagName string) error {
	_, err := Run(repoPath, "push", "--force", "origin", "refs/tags/"+tagName)
	return err
}

func RemoteHasRef(repoPath, remote, ref string) (bool, error) {
	_, err := Run(repoPath, "ls-remote", "--exit-code", remote, ref)
	if exitErr, ok := err.(*ExitError); ok && exitErr.Code == 2 {
		return false, nil
	}
	return err == nil, err
}

func DeleteRemoteRef(repoPath, remote, ref string) error {
	_, err := Run(repoPath, "push", remote, "--delete", ref)
	return err
}
