package repo

import (
	"errors"
	"strings"

	"burlwood/internal/gitcli"
)

type CherryPickOutcome string

const (
	CherryPickCommitted CherryPickOutcome = "committed"
	CherryPickStaged    CherryPickOutcome = "staged"
	CherryPickEmpty     CherryPickOutcome = "empty"
	CherryPickConflict  CherryPickOutcome = "conflict"
)

type CherryPickResult struct {
	Outcome CherryPickOutcome `json:"outcome"`
}

func (m *Manager) CherryPick(repoPath, hash string, commitImmediately bool) (CherryPickResult, error) {
	if _, err := m.Get(repoPath); err != nil {
		return CherryPickResult{}, err
	}
	if hasUnmergedFiles(repoPath) {
		return CherryPickResult{}, errors.New("resolve the conflicted files first")
	}
	before := revParse(repoPath, "HEAD")
	pickErr := gitcli.CherryPick(repoPath, hash, commitImmediately)
	if pickErr != nil {
		if hasUnmergedFiles(repoPath) {
			return CherryPickResult{Outcome: CherryPickConflict}, nil
		}
		// git stops on a pick whose changes are already on the branch and leaves it in
		// progress; there is nothing to resolve, so skip it instead of stranding the user.
		if kind, _ := m.GetMergeKind(repoPath); kind == MergeKindCherryPick && indexMatchesHead(repoPath) {
			if _, err := gitcli.Run(repoPath, "cherry-pick", "--skip"); err != nil {
				return CherryPickResult{}, err
			}
			return CherryPickResult{Outcome: CherryPickEmpty}, nil
		}
		return CherryPickResult{}, pickErr
	}
	if commitImmediately {
		if revParse(repoPath, "HEAD") == before {
			return CherryPickResult{Outcome: CherryPickEmpty}, nil
		}
		return CherryPickResult{Outcome: CherryPickCommitted}, nil
	}
	if indexMatchesHead(repoPath) {
		return CherryPickResult{Outcome: CherryPickEmpty}, nil
	}
	return CherryPickResult{Outcome: CherryPickStaged}, nil
}

func hasUnmergedFiles(repoPath string) bool {
	out, err := gitcli.Run(repoPath, "diff", "--name-only", "--diff-filter=U")
	return err == nil && strings.TrimSpace(out.Stdout) != ""
}

func indexMatchesHead(repoPath string) bool {
	_, err := gitcli.Run(repoPath, "diff", "--cached", "--quiet")
	return err == nil
}
