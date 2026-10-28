package repo

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseWorktrees(t *testing.T) {
	out := "worktree /repo\nHEAD aaaa\nbranch refs/heads/main\n\n" +
		"worktree /repo/.claude/worktrees/fix\nHEAD bbbb\ndetached\nlocked claude session\n\n" +
		"worktree /gone\nHEAD cccc\nbranch refs/heads/old\nprunable gitdir file points to non-existent location\n"
	want := []WorktreeInfo{
		{Path: filepath.FromSlash("/repo"), Head: "aaaa", Branch: "main", IsMain: true},
		{Path: filepath.FromSlash("/repo/.claude/worktrees/fix"), Head: "bbbb", Locked: true, LockReason: "claude session"},
		{Path: filepath.FromSlash("/gone"), Head: "cccc", Branch: "old", Prunable: true},
	}
	if got := parseWorktrees(out); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseWorktrees = %#v, want %#v", got, want)
	}
}

// t.TempDir() sits behind a symlink on macOS, so this also covers git printing the resolved path.
func TestWorktreesAndBranchesFromLinkedWorktree(t *testing.T) {
	dir := newRepo(t)
	wt := filepath.Join(t.TempDir(), "feature-wt")
	gitIn(t, dir, "worktree", "add", "-q", "-b", "feature", wt)

	m := NewManager()
	if err := m.Open(dir); err != nil {
		t.Fatal(err)
	}
	if err := m.Open(wt); err != nil {
		t.Fatal(err)
	}

	worktrees, err := m.ListWorktrees(wt)
	if err != nil {
		t.Fatal(err)
	}
	if len(worktrees) != 2 || !worktrees[0].IsMain || worktrees[0].IsCurrent || !worktrees[1].IsCurrent {
		t.Fatalf("from the linked worktree: %#v", worktrees)
	}

	branches, err := m.ListBranches(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range branches {
		switch b.Name {
		case "main":
			if b.Worktree != "" {
				t.Errorf("main is checked out here, got worktree %q", b.Worktree)
			}
		case "feature":
			if !samePath(b.Worktree, wt) {
				t.Errorf("feature: worktree %q, want %q", b.Worktree, wt)
			}
		}
	}
}
