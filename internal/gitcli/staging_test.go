package gitcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStageFiles_StagesAndUnstagesInOneCall(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "a.txt", "a\n", "init")
	writeFile(t, dir, "a.txt", "changed\n")
	writeFile(t, dir, "new file.txt", "new\n")
	if err := os.Remove(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "b.txt", "b\n")

	paths := []string{"a.txt", "new file.txt", "b.txt"}
	if err := StageFiles(dir, paths); err != nil {
		t.Fatal(err)
	}
	staged, _ := Run(dir, "diff", "--cached", "--name-only")
	for _, p := range paths {
		if !strings.Contains(staged.Stdout, p) {
			t.Fatalf("%q not staged: %q", p, staged.Stdout)
		}
	}
	if err := UnstageFiles(dir, paths); err != nil {
		t.Fatal(err)
	}
	staged, _ = Run(dir, "diff", "--cached", "--name-only")
	if strings.TrimSpace(staged.Stdout) != "" {
		t.Fatalf("still staged: %q", staged.Stdout)
	}
}

func TestRepoRoot_ResolvesSubfolderAndRejectsNonRepo(t *testing.T) {
	dir := initRepo(t)
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := RepoRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(dir)
	got, _ := filepath.EvalSymlinks(root)
	if got != want {
		t.Fatalf("root = %s, want %s", got, want)
	}
	if _, err := RepoRoot(t.TempDir()); err == nil || !strings.Contains(err.Error(), "isn't inside a git repository") {
		t.Fatalf("expected a not-a-repo error, got %v", err)
	}
}
