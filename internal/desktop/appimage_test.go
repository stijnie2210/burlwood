package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func init() {
	shellRescanDelay = 0
}

func TestIntegrateAppImage(t *testing.T) {
	data := t.TempDir()
	appImage := "/home/u/Apps/Burlwood $1.AppImage"
	icon := []byte("png")

	if err := integrateAppImage(data, appImage, icon); err != nil {
		t.Fatal(err)
	}
	entry, err := os.ReadFile(filepath.Join(data, "applications", entryName))
	if err != nil {
		t.Fatal(err)
	}
	iconPath := filepath.Join(data, "burlwood", "burlwood.png")
	for _, want := range []string{
		`Exec="/home/u/Apps/Burlwood \$1.AppImage"`,
		"TryExec=" + appImage,
		"Icon=" + iconPath,
		"StartupWMClass=" + AppID,
	} {
		if !strings.Contains(string(entry), want) {
			t.Errorf("entry lacks %q:\n%s", want, entry)
		}
	}
	if got, _ := os.ReadFile(iconPath); string(got) != "png" {
		t.Errorf("icon = %q", got)
	}

	moved := "/home/u/Burlwood.AppImage"
	if err := integrateAppImage(data, moved, icon); err != nil {
		t.Fatal(err)
	}
	entry, _ = os.ReadFile(filepath.Join(data, "applications", entryName))
	if !strings.Contains(string(entry), "TryExec="+moved) {
		t.Errorf("entry not updated after the AppImage moved:\n%s", entry)
	}
}

func TestIntegrateAppImageSkipsWhenIntegratedElsewhere(t *testing.T) {
	data := t.TempDir()
	apps := filepath.Join(data, "applications")
	appImage := "/home/u/Applications/Burlwood.AppImage"
	if err := os.MkdirAll(apps, 0o755); err != nil {
		t.Fatal(err)
	}
	other := "[Desktop Entry]\nExec=" + appImage + "\n"
	if err := os.WriteFile(filepath.Join(apps, "appimagekit_abc-Burlwood.desktop"), []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := integrateAppImage(data, appImage, []byte("png")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(apps, entryName)); !os.IsNotExist(err) {
		t.Errorf("wrote %s although another entry already launches the AppImage", entryName)
	}
}
