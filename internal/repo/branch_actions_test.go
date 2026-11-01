package repo

import (
	"path/filepath"
	"testing"
)

func repoWithOrigin(t *testing.T) (string, *Manager) {
	t.Helper()
	dir := newRepo(t)
	remote := filepath.Join(t.TempDir(), "origin.git")
	gitIn(t, dir, "init", "-q", "--bare", remote)
	gitIn(t, dir, "remote", "add", "origin", remote)
	gitIn(t, dir, "push", "-q", "-u", "origin", "main")
	return dir, openRepo(t, dir)
}

func commitIn(t *testing.T, dir, file, msg string) {
	t.Helper()
	writeRepoFile(t, dir, file, msg+"\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", msg)
}

func TestDeleteBranch_RestoreBringsBackHashAndUpstream(t *testing.T) {
	dir, m := repoWithOrigin(t)
	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	commitIn(t, dir, "f.txt", "feature work")
	gitIn(t, dir, "push", "-q", "-u", "origin", "feature")
	hash := revParse(dir, "feature")
	gitIn(t, dir, "checkout", "-q", "main")

	deleted, err := m.DeleteBranch(dir, "feature", true)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Hash != hash || deleted.Upstream != "origin/feature" {
		t.Fatalf("unexpected %+v", deleted)
	}
	if revParse(dir, "refs/heads/feature") != "" {
		t.Fatal("branch still exists")
	}

	if err := m.RestoreBranch(dir, deleted); err != nil {
		t.Fatal(err)
	}
	if revParse(dir, "refs/heads/feature") != hash || upstreamOf(dir, "feature") != "origin/feature" {
		t.Fatal("restore lost the hash or the upstream")
	}
}

func TestDeleteBranch_SafeDeleteRefusesUnmergedWork(t *testing.T) {
	dir := newRepo(t)
	m := openRepo(t, dir)
	gitIn(t, dir, "checkout", "-q", "-b", "wip")
	commitIn(t, dir, "w.txt", "wip")
	gitIn(t, dir, "checkout", "-q", "main")

	if _, err := m.DeleteBranch(dir, "wip", false); err == nil {
		t.Fatal("deleting unmerged work without force should fail")
	}
	if revParse(dir, "refs/heads/wip") == "" {
		t.Fatal("the branch must survive a refused delete")
	}
}

func TestPreviewRebase_CountsPushedCommitsItWouldRewrite(t *testing.T) {
	dir, m := repoWithOrigin(t)
	gitIn(t, dir, "branch", "base")
	commitIn(t, dir, "a.txt", "pushed one")
	commitIn(t, dir, "b.txt", "pushed two")
	gitIn(t, dir, "push", "-q", "origin", "main")
	commitIn(t, dir, "c.txt", "local only")
	gitIn(t, dir, "checkout", "-q", "base")
	commitIn(t, dir, "d.txt", "base moved on")
	gitIn(t, dir, "checkout", "-q", "main")

	p, err := m.PreviewRebase(dir, "base")
	if err != nil {
		t.Fatal(err)
	}
	if p.Upstream != "origin/main" || p.PushedCommits != 2 {
		t.Fatalf("want 2 pushed commits on origin/main, got %+v", p)
	}
}

func TestPreviewRebase_NothingPushedOrNoUpstream(t *testing.T) {
	dir, m := repoWithOrigin(t)
	gitIn(t, dir, "branch", "base")
	commitIn(t, dir, "a.txt", "local only")
	if p, _ := m.PreviewRebase(dir, "base"); p.PushedCommits != 0 {
		t.Fatalf("local-only commits aren't pushed: %+v", p)
	}

	gitIn(t, dir, "checkout", "-q", "-b", "untracked-branch")
	if p, _ := m.PreviewRebase(dir, "main"); p.Upstream != "" || p.PushedCommits != 0 {
		t.Fatalf("no upstream means nothing pushed: %+v", p)
	}
}

