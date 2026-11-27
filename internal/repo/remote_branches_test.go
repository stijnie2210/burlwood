package repo

import "testing"

func repoWithPushedFeature(t *testing.T) (string, string) {
	t.Helper()
	dir, remote := tagRepoWithRemote(t)
	gitIn(t, dir, "push", "-q", "-u", "origin", "main")
	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	writeRepoFile(t, dir, "f.txt", "feature\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "feature")
	gitIn(t, dir, "push", "-q", "-u", "origin", "feature")
	gitIn(t, dir, "checkout", "-q", "main")
	return dir, remote
}

func TestDeleteRemoteBranch_AndRestore(t *testing.T) {
	dir, remote := repoWithPushedFeature(t)
	m := openRepo(t, dir)
	hash := revParse(dir, "refs/heads/feature")

	deleted, err := m.DeleteRemoteBranch(dir, "origin", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.AlreadyGone || deleted.Hash != hash {
		t.Fatalf("unexpected result %+v", deleted)
	}
	if revParse(remote, "refs/heads/feature") != "" {
		t.Fatal("feature should be gone from origin")
	}
	if revParse(dir, "refs/remotes/origin/feature") != "" {
		t.Fatal("the remote-tracking ref should be gone too")
	}
	if revParse(dir, "refs/heads/feature") != hash {
		t.Fatal("the local branch should stay")
	}

	if err := m.RestoreRemoteBranch(dir, deleted); err != nil {
		t.Fatal(err)
	}
	if revParse(remote, "refs/heads/feature") != hash {
		t.Fatal("undo should push feature back to origin")
	}
	if revParse(dir, "refs/remotes/origin/feature") != hash {
		t.Fatal("undo should bring the remote-tracking ref back")
	}
}

func TestDeleteRemoteBranch_AlreadyDeletedRemotely(t *testing.T) {
	dir, remote := repoWithPushedFeature(t)
	gitIn(t, remote, "branch", "-D", "feature")
	m := openRepo(t, dir)

	deleted, err := m.DeleteRemoteBranch(dir, "origin", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if !deleted.AlreadyGone {
		t.Fatal("want AlreadyGone for a branch origin no longer has")
	}
	if revParse(dir, "refs/remotes/origin/feature") != "" {
		t.Fatal("the stale remote-tracking ref should be removed")
	}
}
