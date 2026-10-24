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
