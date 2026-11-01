package repo

import "testing"

func openRepo(t *testing.T, dir string) *Manager {
	t.Helper()
	m := NewManager()
	if err := m.Open(dir); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMergeBranch_Outcomes(t *testing.T) {
	dir := newRepo(t)
	m := openRepo(t, dir)

	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	writeRepoFile(t, dir, "b.txt", "feature\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "feature work")
	gitIn(t, dir, "checkout", "-q", "main")

	res, err := m.MergeBranch(dir, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != MergeFastForward {
		t.Fatalf("want fast-forward, got %q", res.Outcome)
	}

	res, err = m.MergeBranch(dir, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != MergeUpToDate {
		t.Fatalf("want up-to-date, got %q", res.Outcome)
	}

	gitIn(t, dir, "checkout", "-q", "feature")
	writeRepoFile(t, dir, "c.txt", "more\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "more feature work")
	gitIn(t, dir, "checkout", "-q", "main")
	writeRepoFile(t, dir, "d.txt", "main\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "main work")

	res, err = m.MergeBranch(dir, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != MergeCommit {
		t.Fatalf("want merge commit, got %q", res.Outcome)
	}
}

func TestMergeBranch_ConflictLeavesMergeInProgress(t *testing.T) {
	dir := newRepo(t)
	m := openRepo(t, dir)

	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	writeRepoFile(t, dir, "a.txt", "theirs\n")
	gitIn(t, dir, "commit", "-q", "-am", "feature edit")
	gitIn(t, dir, "checkout", "-q", "main")
	writeRepoFile(t, dir, "a.txt", "ours\n")
	gitIn(t, dir, "commit", "-q", "-am", "main edit")

	if _, err := m.MergeBranch(dir, "feature"); err == nil {
		t.Fatal("a conflicting merge should return an error")
	}
	inMerge, err := m.IsInMerge(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !inMerge {
		t.Fatal("the conflict should leave the merge in progress")
	}
}

func TestListBranches_ReportsUpstream(t *testing.T) {
	dir := newRepo(t)
	m := openRepo(t, dir)
	gitIn(t, dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	gitIn(t, dir, "config", "remote.origin.url", "https://example.com/repo.git")
	gitIn(t, dir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	gitIn(t, dir, "branch", "--set-upstream-to=origin/main", "main")
	gitIn(t, dir, "branch", "feature")

	branches, err := m.ListBranches(dir)
	if err != nil {
		t.Fatal(err)
	}
	upstreams := map[string]string{}
	for _, b := range branches {
		if !b.IsRemote {
			upstreams[b.Name] = b.Upstream
		}
	}
	if upstreams["main"] != "origin/main" || upstreams["feature"] != "" {
		t.Fatalf("unexpected upstreams %v", upstreams)
	}
}
