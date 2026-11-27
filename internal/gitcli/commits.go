package gitcli

import (
	"fmt"
	"strings"
)

func CheckoutRef(repoPath, ref string) error {
	_, err := Run(repoPath, "checkout", ref)
	return err
}

func CreateBranchAt(repoPath, name, hash string) error {
	_, err := Run(repoPath, "checkout", "-b", name, hash)
	return err
}

func RevertCommit(repoPath, hash string, commitImmediately bool) error {
	args := []string{"revert"}
	if commitImmediately {
		args = append(args, "--no-edit")
	} else {
		args = append(args, "--no-commit")
	}
	_, err := Run(repoPath, append(args, hash)...)
	return err
}

func CherryPick(repoPath, hash string, commitImmediately bool) error {
	args := []string{"cherry-pick"}
	if !commitImmediately {
		args = append(args, "--no-commit")
	}
	_, err := Run(repoPath, append(args, hash)...)
	return err
}

func CreateTag(repoPath, name, hash string) error {
	_, err := Run(repoPath, "tag", name, hash)
	return err
}

func MoveTag(repoPath, name, hash string) error {
	if _, err := Run(repoPath, "rev-parse", "refs/tags/"+name); err != nil {
		return fmt.Errorf("tag %q does not exist", name)
	}
	_, err := Run(repoPath, "tag", "-f", name, hash)
	return err
}

func ResetBranch(repoPath, hash, mode string) error {
	switch mode {
	case "soft", "mixed", "hard":
	default:
		return fmt.Errorf("invalid reset mode: %s", mode)
	}
	_, err := Run(repoPath, "reset", "--"+mode, hash)
	return err
}

func CommitMessage(repoPath, hash string) (string, error) {
	out, err := Run(repoPath, "log", "-1", "--format=%B", hash)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Stdout), nil
}
