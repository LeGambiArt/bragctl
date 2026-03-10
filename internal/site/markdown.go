package site

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/LeGambiArt/bragctl/internal/ai"
	"github.com/LeGambiArt/bragctl/internal/ui"
	"gopkg.in/yaml.v3"
)

// MarkdownEngine creates plain markdown sites with no external dependencies.
type MarkdownEngine struct{}

// Name returns "markdown".
func (m *MarkdownEngine) Name() string { return "markdown" }

// Init creates a new markdown site.
func (m *MarkdownEngine) Init(_ context.Context, opts InitOpts) error {
	if err := os.MkdirAll(opts.Path, 0o750); err != nil {
		return fmt.Errorf("create site dir: %w", err)
	}

	// Create posts directory
	if err := os.MkdirAll(filepath.Join(opts.Path, "posts"), 0o750); err != nil {
		return fmt.Errorf("create posts dir: %w", err)
	}

	// Write site config
	cfg := Config{
		Title:  opts.Title,
		Author: opts.Author,
		Engine: "markdown",
		AI:     opts.AI,
	}
	if err := writeConfig(opts.Path, &cfg); err != nil {
		return err
	}

	// Write README
	readme := fmt.Sprintf("# %s\n\nA brag document by %s.\n", opts.Title, opts.Author)
	if err := os.WriteFile(filepath.Join(opts.Path, "README.md"), []byte(readme), 0o644); err != nil { //nolint:gosec // user content file
		return fmt.Errorf("write README: %w", err)
	}

	// Write .gitignore
	gitignore := "# bragctl generated (overwritten on each ai launch)\nai-spec.md\nCLAUDE.md\nGEMINI.md\n.cursorrules\n\n# Server state\n.server.pid\n\n# MCP config\n.mcp.json\n.claude/\n.cursor/\n.gemini/\n"
	if err := os.WriteFile(filepath.Join(opts.Path, ".gitignore"), []byte(gitignore), 0o644); err != nil { //nolint:gosec // config file
		return fmt.Errorf("write .gitignore: %w", err)
	}

	// Render context.d/ templates with site data
	if err := ai.RenderContextTemplates(opts.Path, opts.Author, "markdown", opts.Title); err != nil {
		return fmt.Errorf("render context templates: %w", err)
	}

	// Write a sample first post
	date := time.Now().Format("2006-01-02")
	postName := date + "-getting-started.md"
	post := fmt.Sprintf(`---
title: "Getting Started"
date: %s
tags: [meta]
---

Welcome to your brag document! Use this to track your professional
accomplishments, contributions, and impact.
`, date)
	if err := os.WriteFile(filepath.Join(opts.Path, "posts", postName), []byte(post), 0o644); err != nil { //nolint:gosec // user content file
		return fmt.Errorf("write sample post: %w", err)
	}

	// Git init
	if err := gitInit(opts.Path); err != nil {
		return err
	}

	return nil
}

// New creates a new markdown post.
func (m *MarkdownEngine) New(_ context.Context, sitePath string, opts NewOpts) (string, error) {
	kind := opts.Kind
	if kind == "" {
		kind = "week"
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	var filename, title string

	switch kind {
	case "week":
		period := CurrentBiWeeklyPeriod()
		filename = fmt.Sprintf("%s-week-%02d.md", date, period.Week)
		title = fmt.Sprintf("Week %d", period.Week)
	case "post":
		slug := "entry"
		if opts.Title != "" {
			slug = opts.Title
		}
		filename = fmt.Sprintf("%s-%s.md", date, slug)
		title = opts.Title
		if title == "" {
			title = "New Entry"
		}
	default:
		return "", fmt.Errorf("markdown engine supports week and post kinds, not %q", kind)
	}

	fullPath := filepath.Join(sitePath, "posts", filename)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil
	}

	content := fmt.Sprintf("---\ntitle: %q\ndate: %s\ntags: []\n---\n\n", title, date)
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil { //nolint:gosec // user content file
		return "", fmt.Errorf("write post: %w", err)
	}

	return fullPath, nil
}

// Serve starts a simple HTTP server that renders markdown posts.
func (m *MarkdownEngine) Serve(ctx context.Context, sitePath string, opts ServeOpts) error {
	addr := fmt.Sprintf("%s:%d", opts.Bind, opts.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           newMarkdownServer(sitePath),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	ui.Info("Serving markdown site at http://%s:%d", opts.Bind, opts.Port)
	ui.Dim("Press Ctrl+C to stop")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// Validate checks that a directory is a valid markdown site.
func (m *MarkdownEngine) Validate(sitePath string) error {
	cfg, err := loadConfig(sitePath)
	if err != nil {
		return fmt.Errorf("not a bragctl site: %w", err)
	}
	if cfg.Engine != "markdown" {
		return fmt.Errorf("site engine is %q, not markdown", cfg.Engine)
	}
	return nil
}

func writeConfig(sitePath string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal site config: %w", err)
	}
	path := filepath.Join(sitePath, "_config.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // config file
		return fmt.Errorf("write site config: %w", err)
	}
	return nil
}

func loadConfig(sitePath string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(sitePath, "_config.yaml")) //nolint:gosec // config file
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func gitInit(dir string) error {
	if err := gitCmd(dir, "init").Run(); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	if err := gitCmd(dir, "add", ".").Run(); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if err := gitCmd(dir, "-c", "commit.gpgsign=false", "commit", "-m", "Initial bragctl site").Run(); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}
