package repo

import (
	"strings"
	"testing"

	"burlwood/internal/gitcli"
)

func cherryPickRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := newRepo(t)
	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	writeRepoFile(t, dir, "b.txt", "feature\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "add b")
	hash := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD"))
	gitIn(t, dir, "checkout", "-q", "main")
	return dir, hash
}

func TestCherryPick_Commits(t *testing.T) {
	dir, hash := cherryPickRepo(t)
	m := openRepo(t, dir)

	res, err := m.CherryPick(dir, hash, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != CherryPickCommitted {
		t.Fatalf("want committed, got %q", res.Outcome)
	}
	if subject := strings.TrimSpace(gitIn(t, dir, "log", "-1", "--format=%s")); subject != "add b" {
		t.Fatalf("want the picked subject on main, got %q", subject)
	}
}

func TestCherryPick_StagesWithoutCommitting(t *testing.T) {
	dir, hash := cherryPickRepo(t)
	m := openRepo(t, dir)
	before := gitIn(t, dir, "rev-parse", "HEAD")

	res, err := m.CherryPick(dir, hash, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != CherryPickStaged {
		t.Fatalf("want staged, got %q", res.Outcome)
	}
	if gitIn(t, dir, "rev-parse", "HEAD") != before {
		t.Fatal("HEAD should not move")
	}
	if staged := strings.TrimSpace(gitIn(t, dir, "diff", "--cached", "--name-only")); staged != "b.txt" {
		t.Fatalf("want b.txt staged, got %q", staged)
	}
}

func TestCherryPick_AlreadyAppliedIsSkipped(t *testing.T) {
	dir, hash := cherryPickRepo(t)
	m := openRepo(t, dir)
	if _, err := m.CherryPick(dir, hash, true); err != nil {
		t.Fatal(err)
	}
	head := gitIn(t, dir, "rev-parse", "HEAD")

	for _, commit := range []bool{true, false} {
		res, err := m.CherryPick(dir, hash, commit)
		if err != nil {
			t.Fatal(err)
		}
		if res.Outcome != CherryPickEmpty {
			t.Fatalf("commit=%v: want empty, got %q", commit, res.Outcome)
		}
		if kind, _ := m.GetMergeKind(dir); kind != MergeKindNone {
			t.Fatalf("commit=%v: an empty pick should not stay in progress, got %q", commit, kind)
		}
		if gitIn(t, dir, "rev-parse", "HEAD") != head {
			t.Fatalf("commit=%v: HEAD should not move", commit)
		}
	}
}

func TestCherryPick_ConflictCanBeAborted(t *testing.T) {
	dir := newRepo(t)
	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	writeRepoFile(t, dir, "a.txt", "theirs\n")
	gitIn(t, dir, "commit", "-q", "-am", "feature edit")
	hash := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD"))
	gitIn(t, dir, "checkout", "-q", "main")
	writeRepoFile(t, dir, "a.txt", "ours\n")
	gitIn(t, dir, "commit", "-q", "-am", "main edit")
	head := gitIn(t, dir, "rev-parse", "HEAD")
	m := openRepo(t, dir)

	res, err := m.CherryPick(dir, hash, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != CherryPickConflict {
		t.Fatalf("want conflict, got %q", res.Outcome)
	}
	if kind, _ := m.GetMergeKind(dir); kind != MergeKindCherryPick {
		t.Fatalf("want cherry-pick in progress, got %q", kind)
	}
	if msg, _ := m.GetMergeMessage(dir); !strings.HasPrefix(msg, "feature edit") {
		t.Fatalf("MERGE_MSG should carry the picked message, got %q", msg)
	}

	if _, err := m.CherryPick(dir, hash, true); err == nil {
		t.Fatal("a second pick should be refused while files are conflicted")
	}

	if err := m.AbortMerge(dir); err != nil {
		t.Fatal(err)
	}
	if kind, _ := m.GetMergeKind(dir); kind != MergeKindNone {
		t.Fatalf("abort should end the cherry-pick, got %q", kind)
	}
	if gitIn(t, dir, "rev-parse", "HEAD") != head {
		t.Fatal("abort should leave HEAD where it was")
	}
}

func TestAbortMerge_EndsConflictedRevert(t *testing.T) {
	dir := newRepo(t)
	writeRepoFile(t, dir, "a.txt", "two\n")
	gitIn(t, dir, "commit", "-q", "-am", "two")
	target := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD"))
	writeRepoFile(t, dir, "a.txt", "three\n")
	gitIn(t, dir, "commit", "-q", "-am", "three")
	m := openRepo(t, dir)

	if _, err := gitcli.Run(dir, "revert", "--no-edit", target); err == nil {
		t.Fatal("reverting an overwritten change should conflict")
	}
	if kind, _ := m.GetMergeKind(dir); kind != MergeKindRevert {
		t.Fatalf("want revert in progress, got %q", kind)
	}
	if err := m.AbortMerge(dir); err != nil {
		t.Fatal(err)
	}
	if kind, _ := m.GetMergeKind(dir); kind != MergeKindNone {
		t.Fatalf("abort should end the revert, got %q", kind)
	}
}
