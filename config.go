package environment

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// GlobalConfig wraps read/write access to ~/.env/config.toml.
type GlobalConfig struct {
	BasePath string `toml:"base-path,omitempty"`
	AptMirror string `toml:"apt.mirror,omitempty"` // optional mirror override for apt
	path     string `toml:"-"`
}

// Config loads the global config file.
func Config() (*GlobalConfig, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	cfg := &GlobalConfig{path: path}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read: %w", err)
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}

	return cfg, nil
}

// Set writes a key/value into the global config.
func (c *GlobalConfig) Set(key, value string) error {
	switch key {
	case "base-path":
		c.BasePath = value
	case "apt.mirror":
		c.AptMirror = value
	default:
		return fmt.Errorf("config: unknown key %q", key)
	}
	return c.save()
}

// Get returns the value for a config key.
func (c *GlobalConfig) Get(key string) (string, error) {
	switch key {
	case "base-path":
		return c.BasePath, nil
	case "apt.mirror":
		return c.AptMirror, nil
	default:
		return "", fmt.Errorf("config: unknown key %q", key)
	}
}

// Unset clears a config key back to its default.
func (c *GlobalConfig) Unset(key string) error {
	return c.Set(key, "")
}

func (c *GlobalConfig) save() error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return fmt.Errorf("config: mkdir: %w", err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(c); err != nil {
		return fmt.Errorf("config: encode: %w", err)
	}

	return os.WriteFile(c.path, buf.Bytes(), 0644)
}

func loadConfig() (*GlobalConfig, error) {
	return Config()
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".env", "config.toml"), nil
}