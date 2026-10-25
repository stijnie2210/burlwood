package repo

import (
	"strings"
	"sync"

	"burlwood/internal/gitcli"
)

const (
	autostashMessage = "burlwood-autostash"
	// Written before the rename; still matched so a pull left conflicted across the update gets its stash back.
	legacyAutostashMessage = "gitgood-autostash"
)

type autostashPhase int

const (
	autostashNone autostashPhase = iota
	// stash pushed, but the pull itself left a merge/rebase conflict; pop once that's concluded
	autostashAwaitConclusion
	// the pop attempt itself conflicted; drop the (now fully-applied) stash once those conflicts clear
	autostashAwaitCleanup
)

func (m *Manager) getAutostashPhase(repoPath string) autostashPhase {
	m.autostashMu.Lock()
	defer m.autostashMu.Unlock()
	return m.autostash[repoPath]
}

func (m *Manager) setAutostashPhase(repoPath string, phase autostashPhase) {
	m.autostashMu.Lock()
	defer m.autostashMu.Unlock()
	if phase == autostashNone {
		delete(m.autostash, repoPath)
	} else {
		m.autostash[repoPath] = phase
	}
}

// Serializes PullBranch/SyncAutostash per repo so concurrent load() calls can't both act on the same pending stash.
func (m *Manager) autostashOpLock(repoPath string) *sync.Mutex {
	m.autostashMu.Lock()
	defer m.autostashMu.Unlock()
	lk, ok := m.autostashLocks[repoPath]
	if !ok {
		lk = &sync.Mutex{}
		m.autostashLocks[repoPath] = lk
	}
	return lk
}

// If the pull conflicts, the stash is left pending; SyncAutostash resolves it once the merge/rebase concludes.
func (m *Manager) PullBranch(repoPath, mode string) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}

	lk := m.autostashOpLock(repoPath)
	lk.Lock()
	defer lk.Unlock()

	stashResult, err := gitcli.Run(repoPath, "stash", "push", "-m", autostashMessage)
	if err != nil {
		return err
	}
	stashed := !strings.Contains(stashResult.Stdout, "No local changes to save")

	pullErr := gitcli.Pull(repoPath, mode)

	if !stashed {
		return pullErr
	}

	if pullErr != nil {
		m.setAutostashPhase(repoPath, autostashAwaitConclusion)
		return pullErr
	}

	return m.popAutostash(repoPath)
}

// Cheap no-op unless a PullBranch call left an autostash pending for this repo.
func (m *Manager) SyncAutostash(repoPath string) error {
	lk := m.autostashOpLock(repoPath)
	lk.Lock()
	defer lk.Unlock()

	switch m.getAutostashPhase(repoPath) {
	case autostashNone:
		return nil

	case autostashAwaitConclusion:
		inMerge, err := m.IsInMerge(repoPath)
		if err != nil {
			return err
		}
		inRebase, err := m.IsInRebase(repoPath)
		if err != nil {
			return err
		}
		if inMerge || inRebase {
			return nil
		}
		return m.popAutostash(repoPath)

	default: // autostashAwaitCleanup
		conflicted, err := m.hasConflictedFiles(repoPath)
		if err != nil {
			return err
		}
		if conflicted {
			return nil
		}
		return m.dropAutostash(repoPath)
	}
}

// Re-derives autostash phase from on-disk git state, since the in-memory map is otherwise lost on restart.
func (m *Manager) recoverAutostashPhase(repoPath string) {
	phase, err := m.deriveAutostashPhase(repoPath)
	if err != nil {
		return
	}
	m.setAutostashPhase(repoPath, phase)
}

func (m *Manager) deriveAutostashPhase(repoPath string) (autostashPhase, error) {
	_, ok, err := topAutostashRef(repoPath)
	if err != nil {
		return autostashNone, err
	}
	if !ok {
		return autostashNone, nil
	}

	inMerge, err := m.IsInMerge(repoPath)
	if err != nil {
		return autostashNone, err
	}
	inRebase, err := m.IsInRebase(repoPath)
	if err != nil {
		return autostashNone, err
	}
	if inMerge || inRebase {
		return autostashAwaitConclusion, nil
	}

	conflicted, err := m.hasConflictedFiles(repoPath)
	if err != nil {
		return autostashNone, err
	}
	if conflicted {
		return autostashAwaitCleanup, nil
	}

	// tagged stash exists with no merge/rebase/conflict — app likely closed right before the pop ran
	return autostashAwaitConclusion, nil
}

func (m *Manager) popAutostash(repoPath string) error {
	ref, ok, err := topAutostashRef(repoPath)
	if err != nil {
		return err
	}
	if !ok {
		m.setAutostashPhase(repoPath, autostashNone)
		return nil
	}

	if _, popErr := gitcli.Run(repoPath, "stash", "pop", ref); popErr != nil {
		if _, stillThere, _ := topAutostashRef(repoPath); stillThere {
			m.setAutostashPhase(repoPath, autostashAwaitCleanup)
			return nil
		}
		return popErr
	}

	m.setAutostashPhase(repoPath, autostashNone)
	return nil
}

func (m *Manager) dropAutostash(repoPath string) error {
	ref, ok, err := topAutostashRef(repoPath)
	if err != nil {
		return err
	}
	if !ok {
		m.setAutostashPhase(repoPath, autostashNone)
		return nil
	}
	if _, err := gitcli.Run(repoPath, "stash", "drop", ref); err != nil {
		return err
	}
	m.setAutostashPhase(repoPath, autostashNone)
	return nil
}

// Finds the newest stash entry tagged with autostashMessage, skipping any unrelated stashes above it.
func topAutostashRef(repoPath string) (string, bool, error) {
	result, err := gitcli.Run(repoPath, "stash", "list", "--format=%gd\t%gs")
	if err != nil {
		return "", false, err
	}
	trimmed := strings.TrimSpace(result.Stdout)
	if trimmed == "" {
		return "", false, nil
	}
	for line := range strings.SplitSeq(trimmed, "\n") {
		ref, msg, ok := strings.Cut(line, "\t")
		if ok && (strings.Contains(msg, autostashMessage) || strings.Contains(msg, legacyAutostashMessage)) {
			return ref, true, nil
		}
	}
	return "", false, nil
}

func (m *Manager) hasConflictedFiles(repoPath string) (bool, error) {
	files, err := m.GetStatus(repoPath)
	if err != nil {
		return false, err
	}
	for _, f := range files {
		if IsConflictedStatus(f.Staged, f.Unstaged) {
			return true, nil
		}
	}
	return false, nil
}
