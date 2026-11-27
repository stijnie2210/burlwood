package repo

import (
	"strings"
	"testing"
)

func rebaseRepo(t *testing.T, n int) (string, []string) {
	t.Helper()
	dir := newRepo(t)
	hashes := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		name := "c" + string(rune('0'+i))
		writeRepoFile(t, dir, name+".txt", name+"\n")
		gitIn(t, dir, "add", ".")
		gitIn(t, dir, "commit", "-q", "-m", name, "-m", "body of "+name)
		hashes = append(hashes, strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD")))
	}
	return dir, hashes
}

func subjects(t *testing.T, dir string) string {
	t.Helper()
	return strings.Join(strings.Fields(gitIn(t, dir, "log", "--reverse", "--format=%s")), " ")
}

func TestGetRebaseTodo_OldestFirstWithMessages(t *testing.T) {
	dir, h := rebaseRepo(t, 3)
	m := openRepo(t, dir)

	todo, err := m.GetRebaseTodo(dir, h[1])
	if err != nil {
		t.Fatal(err)
	}
	if len(todo.Commits) != 2 || todo.Commits[0].Hash != h[1] || todo.Commits[1].Hash != h[2] {
		t.Fatalf("want c2 then c3, got %+v", todo.Commits)
	}
	if c := todo.Commits[0]; c.Subject != "c2" || c.Message != "c2\n\nbody of c2" {
		t.Fatalf("unexpected message %q / %q", c.Subject, c.Message)
	}
}

