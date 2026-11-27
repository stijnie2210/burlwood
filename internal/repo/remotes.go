package repo

import (
	"sort"
	"strings"

	"burlwood/internal/gitcli"
)

func (m *Manager) FetchRemote(repoPath, remote string) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}
	prefs, _ := LoadPrefs()
	return gitcli.FetchRemote(repoPath, remote, prefs.PruneOnFetch)
}

// The pruned names come from comparing the remote's refs before and after, not from git's
// output, which is translated under a non-English locale.
func (m *Manager) PruneRemote(repoPath, remote string) ([]string, error) {
	if _, err := m.Get(repoPath); err != nil {
		return nil, err
	}
	before, err := remoteBranchNames(repoPath, remote)
	if err != nil {
		return nil, err
	}
	if _, err := gitcli.Run(repoPath, "remote", "prune", remote); err != nil {
		return nil, err
	}
	after, err := remoteBranchNames(repoPath, remote)
	if err != nil {
		return nil, err
	}
	pruned := []string{}
	for name := range before {
		if !after[name] {
			pruned = append(pruned, name)
		}
	}
	sort.Strings(pruned)
	return pruned, nil
}

func remoteBranchNames(repoPath, remote string) (map[string]bool, error) {
	prefix := "refs/remotes/" + remote + "/"
	out, err := gitcli.Run(repoPath, "for-each-ref", "--format=%(refname)", prefix)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for line := range strings.SplitSeq(strings.TrimSpace(out.Stdout), "\n") {
		if name := strings.TrimPrefix(line, prefix); line != "" && name != "HEAD" {
			names[name] = true
		}
	}
	return names, nil
}
