package repo

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseTags(t *testing.T) {
	out := "v0.2.0\taaaa\tcccc\nv0.1.0\tbbbb\t\n"
	want := []TagInfo{
		{Name: "v0.2.0", Hash: "cccc"},
		{Name: "v0.1.0", Hash: "bbbb"},
	}
	if got := parseTags(out); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseTags = %#v, want %#v", got, want)
	}
}

func TestParseTagsEmpty(t *testing.T) {
	if got := parseTags(""); len(got) != 0 {
		t.Fatalf("parseTags(\"\") = %#v, want empty", got)
	}
}

func tagRepoWithRemote(t *testing.T) (string, string) {
	t.Helper()
	dir := newRepo(t)
	remote := t.TempDir()
	gitIn(t, remote, "init", "-q", "--bare")
	gitIn(t, dir, "remote", "add", "origin", remote)
	return dir, remote
}

func TestDeleteTag_LocalOnlyAndRestore(t *testing.T) {
	dir := newRepo(t)
	gitIn(t, dir, "tag", "-a", "v1", "-m", "release notes")
	gitIn(t, dir, "tag", "light")
	m := openRepo(t, dir)

	for _, name := range []string{"v1", "light"} {
		before := gitIn(t, dir, "rev-parse", "refs/tags/"+name)
		deleted, err := m.DeleteTag(dir, name, "")
		if err != nil {
			t.Fatal(err)
		}
		if deleted.RemoteDeleted {
			t.Fatalf("%s: nothing should be deleted remotely", name)
		}
		if revParse(dir, "refs/tags/"+name) != "" {
			t.Fatalf("%s should be gone", name)
		}
		if err := m.RestoreTag(dir, deleted); err != nil {
			t.Fatal(err)
		}
		if after := gitIn(t, dir, "rev-parse", "refs/tags/"+name); after != before {
			t.Fatalf("%s restored to %s, want %s", name, after, before)
		}
	}
	if msg := gitIn(t, dir, "tag", "-l", "--format=%(contents:subject)", "v1"); msg != "release notes\n" {
		t.Fatalf("annotated tag lost its message: %q", msg)
	}
}

func TestDeleteTag_FromRemote(t *testing.T) {
	dir, remote := tagRepoWithRemote(t)
	gitIn(t, dir, "tag", "v1")
	gitIn(t, dir, "push", "-q", "origin", "refs/tags/v1")
	gitIn(t, dir, "tag", "local-only")
	m := openRepo(t, dir)

	deleted, err := m.DeleteTag(dir, "v1", "origin")
	if err != nil {
		t.Fatal(err)
	}
	if !deleted.RemoteDeleted {
		t.Fatal("v1 was on origin and should be reported deleted there")
	}
	if revParse(remote, "refs/tags/v1") != "" {
		t.Fatal("v1 should be gone from origin")
	}
	if revParse(dir, "refs/tags/v1") != "" {
		t.Fatal("v1 should be gone locally")
	}

	deleted, err = m.DeleteTag(dir, "local-only", "origin")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.RemoteDeleted {
		t.Fatal("a tag that was never pushed isn't deleted remotely")
	}
	if revParse(dir, "refs/tags/local-only") != "" {
		t.Fatal("local-only should be gone locally")
	}
}

func TestDeleteTag_UnreachableRemoteKeepsLocalTag(t *testing.T) {
	dir := newRepo(t)
	gitIn(t, dir, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))
	gitIn(t, dir, "tag", "v1")
	m := openRepo(t, dir)

	if _, err := m.DeleteTag(dir, "v1", "origin"); err == nil {
		t.Fatal("an unreachable remote should be an error")
	}
	if revParse(dir, "refs/tags/v1") == "" {
		t.Fatal("the local tag should survive a failed remote delete")
	}
}
