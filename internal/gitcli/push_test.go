package gitcli

import (
	"path/filepath"
	"strings"
	"testing"
)

func initRepoWithRemote(t *testing.T) (string, string) {
	t.Helper()
	dir := initRepo(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	if _, err := Run(dir, "init", "-q", "--bare", "-b", "main", remote); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(dir, "remote", "add", "origin", remote); err != nil {
		t.Fatal(err)
	}
	commitFile(t, dir, "a.txt", "one\n", "first")
	return dir, remote
}

func TestPushBranch_SetsUpstreamOnFirstPush(t *testing.T) {
	dir, _ := initRepoWithRemote(t)

	res, err := PushBranch(dir, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.SetUpstream || res.Upstream != "origin/main" || res.Branch != "main" {
		t.Fatalf("unexpected result %+v", res)
	}
	if got := configValue(dir, "branch.main.merge"); got != "refs/heads/main" {
		t.Fatalf("upstream not recorded: %q", got)
	}

	commitFile(t, dir, "a.txt", "two\n", "second")
	res, err = PushBranch(dir, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if res.SetUpstream || res.Upstream != "origin/main" {
		t.Fatalf("second push should reuse the upstream: %+v", res)
	}
}

func TestPushBranch_RejectsRewrittenHistoryThenForcesWithLease(t *testing.T) {
	dir, remote := initRepoWithRemote(t)
	commitFile(t, dir, "a.txt", "two\n", "second")
	if _, err := PushBranch(dir, "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(dir, "commit", "-q", "--amend", "-m", "second, amended"); err != nil {
		t.Fatal(err)
	}

	res, err := PushBranch(dir, "", false)
	if err != nil {
		t.Fatalf("rejection should be a result, not an error: %v", err)
	}
	if !res.Rejected {
		t.Fatalf("expected a rejection, got %+v", res)
	}

	if _, err := PushBranch(dir, "", true); err != nil {
		t.Fatal(err)
	}
	local, _ := revParse(dir, "HEAD")
	remoteHead, _ := revParse(remote, "refs/heads/main")
	if local != remoteHead {
		t.Fatalf("remote not updated: %s vs %s", remoteHead, local)
	}
}

func TestPushBranch_ForceRefusesUnfetchedRemoteWork(t *testing.T) {
	dir, remote := initRepoWithRemote(t)
	if _, err := PushBranch(dir, "", false); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "other")
	if _, err := Run(dir, "clone", "-q", remote, other); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"config", "user.email", "o@example.com"}, {"config", "user.name", "O"}} {
		if _, err := Run(other, args...); err != nil {
			t.Fatal(err)
		}
	}
	commitFile(t, other, "b.txt", "theirs\n", "someone else's work")
	if _, err := Run(other, "push", "-q", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(dir, "commit", "-q", "--amend", "-m", "rewritten"); err != nil {
		t.Fatal(err)
	}

	_, err := PushBranch(dir, "", true)
	if err == nil || !strings.Contains(err.Error(), "aren't in your branch's history") {
		t.Fatalf("expected the lease to refuse, got %v", err)
	}
}

// Auto-fetch updates origin/main in the background; a plain --force-with-lease would then
// accept the teammate's commit as "known" and overwrite it.
func TestPushBranch_ForceRefusesWorkOnlySeenThroughBackgroundFetch(t *testing.T) {
	dir, remote := initRepoWithRemote(t)
	if _, err := PushBranch(dir, "", false); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "other")
	if _, err := Run(dir, "clone", "-q", remote, other); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"config", "user.email", "o@example.com"}, {"config", "user.name", "O"}} {
		if _, err := Run(other, args...); err != nil {
			t.Fatal(err)
		}
	}
	commitFile(t, other, "b.txt", "theirs\n", "teammate's work")
	if _, err := Run(other, "push", "-q", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(dir, "commit", "-q", "--amend", "-m", "rewritten"); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(dir, "fetch", "-q", "--all"); err != nil {
		t.Fatal(err)
	}

	if _, err := PushBranch(dir, "", true); err == nil {
		t.Fatal("force push overwrote a commit that was only fetched, never integrated")
	}
	theirs, _ := revParse(other, "HEAD")
	remoteHead, _ := revParse(remote, "refs/heads/main")
	if remoteHead != theirs {
		t.Fatalf("remote lost the teammate's commit: %s, want %s", remoteHead, theirs)
	}
}

func TestPushBranch_NamedBranchAndNoRemote(t *testing.T) {
	dir, _ := initRepoWithRemote(t)
	if _, err := Run(dir, "branch", "feature"); err != nil {
		t.Fatal(err)
	}
	res, err := PushBranch(dir, "feature", false)
	if err != nil || res.Upstream != "origin/feature" || !res.SetUpstream {
		t.Fatalf("named push: %+v, %v", res, err)
	}

	bare := initRepo(t)
	commitFile(t, bare, "a.txt", "x\n", "init")
	if _, err := PushBranch(bare, "", false); err == nil || !strings.Contains(err.Error(), "no remote") {
		t.Fatalf("expected a no-remote error, got %v", err)
	}
}
