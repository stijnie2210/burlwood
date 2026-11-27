package repo

import (
	"strings"
	"testing"
)

func historyRepo(t *testing.T) string {
	t.Helper()
	dir := newRepo(t)
	writeRepoFile(t, dir, "notes.txt", "alpha\nbeta\ngamma\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "add notes")
	writeRepoFile(t, dir, "notes.txt", "alpha\nBETA\ngamma\n")
	gitIn(t, dir, "commit", "-q", "-am", "shout beta")
	gitIn(t, dir, "mv", "notes.txt", "renamed.txt")
	gitIn(t, dir, "commit", "-q", "-m", "rename notes")
	writeRepoFile(t, dir, "renamed.txt", "alpha\nBETA\ngamma\ndelta\n")
	gitIn(t, dir, "commit", "-q", "-am", "add delta")
	writeRepoFile(t, dir, "a.txt", "unrelated\n")
	gitIn(t, dir, "commit", "-q", "-am", "unrelated")
	return dir
}

func TestGetFileHistory_FollowsRenames(t *testing.T) {
	dir := historyRepo(t)
	m := openRepo(t, dir)

	entries, err := m.GetFileHistory(dir, "renamed.txt")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Status+" "+e.Subject+" "+e.Path)
	}
	want := []string{
		"M add delta renamed.txt",
		"R rename notes renamed.txt",
		"M shout beta notes.txt",
		"A add notes notes.txt",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("history = %v, want %v", got, want)
	}
	if entries[1].OldPath != "notes.txt" {
		t.Fatalf("the rename should name the old path, got %q", entries[1].OldPath)
	}
	if entries[0].Timestamp == 0 || entries[0].Author != "Test" || len(entries[0].Hash) != 40 {
		t.Fatalf("incomplete entry %+v", entries[0])
	}
}

func TestGetFileDiffAtCommit_OnlyThatFile(t *testing.T) {
	dir := historyRepo(t)
	m := openRepo(t, dir)
	entries, _ := m.GetFileHistory(dir, "renamed.txt")

	shout := entries[2]
	diffs, err := m.GetFileDiffAtCommit(dir, shout.Hash, shout.Path, shout.OldPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].NewPath != "notes.txt" || len(diffs[0].Hunks) != 1 {
		t.Fatalf("want one hunk for notes.txt, got %+v", diffs)
	}
	var changed []string
	for _, l := range diffs[0].Hunks[0].Lines {
		if l.Type != "context" {
			changed = append(changed, l.Type+":"+l.Content)
		}
	}
	if strings.Join(changed, ",") != "del:beta,add:BETA" {
		t.Fatalf("unexpected change %v", changed)
	}

	rename := entries[1]
	diffs, err = m.GetFileDiffAtCommit(dir, rename.Hash, rename.Path, rename.OldPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].OldPath != "notes.txt" || diffs[0].NewPath != "renamed.txt" {
		t.Fatalf("the rename should come back as one file, got %+v", diffs)
	}
}

func TestGetBlame_GroupsLinesByCommit(t *testing.T) {
	dir := historyRepo(t)
	m := openRepo(t, dir)

	chunks, err := m.GetBlame(dir, "renamed.txt", "")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range chunks {
		got = append(got, c.Summary+"@"+strings.Join(c.Lines, "+"))
	}
	want := "add notes@alpha|shout beta@BETA|add notes@gamma|add delta@delta"
	if strings.Join(got, "|") != want {
		t.Fatalf("blame = %v", got)
	}
	if c := chunks[1]; c.StartLine != 2 || c.Author != "Test" || c.Timestamp == 0 || len(c.ShortHash) != 7 || c.Uncommitted {
		t.Fatalf("incomplete chunk %+v", c)
	}
}

func TestGetBlame_UncommittedLinesAndOlderRevisions(t *testing.T) {
	dir := historyRepo(t)
	m := openRepo(t, dir)
	writeRepoFile(t, dir, "renamed.txt", "alpha\nBETA\ngamma\ndelta\nnot committed\n")

	chunks, err := m.GetBlame(dir, "renamed.txt", "")
	if err != nil {
		t.Fatal(err)
	}
	last := chunks[len(chunks)-1]
	if !last.Uncommitted || last.Lines[0] != "not committed" || last.StartLine != 5 {
		t.Fatalf("the new line should be marked uncommitted, got %+v", last)
	}

	entries, _ := m.GetFileHistory(dir, "renamed.txt")
	first := entries[len(entries)-1]
	chunks, err = m.GetBlame(dir, first.Path, first.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 || len(chunks[0].Lines) != 3 || chunks[0].Summary != "add notes" {
		t.Fatalf("at its first commit the file has one author for all lines, got %+v", chunks)
	}
}

func TestGetBlame_RefusesBinaryFilesAndBadRevisions(t *testing.T) {
	dir := newRepo(t)
	writeRepoFile(t, dir, "blob.bin", "a\x00b\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "binary")
	m := openRepo(t, dir)

	if _, err := m.GetBlame(dir, "blob.bin", ""); err == nil {
		t.Fatal("a binary file should be refused")
	}
	if _, err := m.GetBlame(dir, "a.txt", "--output=/tmp/x"); err == nil {
		t.Fatal("a revision that isn't a hash should be refused")
	}
}
