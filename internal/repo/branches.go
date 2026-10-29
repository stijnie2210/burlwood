package repo

import (
	"path/filepath"
	"strconv"
	"strings"

	"burlwood/internal/gitcli"
)

type BranchInfo struct {
	Name      string `json:"name"`
	IsRemote  bool   `json:"isRemote"`
	Remote    string `json:"remote"` // e.g. "origin" — empty for local branches
	IsCurrent bool   `json:"isCurrent"`
	Hash      string `json:"hash"`
	Worktree  string `json:"worktree"` // another worktree that has this branch checked out
}

type AheadBehind struct {
	Ahead  int `json:"ahead"`
	Behind int `json:"behind"`
}

// returns zeros silently when no upstream is configured or HEAD is detached
func (m *Manager) GetAheadBehind(repoPath string) (AheadBehind, error) {
	result, err := gitcli.Run(repoPath, "rev-list", "--left-right", "--count", "HEAD...@{u}")
	if err != nil {
		return AheadBehind{}, nil
	}
	parts := strings.Fields(strings.TrimSpace(result.Stdout))
	if len(parts) != 2 {
		return AheadBehind{}, nil
	}
	ahead, _ := strconv.Atoi(parts[0])
	behind, _ := strconv.Atoi(parts[1])
	return AheadBehind{Ahead: ahead, Behind: behind}, nil
}

func (m *Manager) ListBranches(repoPath string) ([]BranchInfo, error) {
	if _, err := m.Get(repoPath); err != nil {
		return nil, err
	}

	// symbolic-ref reads .git/HEAD directly, never stale
	headResult, _ := gitcli.Run(repoPath, "symbolic-ref", "--short", "HEAD")
	currentBranch := strings.TrimSpace(headResult.Stdout)

	result, err := gitcli.Run(repoPath, "for-each-ref",
		"--format=%(refname)\t%(objectname)\t%(upstream:short)\t%(worktreepath)",
		"refs/heads/", "refs/remotes/")
	if err != nil {
		return nil, err
	}

	var branches []BranchInfo
	for line := range strings.SplitSeq(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 2 {
			continue
		}
		refname := parts[0]
		hash := parts[1]

		switch {
		case strings.HasPrefix(refname, "refs/heads/"):
			name := strings.TrimPrefix(refname, "refs/heads/")
			var worktree string
			if len(parts) == 4 && parts[3] != "" && !samePath(parts[3], repoPath) {
				worktree = filepath.FromSlash(parts[3])
			}
			branches = append(branches, BranchInfo{
				Name:      name,
				IsRemote:  false,
				IsCurrent: name == currentBranch,
				Hash:      hash,
				Worktree:  worktree,
			})
		case strings.HasPrefix(refname, "refs/remotes/"):
			short := strings.TrimPrefix(refname, "refs/remotes/")
			// skip HEAD symrefs like origin/HEAD
			if strings.HasSuffix(short, "/HEAD") {
				continue
			}
			rparts := strings.SplitN(short, "/", 2)
			b := BranchInfo{IsRemote: true, Hash: hash}
			if len(rparts) == 2 {
				b.Remote = rparts[0]
				b.Name = rparts[1]
			} else {
				b.Name = short
			}
			branches = append(branches, b)
		}
	}

	return branches, nil
}
