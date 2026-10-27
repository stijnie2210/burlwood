package desktop

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const entryName = "burlwood.desktop"

var shellRescanDelay = time.Second

// An AppImage that is only run never gets a .desktop file, and GNOME on Wayland shows a window's icon only
// through one, so Burlwood installs its own.
func integrateAppImage(dataDir, appImage string, icon []byte) error {
	appsDir := filepath.Join(dataDir, "applications")
	if integratedElsewhere(appsDir, appImage) {
		return nil
	}
	_, statErr := os.Stat(appsDir)
	createdAppsDir := os.IsNotExist(statErr)
	iconPath := filepath.Join(dataDir, "burlwood", "burlwood.png")
	if err := writeIfChanged(iconPath, icon); err != nil {
		return err
	}
	entryPath := filepath.Join(appsDir, entryName)
	if err := writeIfChanged(entryPath, []byte(desktopEntry(appImage, iconPath))); err != nil {
		return err
	}
	// GNOME Shell starts watching a new applications folder only after it has seen the folder appear, and
	// misses a file written right after; touching it once it watches makes the Shell pick it up.
	if createdAppsDir {
		time.Sleep(shellRescanDelay)
		now := time.Now()
		return os.Chtimes(entryPath, now, now)
	}
	return nil
}

func desktopEntry(appImage, iconPath string) string {
	return strings.Join([]string{
		"[Desktop Entry]",
		"Name=Burlwood",
		"Exec=" + quoteExec(appImage),
		"TryExec=" + appImage,
		"Icon=" + iconPath,
		"Type=Application",
		"Categories=Development;",
		"Terminal=false",
		"StartupWMClass=burlwood",
		"",
	}, "\n")
}

func quoteExec(path string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`).Replace(path)
	return `"` + escaped + `"`
}

// AppImageLauncher and Gear Lever write their own entry for the AppImage; a second one would show Burlwood twice.
func integratedElsewhere(appsDir, appImage string) bool {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() == entryName || !strings.HasSuffix(e.Name(), ".desktop") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(appsDir, e.Name()))
		if err == nil && bytes.Contains(data, []byte(appImage)) {
			return true
		}
	}
	return false
}

func writeIfChanged(path string, data []byte) error {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
