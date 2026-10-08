package main

// The settings, in %APPDATA%\DeskBuddy\config.json: where the Desk Buddy server is and its key
// (both filled in by "Connect", from the one-time code the Mac app shows), and Pause.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	Server string `json:"server"`
	Key    string `json:"key"`
	Paused bool   `json:"paused"`
}

var (
	cfgMu sync.Mutex
	cfg   Config
)

func configDir() string {
	d := os.Getenv("APPDATA")
	if d == "" {
		d, _ = os.UserConfigDir()
	}
	return filepath.Join(d, "DeskBuddy")
}

func loadConfig() {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	if b, err := os.ReadFile(filepath.Join(configDir(), "config.json")); err == nil {
		json.Unmarshal(b, &cfg)
	}
}

func saveConfig(c Config) error {
	cfgMu.Lock()
	cfg = c
	cfgMu.Unlock()
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	tmp := filepath.Join(configDir(), "config.json.new")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(configDir(), "config.json"))
}

func current() Config {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	return cfg
}

func connected() bool { c := current(); return c.Server != "" && c.Key != "" }
