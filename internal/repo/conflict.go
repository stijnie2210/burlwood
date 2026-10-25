package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"burlwood/internal/gitcli"
)

type ConflictContent struct {
	Base    string `json:"base"`
	Ours    string `json:"ours"`
	Theirs  string `json:"theirs"`
	Working string `json:"working"`
}

// also catches cherry-pick in progress
func (m *Manager) IsInMerge(repoPath string) (bool, error) {
	if _, err := m.Get(repoPath); err != nil {
		return false, err
	}
	gitDir, err := getGitDir(repoPath)
	if err != nil {
		return false, nil
	}
	for _, marker := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD"} {
		if _, err := os.Stat(filepath.Join(gitDir, marker)); err == nil {
			return true, nil
		}
	}
	return false, nil
}

func (m *Manager) GetMergeMessage(repoPath string) (string, error) {
	gitDir, err := getGitDir(repoPath)
	if err != nil {
		return "", nil
	}
	data, err := os.ReadFile(filepath.Join(gitDir, "MERGE_MSG"))
	if err != nil {
		return "", nil
	}
	return string(data), nil
}

func (m *Manager) GetConflictContent(repoPath, path string) (ConflictContent, error) {
	if _, err := m.Get(repoPath); err != nil {
		return ConflictContent{}, err
	}

	var cc ConflictContent

	// base (stage 1) may not exist for AA/DD conflicts
	baseResult, _ := gitcli.Run(repoPath, "show", ":1:"+path)
	cc.Base = baseResult.Stdout

	oursResult, err := gitcli.Run(repoPath, "show", ":2:"+path)
	if err != nil {
		return cc, fmt.Errorf("git show :2:: %w", err)
	}
	cc.Ours = oursResult.Stdout

	theirsResult, err := gitcli.Run(repoPath, "show", ":3:"+path)
	if err != nil {
		return cc, fmt.Errorf("git show :3:: %w", err)
	}
	cc.Theirs = theirsResult.Stdout

	data, err := os.ReadFile(filepath.Join(repoPath, path))
	if err != nil {
		return cc, fmt.Errorf("read working file: %w", err)
	}
	cc.Working = string(data)

	return cc, nil
}

func (m *Manager) ResolveConflict(repoPath, path, content string) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}
	fullPath := filepath.Join(repoPath, path)
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write resolved file: %w", err)
	}
	return gitcli.StageFile(repoPath, path)
}

func (m *Manager) AbortMerge(repoPath string) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}
	_, err := gitcli.Run(repoPath, "merge", "--abort")
	return err
}

func getGitDir(repoPath string) (string, error) {
	result, err := gitcli.Run(repoPath, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	gitDir := strings.TrimSpace(result.Stdout)
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoPath, gitDir)
	}
	return gitDir, nil
}
