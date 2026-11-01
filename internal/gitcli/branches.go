package gitcli

func DeleteBranch(repoPath, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := Run(repoPath, "branch", flag, name)
	return err
}

func RenameBranch(repoPath, oldName, newName string) error {
	_, err := Run(repoPath, "branch", "-m", oldName, newName)
	return err
}

func SetUpstream(repoPath, branch, upstream string) error {
	_, err := Run(repoPath, "branch", "--set-upstream-to="+upstream, branch)
	return err
}

func UnsetUpstream(repoPath, branch string) error {
	_, err := Run(repoPath, "branch", "--unset-upstream", branch)
	return err
}

func Merge(repoPath, branch string) error {
	_, err := Run(repoPath, "merge", "--no-edit", branch)
	return err
}
