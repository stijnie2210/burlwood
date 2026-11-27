package repo

import (
	"fmt"

	"burlwood/internal/gitcli"
)

type DeletedRemoteBranch struct {
	Remote      string `json:"remote"`
	Name        string `json:"name"`
	Hash        string `json:"hash"`
	AlreadyGone bool   `json:"alreadyGone"`
}

// A branch someone else already deleted only lingers as a stale remote-tracking ref; that ref
// is removed instead of failing on git's "remote ref does not exist".
func (m *Manager) DeleteRemoteBranch(repoPath, remote, name string) (DeletedRemoteBranch, error) {
	if _, err := m.Get(repoPath); err != nil {
		return DeletedRemoteBranch{}, err
	}
	deleted := DeletedRemoteBranch{
		Remote: remote,
		Name:   name,
		Hash:   revParse(repoPath, "refs/remotes/"+remote+"/"+name),
	}
	if deleted.Hash == "" {
		return DeletedRemoteBranch{}, fmt.Errorf("%s/%s does not exist", remote, name)
	}
	onRemote, err := gitcli.RemoteHasRef(repoPath, remote, "refs/heads/"+name)
	if err != nil {
		return DeletedRemoteBranch{}, err
	}
	if !onRemote {
		deleted.AlreadyGone = true
		_, err := gitcli.Run(repoPath, "update-ref", "-d", "refs/remotes/"+remote+"/"+name)
		return deleted, err
	}
	return deleted, gitcli.DeleteRemoteRef(repoPath, remote, "refs/heads/"+name)
}

func (m *Manager) RestoreRemoteBranch(repoPath string, b DeletedRemoteBranch) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}
	_, err := gitcli.Run(repoPath, "push", b.Remote, b.Hash+":refs/heads/"+b.Name)
	return err
}
