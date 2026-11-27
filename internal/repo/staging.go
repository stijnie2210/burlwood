package repo

import (
	"fmt"
	"strings"

	"burlwood/internal/gitcli"
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

func (m *Manager) DiscardHunk(repoPath, path, patch string) (gitcli.DiscardSnapshot, error) {
	snap, err := gitcli.SnapshotWorkingFile(repoPath, path)
	if err != nil {
		return snap, err
	}
	return snap, gitcli.ReverseApplyToWorkingTree(repoPath, []byte(patch))
}

// Every file is snapshotted before any is touched, so Undo covers the whole batch even when
// a later discard fails.
func (m *Manager) DiscardFiles(repoPath string, paths []string) ([]gitcli.DiscardSnapshot, error) {
	snaps := make([]gitcli.DiscardSnapshot, 0, len(paths))
	for _, path := range paths {
		snap, err := gitcli.SnapshotWorkingFile(repoPath, path)
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, snap)
	}
	for _, path := range paths {
		if err := gitcli.DiscardFile(repoPath, path); err != nil {
			return snaps, err
		}
	}
	return snaps, nil
}

func (m *Manager) UndoDiscards(repoPath string, snaps []gitcli.DiscardSnapshot) error {
	for _, snap := range snaps {
		if err := gitcli.RestoreWorkingFile(repoPath, snap); err != nil {
			return err
		}
	}
	return nil
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
	prefs, _ := LoadPrefs()
	return gitcli.FetchAll(repoPath, prefs.PruneOnFetch)
}

func (m *Manager) Stash(repoPath string) error {
	prefs, _ := LoadPrefs()
	return gitcli.Stash(repoPath, prefs.StashIncludeUntracked)
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

func (m *Manager) GetAmendInfo(repoPath string) (gitcli.AmendInfo, error) {
	return gitcli.GetAmendInfo(repoPath)
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
		if l.NoNewline {
			sb.WriteString("\\ No newline at end of file\n")
		}
	}
	return sb.String()
}
