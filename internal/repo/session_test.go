package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoGone(t *testing.T) {
	base := t.TempDir()
	repoDir := filepath.Join(base, "repo")
	worktree := filepath.Join(base, "worktree")
	plain := filepath.Join(base, "plain")
	for _, dir := range []string{filepath.Join(repoDir, ".git"), worktree, plain} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: elsewhere"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := map[string]bool{
		repoDir:                                  false,
		worktree:                                 false,
		plain:                                    true,
		filepath.Join(base, "missing"):           true,
		filepath.Join(base, "unmounted", "repo"): false,
	}
	for path, want := range cases {
		if got := RepoGone(path); got != want {
			t.Errorf("RepoGone(%s) = %v, want %v", filepath.Base(path), got, want)
		}
	}
}

func TestListRecentReposDropsGoneRepos(t *testing.T) {
	configDirOnce.Do(func() {})
	configDir = t.TempDir()

	base := t.TempDir()
	kept := filepath.Join(base, "kept")
	if err := os.MkdirAll(filepath.Join(kept, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeRecentRepos([]recentEntry{
		{Path: kept, Name: "kept"},
		{Path: filepath.Join(base, "deleted"), Name: "deleted"},
	}); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		entries, err := ListRecentRepos()
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Path != kept {
			t.Fatalf("got %v, want only %s", entries, kept)
		}
	}
	saved, _ := LoadRecentRepos()
	if len(saved) != 1 {
		t.Fatalf("repos.json still lists %v", saved)
	}
}
