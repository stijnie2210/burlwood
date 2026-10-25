package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type WindowPrefs struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	X      int `json:"x"`
	Y      int `json:"y"`
}

func windowPrefsPath() (string, error) {
	return configPath("window.json")
}

func LoadWindowPrefs() (*WindowPrefs, error) {
	p, err := windowPrefsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var prefs WindowPrefs
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, nil
	}
	if prefs.Width < 800 || prefs.Height < 600 {
		return nil, nil
	}
	return &prefs, nil
}

func SaveWindowPrefs(width, height, x, y int) error {
	p, err := windowPrefsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	prefs := WindowPrefs{Width: width, Height: height, X: x, Y: y}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}
