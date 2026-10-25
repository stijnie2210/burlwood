package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"burlwood/internal/gitcli"
)

type RebaseState struct {
	Step    int    `json:"step"`
	Total   int    `json:"total"`
	Message string `json:"message"`
	Onto    string `json:"onto"`
}

func (m *Manager) IsInRebase(repoPath string) (bool, error) {
	if _, err := m.Get(repoPath); err != nil {
		return false, err
	}
	gitDir, err := getGitDir(repoPath)
	if err != nil {
		return false, nil
	}
	for _, dir := range []string{"rebase-merge", "rebase-apply"} {
		if info, err := os.Stat(filepath.Join(gitDir, dir)); err == nil && info.IsDir() {
			return true, nil
		}
	}
	return false, nil
}

func (m *Manager) GetRebaseState(repoPath string) (RebaseState, error) {
	if _, err := m.Get(repoPath); err != nil {
		return RebaseState{}, err
	}
	gitDir, err := getGitDir(repoPath)
	if err != nil {
		return RebaseState{}, nil
	}
	for _, sub := range []string{"rebase-merge", "rebase-apply"} {
		dir := filepath.Join(gitDir, sub)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return readRebaseState(dir), nil
		}
	}
	return RebaseState{}, nil
}

func readRebaseState(dir string) RebaseState {
	var s RebaseState

	if data, err := os.ReadFile(filepath.Join(dir, "msgnum")); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &s.Step)
	} else if data, err := os.ReadFile(filepath.Join(dir, "next")); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &s.Step)
	}

	if data, err := os.ReadFile(filepath.Join(dir, "end")); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &s.Total)
	} else if data, err := os.ReadFile(filepath.Join(dir, "last")); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &s.Total)
	}

	if data, err := os.ReadFile(filepath.Join(dir, "message")); err == nil {
		first, _, _ := strings.Cut(string(data), "\n")
		s.Message = strings.TrimSpace(first)
	}

	// Prefer human-readable onto name; fall back to short hash.
	if data, err := os.ReadFile(filepath.Join(dir, "onto_name")); err == nil {
		s.Onto = strings.TrimSpace(string(data))
	} else if data, err := os.ReadFile(filepath.Join(dir, "onto")); err == nil {
		onto := strings.TrimSpace(string(data))
		if len(onto) > 8 {
			onto = onto[:8]
		}
		s.Onto = onto
	}

	return s
}

// conflict-stop (exit 1) is surfaced as an error; callers should check IsInRebase after.
func (m *Manager) StartRebase(repoPath, onto string) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}
	return gitcli.StartRebase(repoPath, onto)
}

func (m *Manager) ContinueRebase(repoPath string) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}
	return gitcli.ContinueRebase(repoPath)
}

func (m *Manager) AbortRebase(repoPath string) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}
	return gitcli.AbortRebase(repoPath)
}
