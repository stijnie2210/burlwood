package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type SessionTab struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type Session struct {
	Tabs        []SessionTab `json:"tabs"`
	ActiveIndex int          `json:"activeIndex"`
}

func sessionPath() (string, error) {
	return configPath("session.json")
}

func LoadSession() (*Session, error) {
	p, err := sessionPath()
	if err != nil {
		return nil, err
	}
	// Tabs must never be nil: it reaches the frontend as null, and restoreSession maps over it.
	s := Session{Tabs: []SessionTab{}}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return &Session{Tabs: []SessionTab{}}, nil
	}
	if s.Tabs == nil {
		s.Tabs = []SessionTab{}
	}
	return &s, nil
}

func SaveSession(tabs []SessionTab, activeIndex int) error {
	p, err := sessionPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	s := Session{Tabs: tabs, ActiveIndex: activeIndex}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// Only a gone repo may lose its tab: other open failures (a slow shell, git off PATH) can be transient.
// A missing parent folder usually means an unmounted drive or share, so that repo is kept.
func RepoGone(root string) bool {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		_, err := os.Stat(filepath.Dir(root))
		return err == nil
	}
	_, err := os.Stat(filepath.Join(root, ".git"))
	return os.IsNotExist(err)
}
