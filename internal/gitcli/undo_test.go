package gitcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
	} {
		if _, err := Run(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func commitFile(t *testing.T, dir, name, content, msg string) {
	t.Helper()
	writeFile(t, dir, name, content)
	if _, err := Run(dir, "add", "--", name); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(dir, "commit", "-q", "-m", msg); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDiscardUndo_ModifiedFile(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "a.txt", "original\n", "init")
	writeFile(t, dir, "a.txt", "edited\n")

	snap, err := SnapshotWorkingFile(dir, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := DiscardFile(dir, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dir, "a.txt"); got != "original\n" {
		t.Fatalf("discard left %q", got)
	}
	if err := RestoreWorkingFile(dir, snap); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dir, "a.txt"); got != "edited\n" {
		t.Fatalf("undo restored %q, want edited", got)
	}
}

func TestDiscardUndo_UntrackedFile(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "a.txt", "x\n", "init")
	writeFile(t, dir, "new.txt", "brand new\n")

	snap, err := SnapshotWorkingFile(dir, "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := DiscardFile(dir, "new.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.txt")); !os.IsNotExist(err) {
		t.Fatal("untracked file should be gone after discard")
	}
	if err := RestoreWorkingFile(dir, snap); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dir, "new.txt"); got != "brand new\n" {
		t.Fatalf("undo restored %q", got)
	}
}

func TestDiscardUndo_DeletedFile(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "a.txt", "keep\n", "init")
	if err := os.Remove(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatal(err)
	}

	snap, err := SnapshotWorkingFile(dir, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := DiscardFile(dir, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := RestoreWorkingFile(dir, snap); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
		t.Fatal("undo should delete the file again")
	}
}

func TestHardResetUndo_RestoresHeadAndLocalChanges(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "a.txt", "one\n", "first")
	first, _ := revParse(dir, "HEAD")
	commitFile(t, dir, "a.txt", "two\n", "second")
	second, _ := revParse(dir, "HEAD")
	writeFile(t, dir, "a.txt", "staged\n")
	if _, err := Run(dir, "add", "a.txt"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "a.txt", "unstaged on top\n")

	undo, err := HardResetWithUndo(dir, first)
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dir, "a.txt"); got != "one\n" {
		t.Fatalf("hard reset left %q", got)
	}
	if err := UndoHardReset(dir, undo); err != nil {
		t.Fatal(err)
	}
	if head, _ := revParse(dir, "HEAD"); head != second {
		t.Fatalf("HEAD = %s, want %s", head, second)
	}
	if got := readFile(t, dir, "a.txt"); got != "unstaged on top\n" {
		t.Fatalf("working tree = %q", got)
	}
	staged, _ := Run(dir, "show", ":a.txt")
	if staged.Stdout != "staged\n" {
		t.Fatalf("index = %q, want staged", staged.Stdout)
	}
}

func TestHardResetUndo_RefusesAfterNewWork(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "a.txt", "one\n", "first")
	first, _ := revParse(dir, "HEAD")
	commitFile(t, dir, "a.txt", "two\n", "second")

	undo, err := HardResetWithUndo(dir, first)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "a.txt", "new work\n")

	err = UndoHardReset(dir, undo)
	if err == nil || !strings.Contains(err.Error(), "changed after the reset") {
		t.Fatalf("expected refusal, got %v", err)
	}
	if got := readFile(t, dir, "a.txt"); got != "new work\n" {
		t.Fatalf("refused undo must not touch files, got %q", got)
	}
}
