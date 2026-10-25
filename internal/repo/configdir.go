package repo

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	configDirOnce sync.Once
	configDir     string
	configDirErr  error
)

func configPath(name string) (string, error) {
	configDirOnce.Do(func() {
		base, err := os.UserConfigDir()
		if err != nil {
			configDirErr = err
			return
		}
		configDir = filepath.Join(base, "burlwood")
		migrateConfigDir(filepath.Join(base, "gitgood"), configDir)
	})
	if configDirErr != nil {
		return "", configDirErr
	}
	return filepath.Join(configDir, name), nil
}

// The app was called gitgood before; moving its folder keeps recent repos, prefs and open tabs.
func migrateConfigDir(oldDir, newDir string) {
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		return
	}
	_ = os.Rename(oldDir, newDir)
}