func TestInteractiveRebase_RewordSquashFixupDropReorder(t *testing.T) {
	dir, h := rebaseRepo(t, 5)
	writeRepoFile(t, dir, "a.txt", "uncommitted\n")
	m := openRepo(t, dir)

	res, err := m.StartInteractiveRebase(dir, h[0], []RebaseStep{
		{Hash: h[0], Action: "reword", Message: "first, reworded\n\nnew body"},
		{Hash: h[1], Action: "squash"},
		{Hash: h[4], Action: "pick"},
		{Hash: h[3], Action: "fixup"},
		{Hash: h[2], Action: "drop"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != InteractiveRebaseDone {
		t.Fatalf("want done, got %q", res.Outcome)
	}
	if got := subjects(t, dir); got != "init first, reworded c5" {
		t.Fatalf("unexpected history %q", got)
	}
	body := gitIn(t, dir, "log", "-1", "--format=%B", "HEAD^")
	if !strings.Contains(body, "new body") || !strings.Contains(body, "body of c2") {
		t.Fatalf("the squash should keep both messages, got %q", body)
	}
	files := gitIn(t, dir, "ls-tree", "--name-only", "HEAD")
	for name, want := range map[string]bool{"c1.txt": true, "c2.txt": true, "c3.txt": false, "c4.txt": true, "c5.txt": true} {
		if strings.Contains(files, name) != want {
			t.Fatalf("%s present = %v, want %v", name, !want, want)
		}
	}
	if got := readRepoFile(t, dir, "a.txt"); got != "uncommitted\n" {
		t.Fatalf("uncommitted changes should survive the rebase, got %q", got)
	}

	if err := m.UndoInteractiveRebase(dir, res); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD")); got != h[4] {
		t.Fatalf("undo should return to the old HEAD, got %s", got)
	}
	if got := readRepoFile(t, dir, "a.txt"); got != "uncommitted\n" {
		t.Fatalf("undo should keep uncommitted changes, got %q", got)
	}
}

func TestInteractiveRebase_FromTheRootCommit(t *testing.T) {
	dir, h := rebaseRepo(t, 1)
	root := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD^"))
	m := openRepo(t, dir)

	res, err := m.StartInteractiveRebase(dir, root, []RebaseStep{
		{Hash: root, Action: "pick"},
		{Hash: h[0], Action: "fixup"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != InteractiveRebaseDone || subjects(t, dir) != "init" {
		t.Fatalf("want one commit 'init', got %q (%s)", subjects(t, dir), res.Outcome)
	}
}

func TestInteractiveRebase_ConflictStopsAndCanBeAborted(t *testing.T) {
	dir := newRepo(t)
	var h []string
	for _, content := range []string{"two\n", "three\n"} {
		writeRepoFile(t, dir, "a.txt", content)
		gitIn(t, dir, "commit", "-q", "-am", strings.TrimSpace(content))
		h = append(h, strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD")))
	}
	m := openRepo(t, dir)

	res, err := m.StartInteractiveRebase(dir, h[0], []RebaseStep{
		{Hash: h[1], Action: "reword", Message: "three, first"},
		{Hash: h[0], Action: "pick"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != InteractiveRebaseStopped {
		t.Fatalf("reordering edits of one line should stop on a conflict, got %q", res.Outcome)
	}
	if inRebase, _ := m.IsInRebase(dir); !inRebase {
		t.Fatal("the rebase should be in progress")
	}
	if err := m.AbortRebase(dir); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD")); got != h[1] {
		t.Fatalf("abort should restore HEAD, got %s", got)
	}
}

func TestInteractiveRebase_RejectsBadPlans(t *testing.T) {
	dir, h := rebaseRepo(t, 2)
	m := openRepo(t, dir)
	head := gitIn(t, dir, "rev-parse", "HEAD")

	plans := map[string][]RebaseStep{
		"squash first":   {{Hash: h[0], Action: "squash"}, {Hash: h[1], Action: "pick"}},
		"all dropped":    {{Hash: h[0], Action: "drop"}, {Hash: h[1], Action: "drop"}},
		"missing commit": {{Hash: h[0], Action: "pick"}},
		"same twice":     {{Hash: h[0], Action: "pick"}, {Hash: h[0], Action: "pick"}},
		"empty reword":   {{Hash: h[0], Action: "reword", Message: "  "}, {Hash: h[1], Action: "pick"}},
		"unknown action": {{Hash: h[0], Action: "exec rm -rf x"}, {Hash: h[1], Action: "pick"}},
	}
	for name, steps := range plans {
		if _, err := m.StartInteractiveRebase(dir, h[0], steps); err == nil {
			t.Fatalf("%s: want an error", name)
		}
		if gitIn(t, dir, "rev-parse", "HEAD") != head {
			t.Fatalf("%s: HEAD must not move", name)
		}
	}
}

func TestGetRebaseTodo_RefusesMergesAndOtherBranches(t *testing.T) {
	dir, h := rebaseRepo(t, 1)
	gitIn(t, dir, "checkout", "-q", "-b", "side", h[0]+"^")
	writeRepoFile(t, dir, "side.txt", "side\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "side")
	side := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD"))
	gitIn(t, dir, "checkout", "-q", "main")
	m := openRepo(t, dir)

	if _, err := m.GetRebaseTodo(dir, side); err == nil {
		t.Fatal("a commit on another branch should be refused")
	}
	gitIn(t, dir, "merge", "-q", "--no-ff", "--no-edit", "side")
	if _, err := m.GetRebaseTodo(dir, h[0]); err == nil {
		t.Fatal("a range with a merge commit should be refused")
	}
}

func TestGetRebaseTodo_MarksPushedCommits(t *testing.T) {
	dir, h := rebaseRepo(t, 2)
	gitIn(t, dir, "update-ref", "refs/remotes/origin/main", h[0])
	gitIn(t, dir, "config", "remote.origin.url", "https://example.com/repo.git")
	gitIn(t, dir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	gitIn(t, dir, "branch", "--set-upstream-to=origin/main", "main")
	m := openRepo(t, dir)

	todo, err := m.GetRebaseTodo(dir, h[0])
	if err != nil {
		t.Fatal(err)
	}
	if !todo.Commits[0].Pushed || todo.Commits[1].Pushed || todo.Upstream != "origin/main" {
		t.Fatalf("want only c1 pushed to origin/main, got %+v", todo)
	}
}
