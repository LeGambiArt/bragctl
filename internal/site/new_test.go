package site

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeGambiArt/bragctl/internal/config"
)

func TestHugoNew(t *testing.T) {
	sitePath, hugoCmd := hugoTestSite(t)

	cfg := &config.Config{}
	cfg.Hugo.Command = hugoCmd
	e := NewHugoEngine(cfg)

	// Create a week entry
	path, err := e.New(context.Background(), sitePath, NewOpts{Kind: "week"})
	if err != nil {
		t.Fatalf("New week: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("week file not created: %v", err)
	}

	// Should contain resolved frontmatter (no raw template syntax)
	data, err := os.ReadFile(path) //nolint:gosec // test
	if err != nil {
		t.Fatalf("read week file: %v", err)
	}
	if strings.Contains(string(data), "{{") {
		t.Error("week file has unresolved template syntax")
	}

	// Year index should have been auto-created
	period := CurrentBiWeeklyPeriod()
	yearIndex := filepath.Join(sitePath, "content", period.Year, "_index.md")
	if _, err := os.Stat(yearIndex); err != nil {
		t.Errorf("year index not auto-created: %v", err)
	}

	// Month index should have been auto-created
	monthIndex := filepath.Join(sitePath, "content", period.Dir, "_index.md")
	if _, err := os.Stat(monthIndex); err != nil {
		t.Errorf("month index not auto-created: %v", err)
	}
}

func TestHugoNewExistingFile(t *testing.T) {
	sitePath, hugoCmd := hugoTestSite(t)

	cfg := &config.Config{}
	cfg.Hugo.Command = hugoCmd
	e := NewHugoEngine(cfg)

	// Create twice — second should return existing path without error
	path1, err := e.New(context.Background(), sitePath, NewOpts{Kind: "week"})
	if err != nil {
		t.Fatalf("first New: %v", err)
	}

	path2, err := e.New(context.Background(), sitePath, NewOpts{Kind: "week"})
	if err != nil {
		t.Fatalf("second New: %v", err)
	}

	if path1 != path2 {
		t.Errorf("paths differ: %q vs %q", path1, path2)
	}
}

func TestHugoNewMonth(t *testing.T) {
	sitePath, hugoCmd := hugoTestSite(t)

	cfg := &config.Config{}
	cfg.Hugo.Command = hugoCmd
	e := NewHugoEngine(cfg)

	path, err := e.New(context.Background(), sitePath, NewOpts{Kind: "month"})
	if err != nil {
		t.Fatalf("New month: %v", err)
	}

	if !strings.HasSuffix(path, "_index.md") {
		t.Errorf("month path should end with _index.md: %s", path)
	}

	// Year index should also be created
	period := CurrentBiWeeklyPeriod()
	yearIndex := filepath.Join(sitePath, "content", period.Year, "_index.md")
	if _, err := os.Stat(yearIndex); err != nil {
		t.Errorf("year index not auto-created for month: %v", err)
	}
}

func TestHugoNewYear(t *testing.T) {
	sitePath, hugoCmd := hugoTestSite(t)

	cfg := &config.Config{}
	cfg.Hugo.Command = hugoCmd
	e := NewHugoEngine(cfg)

	path, err := e.New(context.Background(), sitePath, NewOpts{Kind: "year"})
	if err != nil {
		t.Fatalf("New year: %v", err)
	}

	if !strings.HasSuffix(path, "_index.md") {
		t.Errorf("year path should end with _index.md: %s", path)
	}
}

func TestMarkdownNew(t *testing.T) {
	dir := t.TempDir()
	postsDir := filepath.Join(dir, "posts")
	if err := os.MkdirAll(postsDir, 0o750); err != nil {
		t.Fatal(err)
	}

	e := &MarkdownEngine{}

	// Default (week)
	path, err := e.New(context.Background(), dir, NewOpts{})
	if err != nil {
		t.Fatalf("New default: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("post not created: %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // test
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "title:") {
		t.Error("post missing title frontmatter")
	}
	if !strings.Contains(content, "date:") {
		t.Error("post missing date frontmatter")
	}
	if !strings.Contains(content, "Week") {
		t.Error("post title should contain Week")
	}
}

func TestMarkdownNewPost(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "posts"), 0o750); err != nil {
		t.Fatal(err)
	}

	e := &MarkdownEngine{}

	path, err := e.New(context.Background(), dir, NewOpts{Kind: "post", Title: "my-topic"})
	if err != nil {
		t.Fatalf("New post: %v", err)
	}

	if !strings.Contains(path, "my-topic") {
		t.Errorf("post path should contain title: %s", path)
	}
}

func TestMarkdownNewExisting(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "posts"), 0o750); err != nil {
		t.Fatal(err)
	}

	e := &MarkdownEngine{}

	path1, err := e.New(context.Background(), dir, NewOpts{Kind: "post", Title: "dup"})
	if err != nil {
		t.Fatal(err)
	}
	path2, err := e.New(context.Background(), dir, NewOpts{Kind: "post", Title: "dup"})
	if err != nil {
		t.Fatal(err)
	}

	if path1 != path2 {
		t.Errorf("paths differ: %q vs %q", path1, path2)
	}
}

func TestMarkdownNewUnsupportedKind(t *testing.T) {
	e := &MarkdownEngine{}
	_, err := e.New(context.Background(), t.TempDir(), NewOpts{Kind: "year"})
	if err == nil {
		t.Error("expected error for unsupported kind")
	}
}

// Ensure hugo-bragctl or hugo is available for Hugo tests.
func init() {
	// Pre-warm theme cache if Hugo is available
	cfg := &config.Config{}
	if _, err := cfg.ResolveHugoCommand(); err == nil {
		e := NewHugoEngine(cfg)
		_ = e.ensureThemeCache()
	}
}

// Suppress hugo init output in tests.
func init() { //nolint:gochecknoinits // test helper
	if _, err := exec.LookPath("hugo-bragctl"); err != nil {
		if _, err := exec.LookPath("hugo"); err != nil {
			return
		}
	}
}
