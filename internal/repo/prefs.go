package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type AppPrefs struct {
	Editor                string `json:"editor"`
	AutoFetchEnabled      bool   `json:"autoFetchEnabled"`
	AutoFetchIntervalSecs int    `json:"autoFetchIntervalSecs"`
	CommitGraphLimit      int    `json:"commitGraphLimit"`
	DiffContextLines      int    `json:"diffContextLines"`
	DateFormat            string `json:"dateFormat"` // "relative" | "absolute"
	PullMode              string `json:"pullMode"`   // "fetch" | "ff" | "ff-only" | "rebase" — default action for the toolbar Pull button
}

var (
	prefsCacheMu sync.RWMutex
	prefsCache   *AppPrefs
)

func DefaultPrefs() AppPrefs {
	return AppPrefs{
		Editor:                "",
		AutoFetchEnabled:      true,
		AutoFetchIntervalSecs: 60,
		CommitGraphLimit:      2000,
		DiffContextLines:      3,
		DateFormat:            "relative",
		PullMode:              "ff",
	}
}

func prefsPath() (string, error) {
	return configPath("prefs.json")
}

func LoadPrefs() (AppPrefs, error) {
	prefsCacheMu.RLock()
	if prefsCache != nil {
		p := *prefsCache
		prefsCacheMu.RUnlock()
		return p, nil
	}
	prefsCacheMu.RUnlock()

	p, err := prefsPath()
	if err != nil {
		return DefaultPrefs(), err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		defaults := DefaultPrefs()
		prefsCacheMu.Lock()
		prefsCache = &defaults
		prefsCacheMu.Unlock()
		return defaults, nil
	}
	if err != nil {
		return DefaultPrefs(), err
	}
	loaded := DefaultPrefs()
	if err := json.Unmarshal(data, &loaded); err != nil {
		return DefaultPrefs(), nil
	}
	prefsCacheMu.Lock()
	prefsCache = &loaded
	prefsCacheMu.Unlock()
	return loaded, nil
}

func SavePrefs(prefs AppPrefs) error {
	p, err := prefsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		return err
	}
	saved := prefs
	prefsCacheMu.Lock()
	prefsCache = &saved
	prefsCacheMu.Unlock()
	return nil
}
