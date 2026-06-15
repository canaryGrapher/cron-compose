// Package config loads and saves the CLI's persistent state (API base URL + saved
// session cookie). Lives in ~/.croncompose/config.json with 0600 perms.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config is the on-disk shape.
type Config struct {
	APIBase   string `json:"api_base"`
	SessionID string `json:"session_id,omitempty"` // value of cc_session cookie
}

// Path returns ~/.croncompose/config.json.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".croncompose", "config.json"), nil
}

// Load reads the config file, returning sensible defaults if it doesn't exist.
func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return Config{}, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return Config{APIBase: defaultBase()}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	if c.APIBase == "" {
		c.APIBase = defaultBase()
	}
	return c, nil
}

// Save writes the config back to disk atomically.
func Save(c Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func defaultBase() string {
	if v := os.Getenv("CC_API_BASE"); v != "" {
		return v
	}
	return "http://localhost:8080/api/v1"
}
