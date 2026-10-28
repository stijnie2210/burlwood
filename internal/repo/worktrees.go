package repo

import (
	"path/filepath"
	"strings"

	"burlwood/internal/gitcli"
)

type WorktreeInfo struct {
	Path       string `json:"path"`
	Head       string `json:"head"`
	Branch     string `json:"branch"` // empty when detached
	IsMain     bool   `json:"isMain"`
	IsCurrent  bool   `json:"isCurrent"`
	Bare       bool   `json:"bare"`
	Locked     bool   `json:"locked"`
	LockReason string `json:"lockReason"`
	Prunable   bool   `json:"prunable"` // its folder is gone; git worktree prune drops the entry
}

func (m *Manager) ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	if _, err := m.Get(repoPath); err != nil {
		return nil, err
	}
	result, err := gitcli.Run(repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	worktrees := parseWorktrees(result.Stdout)
	for i := range worktrees {
		worktrees[i].IsCurrent = samePath(worktrees[i].Path, repoPath)
	}
	return worktrees, nil
}

func parseWorktrees(out string) []WorktreeInfo {
	var worktrees []WorktreeInfo
	for block := range strings.SplitSeq(strings.TrimSpace(out), "\n\n") {
		var wt WorktreeInfo
		for line := range strings.SplitSeq(block, "\n") {
			key, value, _ := strings.Cut(line, " ")
			switch key {
			case "worktree":
				wt.Path = filepath.FromSlash(value)
			case "HEAD":
				wt.Head = value
			case "branch":
				wt.Branch = strings.TrimPrefix(value, "refs/heads/")
			case "bare":
				wt.Bare = true
			case "locked":
				wt.Locked = true
				wt.LockReason = value
			case "prunable":
				wt.Prunable = true
			}
		}
		if wt.Path != "" {
			wt.IsMain = len(worktrees) == 0
			worktrees = append(worktrees, wt)
		}
	}
	return worktrees
}

// Git may print a worktree through a different spelling than the folder the repo was opened
// from (macOS /tmp is /private/tmp).
func samePath(a, b string) bool {
	if a == b {
		return true
	}
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}
