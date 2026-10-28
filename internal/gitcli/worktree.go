package gitcli

func RemoveWorktree(repoPath, path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = []string{"worktree", "remove", "--force", path}
	}
	_, err := Run(repoPath, args...)
	return err
}

func PruneWorktrees(repoPath string) error {
	_, err := Run(repoPath, "worktree", "prune")
	return err
}
