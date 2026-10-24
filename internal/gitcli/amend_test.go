package gitcli

import "testing"

func TestGetAmendInfo_KeepsBodyAndDetectsPushed(t *testing.T) {
	dir, _ := initRepoWithRemote(t)
	writeFile(t, dir, "a.txt", "two\n")
	if _, err := Run(dir, "commit", "-q", "-am", "Fix the thing\n\nWhy it broke.\n\nMore detail."); err != nil {
		t.Fatal(err)
	}

	info, err := GetAmendInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Summary != "Fix the thing" || info.Description != "Why it broke.\n\nMore detail." {
		t.Fatalf("message split wrong: %+v", info)
	}
	if info.Pushed {
		t.Fatal("not pushed yet")
	}

	if _, err := PushBranch(dir, "", false); err != nil {
		t.Fatal(err)
	}
	info, _ = GetAmendInfo(dir)
	if !info.Pushed {
		t.Fatal("HEAD is on its upstream now")
	}
}
