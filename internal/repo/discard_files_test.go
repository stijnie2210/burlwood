package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"burlwood/internal/gitcli"
)

func TestDiscardFiles_AllAtOnceAndUndo(t *testing.T) {
	dir := newRepo(t)
	writeRepoFile(t, dir, "b.txt", "b\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "b")
	writeRepoFile(t, dir, "a.txt", "edited\n")
	writeRepoFile(t, dir, "b.txt", "staged\n")
	gitIn(t, dir, "add", "b.txt")
	writeRepoFile(t, dir, "b.txt", "staged, then edited\n")
	writeRepoFile(t, dir, "new.txt", "untracked\n")
	m := openRepo(t, dir)

	snaps, err := m.DiscardFiles(dir, []string{"a.txt", "b.txt", "new.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if got := readRepoFile(t, dir, "a.txt"); got != "one\n" {
		t.Fatalf("a.txt should be back at HEAD, got %q", got)
	}
	if got := readRepoFile(t, dir, "b.txt"); got != "staged\n" {
		t.Fatalf("b.txt should keep its staged version, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.txt")); !os.IsNotExist(err) {
		t.Fatal("the untracked file should be gone")
	}

	if err := m.UndoDiscards(dir, snaps); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"a.txt": "edited\n", "b.txt": "staged, then edited\n", "new.txt": "untracked\n"} {
		if got := readRepoFile(t, dir, name); got != want {
			t.Fatalf("undo: %s = %q, want %q", name, got, want)
		}
	}
}

func TestCommitMessage_IncludesTheBody(t *testing.T) {
	dir := newRepo(t)
	writeRepoFile(t, dir, "a.txt", "two\n")
	gitIn(t, dir, "commit", "-q", "-am", "subject line", "-m", "body paragraph")
	hash := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD"))

	msg, err := gitcli.CommitMessage(dir, hash)
	if err != nil {
		t.Fatal(err)
	}
	if msg != "subject line\n\nbody paragraph" {
		t.Fatalf("got %q", msg)
	}
}
