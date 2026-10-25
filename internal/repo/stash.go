package repo

import "burlwood/internal/gitcli"

func (m *Manager) ListStashes(repoPath string) ([]gitcli.StashEntry, error) {
	return gitcli.ListStashes(repoPath)
}

func (m *Manager) ApplyStash(repoPath, ref string) error {
	return gitcli.ApplyStash(repoPath, ref)
}

func (m *Manager) DropStash(repoPath, ref string) error {
	return gitcli.DropStash(repoPath, ref)
}

func (m *Manager) PopStash(repoPath, ref string) error {
	return gitcli.PopStash(repoPath, ref)
}

func (m *Manager) RenameStash(repoPath, ref, newMessage string) error {
	return gitcli.RenameStash(repoPath, ref, newMessage)
}
