// Package config handles bragctl configuration and state persistence.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds user preferences.
type Config struct {
	DefaultSite   string    `toml:"default_site"`
	DefaultAI     string    `toml:"default_ai"`
	DefaultEngine string    `toml:"default_engine"`
	MCP           MCPConfig `toml:"mcp"`
}

// MCPConfig holds MCP server settings.
type MCPConfig struct {
	Server string `toml:"server"` // path to what-the-mcp binary
}

// BaseDir returns the bragctl base directory.
// Uses BRAGCTL_HOME env var, falls back to ~/.bragctl.
func BaseDir() string {
	if dir := os.Getenv("BRAGCTL_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bragctl"
	}
	return filepath.Join(home, ".bragctl")
}

// SitesDir returns the directory where sites are stored.
func SitesDir() string {
	return filepath.Join(BaseDir(), "sites")
}

// Path returns the path to the config file.
func Path() string {
	return filepath.Join(BaseDir(), "config.toml")
}

// Load reads the config file. Returns defaults if file doesn't exist.
func Load() (*Config, error) {
	cfg := &Config{
		DefaultEngine: "markdown",
	}

	data, err := os.ReadFile(Path()) //nolint:gosec // config path from known location
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	dir := BaseDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	f, err := os.OpenFile(Path(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return toml.NewEncoder(f).Encode(cfg)
}
