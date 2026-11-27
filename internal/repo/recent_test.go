package repo

import (
	"reflect"
	"testing"
)

func recentPaths(entries []recentEntry) []string {
	paths := []string{}
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	return paths
}

func TestDedupeRecent_WindowsSpellingsAreOneFolder(t *testing.T) {
	entries := []recentEntry{
		{Path: `C:\Users\me\burlwood`, Name: "burlwood"},
		{Path: `C:/Users/me/burlwood`, Name: "burlwood"},
		{Path: `c:\users\ME\burlwood\`, Name: "burlwood"},
		{Path: `C:\Users\me\other`, Name: "other"},
		{Path: `C:\`, Name: "C:"},
	}
	want := []string{`C:\Users\me\burlwood`, `C:\Users\me\other`, `C:\`}
	if got := recentPaths(dedupeRecent(entries, "windows")); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestDedupeRecent_ElsewhereCaseAndBackslashesAreSignificant(t *testing.T) {
	entries := []recentEntry{
		{Path: "/home/me/Repo"},
		{Path: "/home/me/repo"},
		{Path: "/home/me/repo/"},
		{Path: "/home/me/repo"},
	}
	want := []string{"/home/me/Repo", "/home/me/repo"}
	if got := recentPaths(dedupeRecent(entries, "linux")); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
