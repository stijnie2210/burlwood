package gitcli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoRoot(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := RepoRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(root)
	if gotReal, _ := filepath.EvalSymlinks(got); gotReal != want {
		t.Errorf("RepoRoot(sub) = %s, want %s", got, root)
	}

	_, err = RepoRoot(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "isn't inside a git repository") {
		t.Errorf("plain folder: got %v", err)
	}

	_, err = RepoRoot(filepath.Join(root, "missing"))
	if err == nil || strings.Contains(err.Error(), "isn't inside a git repository") {
		t.Errorf("missing folder should report the real error, got %v", err)
	}
}
