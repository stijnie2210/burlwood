package repo

import (
	"strings"

	"burlwood/internal/gitcli"
)

type TagInfo struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

func (m *Manager) ListTags(repoPath string) ([]TagInfo, error) {
	if _, err := m.Get(repoPath); err != nil {
		return nil, err
	}
	result, err := gitcli.Run(repoPath, "for-each-ref", "--sort=-creatordate",
		"--format=%(refname:short)\t%(objectname)\t%(*objectname)", "refs/tags/")
	if err != nil {
		return nil, err
	}
	return parseTags(result.Stdout), nil
}

// An annotated tag points at a tag object; %(*objectname) is the commit it peels to.
func parseTags(out string) []TagInfo {
	tags := []TagInfo{}
	for line := range strings.SplitSeq(strings.Trim(out, "\n"), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) != 3 || parts[0] == "" {
			continue
		}
		hash := parts[1]
		if parts[2] != "" {
			hash = parts[2]
		}
		tags = append(tags, TagInfo{Name: parts[0], Hash: hash})
	}
	return tags
}
