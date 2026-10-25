package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateConfigDirMovesOldFolder(t *testing.T) {
	base := t.TempDir()
	oldDir := filepath.Join(base, "gitgood")
	newDir := filepath.Join(base, "burlwood")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "repos.json"), []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}

	migrateConfigDir(oldDir, newDir)

	if _, err := os.Stat(filepath.Join(newDir, "repos.json")); err != nil {
		t.Fatalf("repos.json not moved: %v", err)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("old folder still exists: %v", err)
	}
}

func TestMigrateConfigDirKeepsExistingNewFolder(t *testing.T) {
	base := t.TempDir()
	oldDir := filepath.Join(base, "gitgood")
	newDir := filepath.Join(base, "burlwood")
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(newDir, "prefs.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	migrateConfigDir(oldDir, newDir)

	if _, err := os.Stat(filepath.Join(newDir, "prefs.json")); err != nil {
		t.Fatalf("new folder was touched: %v", err)
	}
	if _, err := os.Stat(oldDir); err != nil {
		t.Fatalf("old folder should stay when the new one exists: %v", err)
	}
}

func TestMigrateConfigDirWithoutOldFolder(t *testing.T) {
	base := t.TempDir()
	newDir := filepath.Join(base, "burlwood")

	migrateConfigDir(filepath.Join(base, "gitgood"), newDir)

	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		t.Fatalf("new folder should not be created: %v", err)
	}
}
