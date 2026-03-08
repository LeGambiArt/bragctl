// Package config handles bragctl configuration and state persistence.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds global bragctl preferences.
type Config struct {
	DefaultSite string    `toml:"default_site,omitempty"`
	MCP         MCPConfig `toml:"mcp,omitempty"`
}

// MCPConfig describes how to launch what-the-mcp.
type MCPConfig struct {
	// Command is the path to the what-the-mcp binary.
	// Default: "what-the-mcp" (from PATH).
	Command string `toml:"command,omitempty"`

	// Workdir is the what-the-mcp working directory.
	// Default: same as bragctl BaseDir().
	Workdir string `toml:"workdir,omitempty"`

	// Args are extra flags passed to what-the-mcp.
	Args []string `toml:"args,omitempty"`
}

// MCPCommand returns the resolved what-the-mcp command.
func (c *Config) MCPCommand() string {
	if c.MCP.Command != "" {
		return c.MCP.Command
	}
	return "what-the-mcp"
}

// MCPWorkdir returns the resolved what-the-mcp workdir.
func (c *Config) MCPWorkdir() string {
	if c.MCP.Workdir != "" {
		return c.MCP.Workdir
	}
	return BaseDir()
}

// MCPArgs returns the full argument list for what-the-mcp,
// including --workdir and any extra configured args.
func (c *Config) MCPArgs() []string {
	args := []string{"--workdir", c.MCPWorkdir()}
	args = append(args, c.MCP.Args...)
	return args
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
	cfg := &Config{}

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
