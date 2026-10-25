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
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &Session{}, nil
	}
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return &Session{}, nil
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
