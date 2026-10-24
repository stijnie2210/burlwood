package gitcli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DiscardSnapshot struct {
	Path     string `json:"path"`
	Blob     string `json:"blob"`
	Mode     uint32 `json:"mode"`
	Existed  bool   `json:"existed"`
	Undoable bool   `json:"undoable"`
}

type ResetUndo struct {
	Head   string `json:"head"`
	Target string `json:"target"`
	Stash  string `json:"stash"`
}

// The content is written into the object store rather than kept in memory, so it
// stays recoverable after an app restart until git gc prunes the dangling blob.
func SnapshotWorkingFile(repoPath, path string) (DiscardSnapshot, error) {
	snap := DiscardSnapshot{Path: path}
	info, err := os.Lstat(filepath.Join(repoPath, path))
	if errors.Is(err, os.ErrNotExist) {
		snap.Undoable = true
		return snap, nil
	}
	if err != nil {
		return snap, err
	}
	if !info.Mode().IsRegular() {
		return snap, nil
	}
	res, err := Run(repoPath, "hash-object", "-w", "--no-filters", "--", path)
	if err != nil {
		return snap, err
	}
	snap.Blob = strings.TrimSpace(res.Stdout)
	snap.Mode = uint32(info.Mode().Perm())
	snap.Existed = true
	snap.Undoable = true
	return snap, nil
}

func RestoreWorkingFile(repoPath string, snap DiscardSnapshot) error {
	if !snap.Undoable {
		return fmt.Errorf("no snapshot was taken for %s", snap.Path)
	}
	full := filepath.Join(repoPath, snap.Path)
	if !snap.Existed {
		if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	res, err := Run(repoPath, "cat-file", "blob", snap.Blob)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(full, []byte(res.Stdout), os.FileMode(snap.Mode)); err != nil {
		return err
	}
	return os.Chmod(full, os.FileMode(snap.Mode))
}

func HardResetWithUndo(repoPath, hash string) (ResetUndo, error) {
	var undo ResetUndo
	head, err := revParse(repoPath, "HEAD")
	if err != nil {
		return undo, err
	}
	target, err := revParse(repoPath, hash+"^{commit}")
	if err != nil {
		return undo, err
	}
	stash, err := Run(repoPath, "stash", "create")
	if err != nil {
		return undo, err
	}
	if _, err := Run(repoPath, "reset", "--hard", target); err != nil {
		return undo, err
	}
	undo.Head = head
	undo.Target = target
	undo.Stash = strings.TrimSpace(stash.Stdout)
	return undo, nil
}

// Refuses when HEAD or the tracked working tree changed after the reset, because
// restoring on top of newer work would destroy that work instead.
func UndoHardReset(repoPath string, undo ResetUndo) error {
	current, err := revParse(repoPath, "HEAD")
	if err != nil {
		return err
	}
	if current != undo.Target {
		return fmt.Errorf("HEAD moved after the reset, so it can't be undone automatically. The previous HEAD was %s", undo.Head)
	}
	status, err := Run(repoPath, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return err
	}
	if strings.TrimSpace(status.Stdout) != "" {
		return fmt.Errorf("files changed after the reset, so it can't be undone automatically. The previous HEAD was %s", undo.Head)
	}
	if _, err := Run(repoPath, "reset", "--hard", undo.Head); err != nil {
		return err
	}
	if undo.Stash == "" {
		return nil
	}
	if _, err := Run(repoPath, "stash", "apply", "--index", undo.Stash); err != nil {
		if _, err := Run(repoPath, "stash", "apply", undo.Stash); err != nil {
			return fmt.Errorf("branch restored, but local changes could not be reapplied. Recover them with: git stash apply %s", undo.Stash)
		}
	}
	return nil
}

func revParse(repoPath, rev string) (string, error) {
	res, err := Run(repoPath, "rev-parse", "--verify", rev)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Stdout), nil
}
