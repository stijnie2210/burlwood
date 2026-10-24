package repo

import (
	"fmt"
	"strings"

	"gitgood/internal/gitcli"
)

func (m *Manager) StageFile(repoPath, path string) error {
	return gitcli.StageFile(repoPath, path)
}

func (m *Manager) UnstageFile(repoPath, path string) error {
	return gitcli.UnstageFile(repoPath, path)
}

func (m *Manager) StageFiles(repoPath string, paths []string) error {
	return gitcli.StageFiles(repoPath, paths)
}

func (m *Manager) UnstageFiles(repoPath string, paths []string) error {
	return gitcli.UnstageFiles(repoPath, paths)
}

func (m *Manager) DiscardFile(repoPath, path string) (gitcli.DiscardSnapshot, error) {
	snap, err := gitcli.SnapshotWorkingFile(repoPath, path)
	if err != nil {
		return snap, err
	}
	return snap, gitcli.DiscardFile(repoPath, path)
}

func (m *Manager) UndoDiscard(repoPath string, snap gitcli.DiscardSnapshot) error {
	return gitcli.RestoreWorkingFile(repoPath, snap)
}

// patch is built by the frontend from the displayed hunk data
func (m *Manager) StageHunk(repoPath, patch string) error {
	return gitcli.ApplyPatch(repoPath, []byte(patch), false)
}

func (m *Manager) UnstageHunk(repoPath, patch string) error {
	return gitcli.ApplyPatch(repoPath, []byte(patch), true)
}

func (m *Manager) FetchAll(repoPath string) error {
	return gitcli.FetchAll(repoPath)
}

func (m *Manager) PushBranch(repoPath string) error {
	return gitcli.Push(repoPath)
}

func (m *Manager) Stash(repoPath string) error {
	return gitcli.Stash(repoPath)
}

func (m *Manager) StashPop(repoPath string) error {
	return gitcli.StashPop(repoPath)
}

func (m *Manager) CreateBranch(repoPath, name string) error {
	return gitcli.CreateBranch(repoPath, name)
}

func (m *Manager) Commit(repoPath, message string, amend bool) error {
	return gitcli.Commit(repoPath, message, amend)
}

func (m *Manager) GetLastCommitSubject(repoPath string) (string, error) {
	result, err := gitcli.Run(repoPath, "log", "-1", "--pretty=%s")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(result.Stdout), nil
}

func BuildHunkPatch(path string, hunk Hunk) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "diff --git a/%s b/%s\n", path, path)
	fmt.Fprintf(&sb, "--- a/%s\n", path)
	fmt.Fprintf(&sb, "+++ b/%s\n", path)
	fmt.Fprintf(&sb, "%s\n", hunk.Header)
	for _, l := range hunk.Lines {
		switch l.Type {
		case "add":
			fmt.Fprintf(&sb, "+%s\n", l.Content)
		case "del":
			fmt.Fprintf(&sb, "-%s\n", l.Content)
		default:
			fmt.Fprintf(&sb, " %s\n", l.Content)
		}
	}
	return sb.String()
}
