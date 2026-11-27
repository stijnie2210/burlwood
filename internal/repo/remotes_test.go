package repo

import (
	"reflect"
	"testing"
)

func repoWithRemoteBranches(t *testing.T, names ...string) (string, string) {
	t.Helper()
	dir, remote := tagRepoWithRemote(t)
	gitIn(t, dir, "push", "-q", "-u", "origin", "main")
	for _, name := range names {
		gitIn(t, dir, "push", "-q", "origin", "main:refs/heads/"+name)
	}
	gitIn(t, dir, "fetch", "-q", "origin")
	return dir, remote
}

func TestPruneRemote_ReportsTheBranchesItRemoved(t *testing.T) {
	dir, remote := repoWithRemoteBranches(t, "old-a", "old-b", "keep")
	gitIn(t, remote, "branch", "-D", "old-b", "old-a")
	m := openRepo(t, dir)

	pruned, err := m.PruneRemote(dir, "origin")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"old-a", "old-b"}; !reflect.DeepEqual(pruned, want) {
		t.Fatalf("pruned %v, want %v", pruned, want)
	}
	if revParse(dir, "refs/remotes/origin/keep") == "" || revParse(dir, "refs/remotes/origin/main") == "" {
		t.Fatal("branches still on origin should stay")
	}

	pruned, err = m.PruneRemote(dir, "origin")
	if err != nil {
		t.Fatal(err)
	}
	if len(pruned) != 0 {
		t.Fatalf("nothing left to prune, got %v", pruned)
	}
}

func TestFetchAll_PrunesWhenTheSettingIsOn(t *testing.T) {
	for _, prune := range []bool{true, false} {
		dir, remote := repoWithRemoteBranches(t, "gone")
		gitIn(t, remote, "branch", "-D", "gone")
		m := openRepo(t, dir)
		withPrefs(t, func(p *AppPrefs) { p.PruneOnFetch = prune })

		if err := m.FetchAll(dir); err != nil {
			t.Fatal(err)
		}
		if stillThere := revParse(dir, "refs/remotes/origin/gone") != ""; stillThere == prune {
			t.Fatalf("prune=%v: origin/gone still there = %v", prune, stillThere)
		}
	}
}

// Sets the cached prefs only, so tests never read or write the real prefs.json.
func withPrefs(t *testing.T, edit func(*AppPrefs)) {
	t.Helper()
	prefsCacheMu.Lock()
	old := prefsCache
	p := DefaultPrefs()
	edit(&p)
	prefsCache = &p
	prefsCacheMu.Unlock()
	t.Cleanup(func() {
		prefsCacheMu.Lock()
		prefsCache = old
		prefsCacheMu.Unlock()
	})
}
