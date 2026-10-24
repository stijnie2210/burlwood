package gitcli

import (
	"os"
	"path/filepath"
	"strings"
)

func StageFile(repoPath, path string) error {
	_, err := Run(repoPath, "add", "--", path)
	return err
}

func UnstageFile(repoPath, path string) error {
	_, err := Run(repoPath, "restore", "--staged", "--", path)
	return err
}

// Paths go through stdin so large selections don't hit the argument-length limit.
func StageFiles(repoPath string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	_, err := RunWithInput(repoPath, nulJoin(paths), "add", "--pathspec-from-file=-", "--pathspec-file-nul")
	return err
}

func UnstageFiles(repoPath string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	_, err := RunWithInput(repoPath, nulJoin(paths), "restore", "--staged", "--pathspec-from-file=-", "--pathspec-file-nul")
	return err
}

func nulJoin(paths []string) []byte {
	return []byte(strings.Join(paths, "\x00"))
}

func ApplyPatch(repoPath string, patch []byte, reverse bool) error {
	args := []string{"apply", "--cached"}
	if reverse {
		args = append(args, "--reverse")
	}
	_, err := RunWithInput(repoPath, patch, args...)
	return err
}

func DiscardFile(repoPath, path string) error {
	_, err := Run(repoPath, "restore", "--", path)
	if err != nil {
		if exitErr, ok := err.(*ExitError); ok && strings.Contains(exitErr.Stderr, "did not match any file") {
			return os.Remove(filepath.Join(repoPath, path))
		}
		return err
	}
	return nil
}
