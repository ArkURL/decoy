// Package config loads optional user configuration and key bindings.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user-tunable settings. All fields are optional.
type Config struct {
	IdleSeconds int                 `json:"idleSeconds"`
	FakeTitle   string              `json:"fakeTitle"`
	Keys        map[string][]string `json:"keys"`
}

// Default returns the built-in defaults.
func Default() Config {
	return Config{
		IdleSeconds: 30,
		FakeTitle:   "~/dev/api — vim",
	}
}

// Path returns the config file location.
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, "cloak", "config.json")
}

// Load reads the config file, falling back to defaults for missing values.
func Load() Config {
	c := Default()
	p := Path()
	if p == "" {
		return c
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return c
	}
	var file Config
	if err := json.Unmarshal(data, &file); err != nil {
		return c
	}
	if file.IdleSeconds > 0 {
		c.IdleSeconds = file.IdleSeconds
	}
	if file.FakeTitle != "" {
		c.FakeTitle = file.FakeTitle
	}
	if len(file.Keys) > 0 {
		c.Keys = file.Keys
	}
	return c
}