func TestStartRebase_ReportsUpToDate(t *testing.T) {
	dir := newRepo(t)
	m := openRepo(t, dir)
	gitIn(t, dir, "branch", "old")

	res, err := m.StartRebase(dir, "old")
	if err != nil {
		t.Fatal(err)
	}
	if !res.UpToDate {
		t.Fatal("rebasing onto an ancestor changes nothing")
	}

	gitIn(t, dir, "checkout", "-q", "-b", "ahead")
	commitIn(t, dir, "x.txt", "ahead")
	gitIn(t, dir, "checkout", "-q", "main")
	commitIn(t, dir, "y.txt", "main moved")
	gitIn(t, dir, "checkout", "-q", "ahead")
	if res, err := m.StartRebase(dir, "main"); err != nil || res.UpToDate {
		t.Fatalf("a real rebase moves HEAD: %+v %v", res, err)
	}
}

func TestCheckoutRemoteBranch(t *testing.T) {
	dir, m := repoWithOrigin(t)
	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	commitIn(t, dir, "f.txt", "one")
	gitIn(t, dir, "push", "-q", "-u", "origin", "feature")
	commitIn(t, dir, "g.txt", "two")
	gitIn(t, dir, "push", "-q", "origin", "feature")
	gitIn(t, dir, "reset", "-q", "--hard", "HEAD~1")
	gitIn(t, dir, "checkout", "-q", "main")

	res, err := m.CheckoutRemoteBranch(dir, "origin", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if revParse(dir, "HEAD") != revParse(dir, "origin/feature") {
		t.Fatal("a branch behind the remote it tracks should be fast-forwarded")
	}
	if res.Message != "Switched to feature and fast-forwarded it to origin/feature" {
		t.Fatalf("unexpected message %q", res.Message)
	}

	gitIn(t, dir, "checkout", "-q", "main")
	gitIn(t, dir, "branch", "-q", "-D", "feature")
	if _, err := m.CheckoutRemoteBranch(dir, "origin", "feature"); err != nil {
		t.Fatal(err)
	}
	if upstreamOf(dir, "feature") != "origin/feature" {
		t.Fatal("without a local branch it creates one that tracks the remote")
	}
}

func TestCheckoutRemoteBranch_SaysHowALocalBranchItCantMoveDiffers(t *testing.T) {
	dir, m := repoWithOrigin(t)
	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	gitIn(t, dir, "push", "-q", "-u", "origin", "feature")
	commitIn(t, dir, "local.txt", "local work")
	local := revParse(dir, "HEAD")
	gitIn(t, dir, "checkout", "-q", "main")

	res, err := m.CheckoutRemoteBranch(dir, "origin", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if revParse(dir, "HEAD") != local {
		t.Fatal("a local branch ahead of its remote must not move")
	}
	if want := "Switched to feature. It has 1 commit that origin/feature doesn't."; res.Message != want {
		t.Fatalf("ahead: got %q, want %q", res.Message, want)
	}

	gitIn(t, dir, "checkout", "-q", "-b", "elsewhere", "origin/feature")
	commitIn(t, dir, "r1.txt", "remote one")
	commitIn(t, dir, "r2.txt", "remote two")
	gitIn(t, dir, "push", "-q", "origin", "HEAD:feature")
	gitIn(t, dir, "checkout", "-q", "main")

	res, err = m.CheckoutRemoteBranch(dir, "origin", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if revParse(dir, "HEAD") != local {
		t.Fatal("a diverged local branch must not move")
	}
	want := "Switched to feature. It has 1 commit that origin/feature doesn't, and is 2 behind it. Pull to bring those in."
	if res.Message != want {
		t.Fatalf("diverged: got %q, want %q", res.Message, want)
	}
}

func TestCheckoutRemoteBranch_NamesWhatTheLocalBranchTracksInstead(t *testing.T) {
	dir, m := repoWithOrigin(t)
	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	gitIn(t, dir, "push", "-q", "origin", "feature")
	gitIn(t, dir, "checkout", "-q", "main")

	res, err := m.CheckoutRemoteBranch(dir, "origin", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if want := "Switched to feature. It doesn't track origin/feature."; res.Message != want {
		t.Fatalf("no upstream: got %q, want %q", res.Message, want)
	}

	gitIn(t, dir, "branch", "-q", "--set-upstream-to=origin/main", "feature")
	gitIn(t, dir, "checkout", "-q", "main")
	res, err = m.CheckoutRemoteBranch(dir, "origin", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if want := "Switched to feature. It tracks origin/main, not origin/feature."; res.Message != want {
		t.Fatalf("other upstream: got %q, want %q", res.Message, want)
	}
}
