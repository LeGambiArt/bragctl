package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteContext(t *testing.T) {
	dir := t.TempDir()

	if err := WriteContext(Claude, dir, "my-site", "markdown"); err != nil {
		t.Fatalf("WriteContext: %v", err)
	}

	// autogen-context.md should exist
	autogenPath := filepath.Join(dir, aiSpecFile)
	content, err := os.ReadFile(autogenPath) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("read autogen-context.md: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("autogen-context.md is empty")
	}

	// Check content includes key sections
	s := string(content)
	for _, want := range []string{
		"context.d/",
		"MCP Tools",
		"Writing Brag Entries",
		"posts/",
	} {
		if !contains(s, want) {
			t.Errorf("content missing %q", want)
		}
	}

	// All three symlinks should exist and point to autogen-context.md
	for _, a := range AllAssistants() {
		linkPath := filepath.Join(dir, a.ContextFile)
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Errorf("%s: not a symlink: %v", a.ContextFile, err)
			continue
		}
		if target != aiSpecFile {
			t.Errorf("%s: symlink target = %q, want %q", a.ContextFile, target, aiSpecFile)
		}

		// Reading the symlink should give the same content
		linkContent, err := os.ReadFile(linkPath) //nolint:gosec // test file
		if err != nil {
			t.Errorf("read %s via symlink: %v", a.ContextFile, err)
			continue
		}
		if string(linkContent) != s {
			t.Errorf("%s content differs from autogen-context.md", a.ContextFile)
		}
	}
}

func TestWriteContextReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()

	// Create a regular file where the symlink should go
	existingPath := filepath.Join(dir, Claude.ContextFile)
	if err := os.WriteFile(existingPath, []byte("old content"), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	if err := WriteContext(Claude, dir, "test", "markdown"); err != nil {
		t.Fatalf("WriteContext: %v", err)
	}

	// Should now be a symlink, not a regular file
	target, err := os.Readlink(existingPath)
	if err != nil {
		t.Fatalf("not a symlink after WriteContext: %v", err)
	}
	if target != aiSpecFile {
		t.Errorf("symlink target = %q, want %q", target, aiSpecFile)
	}
}

func TestWriteContextIdempotent(t *testing.T) {
	dir := t.TempDir()

	// Call twice — should not error
	if err := WriteContext(Claude, dir, "test", "markdown"); err != nil {
		t.Fatalf("first WriteContext: %v", err)
	}
	if err := WriteContext(Cursor, dir, "test", "markdown"); err != nil {
		t.Fatalf("second WriteContext: %v", err)
	}

	// autogen file should still exist and symlinks should work
	for _, a := range AllAssistants() {
		linkPath := filepath.Join(dir, a.ContextFile)
		if _, err := os.Readlink(linkPath); err != nil {
			t.Errorf("%s: not a symlink after double write: %v", a.ContextFile, err)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
