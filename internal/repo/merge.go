package repo

import (
	"strings"

	"burlwood/internal/gitcli"
)

type MergeOutcome string

const (
	MergeUpToDate    MergeOutcome = "up-to-date"
	MergeFastForward MergeOutcome = "fast-forward"
	MergeCommit      MergeOutcome = "merged"
)

type MergeResult struct {
	Outcome MergeOutcome `json:"outcome"`
}

// The outcome comes from comparing HEAD before and after, not from git's output, which is
// translated under a non-English locale. A conflict-stop (exit 1) is returned as an error;
// callers check IsInMerge after.
func (m *Manager) MergeBranch(repoPath, branch string) (MergeResult, error) {
	if _, err := m.Get(repoPath); err != nil {
		return MergeResult{}, err
	}
	before := revParse(repoPath, "HEAD")
	if err := gitcli.Merge(repoPath, branch); err != nil {
		return MergeResult{}, err
	}
	after := revParse(repoPath, "HEAD")
	switch {
	case after == before:
		return MergeResult{Outcome: MergeUpToDate}, nil
	case revParse(repoPath, "HEAD^2") != "":
		return MergeResult{Outcome: MergeCommit}, nil
	default:
		return MergeResult{Outcome: MergeFastForward}, nil
	}
}

func revParse(repoPath, rev string) string {
	out, err := gitcli.Run(repoPath, "rev-parse", "--verify", "--quiet", rev)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out.Stdout)
}
