// Package config handles bragctl configuration and state persistence.
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds global bragctl preferences.
type Config struct {
	DefaultSite string     `toml:"default_site,omitempty"`
	MCP         MCPConfig  `toml:"mcp,omitempty"`
	Hugo        HugoConfig `toml:"hugo,omitempty"`
}

// HugoConfig describes how to find the Hugo binary and theme settings.
type HugoConfig struct {
	// Command is an explicit path to the Hugo binary.
	// Default: resolved via ResolveHugoCommand().
	Command string `toml:"command,omitempty"`

	// ThemeRepo is the git URL for the Hugo theme.
	// Default: hugo-book.
	ThemeRepo string `toml:"theme_repo,omitempty"`

	// ThemeCommit pins the theme to a specific commit.
	// Default: known-good hugo-book commit.
	ThemeCommit string `toml:"theme_commit,omitempty"`
}

// MCPConfig describes how to launch wtmcp.
type MCPConfig struct {
	// Command is the path to the wtmcp binary.
	// Default: "wtmcp" (from PATH).
	Command string `toml:"command,omitempty"`

	// Workdir is the wtmcp working directory.
	// Default: same as bragctl BaseDir().
	Workdir string `toml:"workdir,omitempty"`

	// Args are extra flags passed to wtmcp.
	Args []string `toml:"args,omitempty"`
}

// ResolveHugoCommand returns the Hugo binary to use.
// Three-tier lookup: config override → hugo-bragctl in PATH → hugo in PATH.
func (c *Config) ResolveHugoCommand() (string, error) {
	if c.Hugo.Command != "" {
		return c.Hugo.Command, nil
	}
	if path, err := exec.LookPath("hugo-bragctl"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("hugo"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("hugo not found: install Hugo or set [hugo] command in %s", Path())
}

// MCPCommand returns the resolved wtmcp command.
func (c *Config) MCPCommand() string {
	if c.MCP.Command != "" {
		return c.MCP.Command
	}
	return "wtmcp"
}

// MCPWorkdir returns the resolved wtmcp workdir.
func (c *Config) MCPWorkdir() string {
	if c.MCP.Workdir != "" {
		return c.MCP.Workdir
	}
	return BaseDir()
}

// MCPArgs returns the full argument list for wtmcp,
// including --workdir and any extra configured args.
func (c *Config) MCPArgs() []string {
	args := []string{"--workdir", c.MCPWorkdir()}
	args = append(args, c.MCP.Args...)
	return args
}

// CredentialsDir returns the credentials directory for a given provider.
// e.g. CredentialsDir("google") → ~/.bragctl/credentials/google/
func CredentialsDir(provider string) string {
	return filepath.Join(BaseDir(), "credentials", provider)
}

// validateBragctlHome validates the BRAGCTL_HOME path for security.
// It must be an absolute path and must not be in system directories.
func validateBragctlHome(dir string) error {
	if dir == "" {
		return fmt.Errorf("BRAGCTL_HOME cannot be empty")
	}

	// Must be absolute path
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("BRAGCTL_HOME must be an absolute path: %q", dir)
	}

	// Reject paths containing .. components (before cleaning)
	if strings.Contains(dir, "..") {
		return fmt.Errorf("BRAGCTL_HOME cannot contain '..' components: %q", dir)
	}

	// Clean the path for system directory checks
	cleaned := filepath.Clean(dir)

	// Reject system directories (prevent writing to sensitive locations)
	systemDirs := []string{"/etc", "/usr", "/bin", "/sbin", "/var", "/dev", "/proc", "/sys", "/boot", "/lib", "/lib64"}
	for _, sysDir := range systemDirs {
		if cleaned == sysDir || strings.HasPrefix(cleaned, sysDir+"/") {
			return fmt.Errorf("BRAGCTL_HOME cannot be in system directory %s: %q", sysDir, dir)
		}
	}

	return nil
}

// BaseDir returns the bragctl base directory.
// Uses BRAGCTL_HOME env var, falls back to ~/.bragctl.
// If BRAGCTL_HOME is invalid, warns to stderr and uses default.
func BaseDir() string {
	if dir := os.Getenv("BRAGCTL_HOME"); dir != "" {
		if err := validateBragctlHome(dir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid BRAGCTL_HOME (%v), using default\n", err)
			// Fall through to default
		} else {
			return dir
		}
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

var siteNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// ValidateSiteName validates that a site name is safe to use in filesystem paths.
// It rejects path traversal attempts, special directory names, and unsafe characters.
func ValidateSiteName(name string) error {
	if name == "" {
		return fmt.Errorf("site name cannot be empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("site name cannot be '.' or '..'")
	}
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("site name cannot contain null bytes")
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("site name cannot contain path separators")
	}
	if filepath.Base(name) != name {
		return fmt.Errorf("invalid site name: %q contains path components", name)
	}
	if !siteNamePattern.MatchString(name) {
		return fmt.Errorf("site name must start with alphanumeric and contain only alphanumeric, dots, hyphens, or underscores: %q", name)
	}
	return nil
}

// SitePath validates a site name and returns its full path under SitesDir().
// Returns an error if the name is invalid or resolves outside SitesDir().
func SitePath(name string) (string, error) {
	if err := ValidateSiteName(name); err != nil {
		return "", err
	}

	sitesDir := SitesDir()
	sitePath := filepath.Join(sitesDir, name)

	// Defense-in-depth: verify the resolved path is still under sitesDir
	relPath, err := filepath.Rel(sitesDir, sitePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve site path: %w", err)
	}
	if strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || relPath == ".." {
		return "", fmt.Errorf("site path escapes sites directory: %q", name)
	}

	return sitePath, nil
}

// Path returns the path to the config file.
func Path() string {
	return filepath.Join(BaseDir(), "config.toml")
}

// Load reads the config file. Returns defaults if file doesn't exist.
// Also ensures the standard directory structure exists.
func Load() (*Config, error) {
	cfg := &Config{}

	if err := EnsureDirs(); err != nil {
		return nil, err
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

// EnsureDirs creates the standard bragctl directory structure.
// Safe to call repeatedly — only creates directories that don't exist.
func EnsureDirs() error {
	base := BaseDir()
	dirs := []string{
		base,
		filepath.Join(base, "sites"),
		filepath.Join(base, "plugins"),
		filepath.Join(base, "env.d"),
		filepath.Join(base, "credentials"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
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
