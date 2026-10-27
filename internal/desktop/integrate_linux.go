package desktop

import (
	"os"
	"path/filepath"
)

func IntegrateAppImage(icon []byte) {
	appImage := os.Getenv("APPIMAGE")
	if appImage == "" {
		return
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	if err := integrateAppImage(dataDir, appImage, icon); err != nil {
		println("Desktop integration:", err.Error())
	}
}
