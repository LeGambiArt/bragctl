package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.json")

	testData := map[string]string{"key": "value"}
	if err := writeJSON(path, testData); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}

	// Verify permissions are 0o600 (owner read/write only)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file permissions = %o, want 0o600", info.Mode().Perm())
	}
}

func TestSetupCreatesConfig(t *testing.T) {
	dir := t.TempDir()

	err := Setup("claude", dir, "wtmcp", []string{"--workdir", dir})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// Verify .mcp.json was created
	mcpPath := filepath.Join(dir, ".mcp.json")
	if _, err := os.Stat(mcpPath); err != nil {
		t.Errorf(".mcp.json not created: %v", err)
	}

	// Verify it has correct permissions
	info, err := os.Stat(mcpPath)
	if err != nil {
		t.Fatalf("stat .mcp.json: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf(".mcp.json permissions = %o, want 0o600", info.Mode().Perm())
	}
}

func TestSetupCursor(t *testing.T) {
	dir := t.TempDir()

	err := Setup("cursor", dir, "wtmcp", []string{"--workdir", dir})
	if err != nil {
		t.Fatalf("Setup cursor: %v", err)
	}

	// Verify .cursor/mcp.json was created
	mcpPath := filepath.Join(dir, ".cursor", "mcp.json")
	if _, err := os.Stat(mcpPath); err != nil {
		t.Errorf(".cursor/mcp.json not created: %v", err)
	}

	// Verify permissions
	info, err := os.Stat(mcpPath)
	if err != nil {
		t.Fatalf("stat .cursor/mcp.json: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf(".cursor/mcp.json permissions = %o, want 0o600", info.Mode().Perm())
	}
}

func TestSetupGemini(t *testing.T) {
	dir := t.TempDir()

	err := Setup("gemini", dir, "wtmcp", []string{"--workdir", dir})
	if err != nil {
		t.Fatalf("Setup gemini: %v", err)
	}

	// Verify .gemini/settings.json was created
	settingsPath := filepath.Join(dir, ".gemini", "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Errorf(".gemini/settings.json not created: %v", err)
	}

	// Verify permissions
	info, err := os.Stat(settingsPath)
	if err != nil {
		t.Fatalf("stat .gemini/settings.json: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf(".gemini/settings.json permissions = %o, want 0o600", info.Mode().Perm())
	}
}
