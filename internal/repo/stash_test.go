package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStash_IncludesUntrackedFilesWhenTheSettingIsOn(t *testing.T) {
	for _, include := range []bool{true, false} {
		dir := newRepo(t)
		writeRepoFile(t, dir, "a.txt", "changed\n")
		writeRepoFile(t, dir, "new.txt", "untracked\n")
		m := openRepo(t, dir)
		withPrefs(t, func(p *AppPrefs) { p.StashIncludeUntracked = include })

		if err := m.Stash(dir); err != nil {
			t.Fatal(err)
		}
		if status := strings.TrimSpace(gitIn(t, dir, "status", "--porcelain")); include && status != "" {
			t.Fatalf("include=true: the working tree should be clean, got %q", status)
		} else if !include && status != "?? new.txt" {
			t.Fatalf("include=false: only new.txt should be left, got %q", status)
		}

		gitIn(t, dir, "stash", "pop", "-q")
		if _, err := os.Stat(filepath.Join(dir, "new.txt")); err != nil {
			t.Fatalf("include=%v: new.txt should be back after pop: %v", include, err)
		}
	}
}
