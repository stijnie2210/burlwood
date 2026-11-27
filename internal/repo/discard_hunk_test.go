package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func numberedLines(n int, change map[int]string) string {
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		if c, ok := change[i]; ok {
			sb.WriteString(c + "\n")
			continue
		}
		fmt.Fprintf(&sb, "line %d\n", i)
	}
	return sb.String()
}

func readRepoFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func firstHunkPatch(t *testing.T, m *Manager, dir, path string) string {
	t.Helper()
	diffs, err := m.GetWorkingDiff(dir, path, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || len(diffs[0].Hunks) == 0 {
		t.Fatalf("want one file with hunks, got %+v", diffs)
	}
	return BuildHunkPatch(path, diffs[0].Hunks[0])
}

func TestDiscardHunk_KeepsOtherHunksAndCanBeUndone(t *testing.T) {
	dir := newRepo(t)
	writeRepoFile(t, dir, "f.txt", numberedLines(30, nil))
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "f")
	edited := numberedLines(30, map[int]string{2: "changed 2", 28: "changed 28"})
	writeRepoFile(t, dir, "f.txt", edited)
	m := openRepo(t, dir)

	snap, err := m.DiscardHunk(dir, "f.txt", firstHunkPatch(t, m, dir, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := readRepoFile(t, dir, "f.txt"), numberedLines(30, map[int]string{28: "changed 28"}); got != want {
		t.Fatalf("only the first hunk should be discarded, got:\n%s", got)
	}
	if staged := gitIn(t, dir, "diff", "--cached", "--name-only"); staged != "" {
		t.Fatalf("the index should be untouched, got %q staged", staged)
	}

	if err := m.UndoDiscard(dir, snap); err != nil {
		t.Fatal(err)
	}
	if got := readRepoFile(t, dir, "f.txt"); got != edited {
		t.Fatalf("undo should bring the edit back, got:\n%s", got)
	}
}

func TestDiscardHunk_LeavesStagedChangesAlone(t *testing.T) {
	dir := newRepo(t)
	writeRepoFile(t, dir, "f.txt", numberedLines(30, nil))
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "f")
	writeRepoFile(t, dir, "f.txt", numberedLines(30, map[int]string{2: "staged 2"}))
	gitIn(t, dir, "add", "f.txt")
	writeRepoFile(t, dir, "f.txt", numberedLines(30, map[int]string{2: "staged 2", 28: "unstaged 28"}))
	m := openRepo(t, dir)

	if _, err := m.DiscardHunk(dir, "f.txt", firstHunkPatch(t, m, dir, "f.txt")); err != nil {
		t.Fatal(err)
	}
	if got, want := readRepoFile(t, dir, "f.txt"), numberedLines(30, map[int]string{2: "staged 2"}); got != want {
		t.Fatalf("want the staged version back, got:\n%s", got)
	}
}

func TestDiscardHunk_AtEndOfFileWithoutNewline(t *testing.T) {
	dir := newRepo(t)
	writeRepoFile(t, dir, "f.txt", "one\ntwo\nthree")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "f")
	writeRepoFile(t, dir, "f.txt", "one\ntwo\nTHREE")
	m := openRepo(t, dir)

	if _, err := m.DiscardHunk(dir, "f.txt", firstHunkPatch(t, m, dir, "f.txt")); err != nil {
		t.Fatal(err)
	}
	if got := readRepoFile(t, dir, "f.txt"); got != "one\ntwo\nthree" {
		t.Fatalf("got %q", got)
	}
}

func TestStageHunk_AtEndOfFileWithoutNewline(t *testing.T) {
	dir := newRepo(t)
	writeRepoFile(t, dir, "f.txt", "one\ntwo\nthree")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "f")
	writeRepoFile(t, dir, "f.txt", "one\ntwo\nTHREE")
	m := openRepo(t, dir)

	if err := m.StageHunk(dir, firstHunkPatch(t, m, dir, "f.txt")); err != nil {
		t.Fatal(err)
	}
	if staged := gitIn(t, dir, "show", ":f.txt"); staged != "one\ntwo\nTHREE" {
		t.Fatalf("got %q staged", staged)
	}
}
