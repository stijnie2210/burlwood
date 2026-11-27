package repo

import (
	"fmt"
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

type DeletedTag struct {
	Name          string `json:"name"`
	Target        string `json:"target"`
	Remote        string `json:"remote"`
	RemoteDeleted bool   `json:"remoteDeleted"`
}

// The remote goes first, so a network or auth failure leaves the local tag in place.
func (m *Manager) DeleteTag(repoPath, name, remote string) (DeletedTag, error) {
	if _, err := m.Get(repoPath); err != nil {
		return DeletedTag{}, err
	}
	deleted := DeletedTag{Name: name, Target: revParse(repoPath, "refs/tags/"+name), Remote: remote}
	if deleted.Target == "" {
		return DeletedTag{}, fmt.Errorf("tag %q does not exist", name)
	}
	if remote != "" {
		onRemote, err := gitcli.RemoteHasRef(repoPath, remote, "refs/tags/"+name)
		if err != nil {
			return DeletedTag{}, err
		}
		if onRemote {
			if err := gitcli.DeleteRemoteRef(repoPath, remote, "refs/tags/"+name); err != nil {
				return DeletedTag{}, err
			}
			deleted.RemoteDeleted = true
		}
	}
	if _, err := gitcli.Run(repoPath, "tag", "-d", name); err != nil {
		return deleted, err
	}
	return deleted, nil
}

func (m *Manager) RestoreTag(repoPath string, t DeletedTag) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}
	_, err := gitcli.Run(repoPath, "update-ref", "refs/tags/"+t.Name, t.Target, "")
	return err
}
