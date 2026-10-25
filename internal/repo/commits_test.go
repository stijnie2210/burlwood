package repo

import (
	"os"
	"path/filepath"
	"testing"

	"burlwood/internal/gitcli"
)

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitcli.Run(dir, args...)
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return out.Stdout
}

func writeRepoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "config", "user.email", "test@example.com")
	gitIn(t, dir, "config", "user.name", "Test")
	writeRepoFile(t, dir, "a.txt", "one\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "init")
	return dir
}

// Stashes have two or three parents; each changed file must appear once.
func TestCommitDiff_StashListsEachFileOnce(t *testing.T) {
	dir := newRepo(t)
	writeRepoFile(t, dir, "a.txt", "two\n")
	gitIn(t, dir, "add", "a.txt")
	writeRepoFile(t, dir, "a.txt", "three\n")
	gitIn(t, dir, "stash", "-q")
	stash := gitIn(t, dir, "rev-parse", "stash@{0}")

	diffs, err := getCommitDiffShell(dir, stash[:len(stash)-1])
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].NewPath != "a.txt" {
		t.Fatalf("want a.txt once, got %+v", diffs)
	}
}

func TestCommitDiff_MergeShowsFirstParentChanges(t *testing.T) {
	dir := newRepo(t)
	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	writeRepoFile(t, dir, "b.txt", "feature\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "feature")
	gitIn(t, dir, "checkout", "-q", "main")
	writeRepoFile(t, dir, "c.txt", "main\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "main work")
	gitIn(t, dir, "merge", "-q", "--no-edit", "feature")
	merge := gitIn(t, dir, "rev-parse", "HEAD")

	diffs, err := getCommitDiffShell(dir, merge[:len(merge)-1])
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].NewPath != "b.txt" {
		t.Fatalf("want only b.txt (what the merge brought in), got %+v", diffs)
	}
}

func TestCommitGraph_HidesRemoteHEADLabel(t *testing.T) {
	dir := newRepo(t)
	gitIn(t, dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	gitIn(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	m := NewManager()
	if err := m.Open(dir); err != nil {
		t.Fatal(err)
	}
	rows, err := m.GetCommitGraph(dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, l := range rows[0].Labels {
		names = append(names, l.Remote+"/"+l.Name)
	}
	for _, n := range names {
		if n == "origin/HEAD" {
			t.Fatalf("origin/HEAD should be hidden, got %v", names)
		}
	}
	if len(names) != 2 {
		t.Fatalf("want main and origin/main, got %v", names)
	}
}
