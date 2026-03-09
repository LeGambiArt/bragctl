package site

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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
	gitignore := "# bragctl generated (overwritten on each ai launch)\nautogen-context.md\nCLAUDE.md\nGEMINI.md\n.cursorrules\n\n# MCP config\n.mcp.json\n.claude/\n.cursor/\n.gemini/\n"
	if err := os.WriteFile(filepath.Join(opts.Path, ".gitignore"), []byte(gitignore), 0o644); err != nil { //nolint:gosec // config file
		return fmt.Errorf("write .gitignore: %w", err)
	}

	// Create context.d/ with starter file
	ctxDir := filepath.Join(opts.Path, "context.d")
	if err := os.MkdirAll(ctxDir, 0o750); err != nil {
		return fmt.Errorf("create context.d: %w", err)
	}
	ctxExample := `# Custom Context

Add your custom instructions here. This file is never overwritten
by bragctl — it persists across regenerations.

Examples of what to put here:
- Team conventions and project names
- Preferred writing style for brag entries
- JQL queries you use frequently
- Sprint naming patterns
`
	if err := os.WriteFile(filepath.Join(ctxDir, "example.md"), []byte(ctxExample), 0o644); err != nil { //nolint:gosec // user content
		return fmt.Errorf("write context.d/example.md: %w", err)
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
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// Initial commit
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial bragctl site")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	return nil
}
