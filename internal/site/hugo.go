package site

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gitlab.cee.redhat.com/bragctl/bragctl/internal/ai"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/config"
)

//go:embed templates/hugo/*
var hugoTemplates embed.FS

const (
	defaultThemeRepo   = "https://github.com/alex-shpak/hugo-book"
	defaultThemeCommit = "81a841c92d62f2ed8d9134b0b18623b8b2471661"
)

// HugoEngine creates Hugo-based brag document sites.
type HugoEngine struct {
	cfg *config.Config
}

// NewHugoEngine creates a HugoEngine with config for binary resolution.
func NewHugoEngine(cfg *config.Config) *HugoEngine {
	return &HugoEngine{cfg: cfg}
}

// Name returns "hugo".
func (h *HugoEngine) Name() string { return "hugo" }

// Init creates a new Hugo site.
func (h *HugoEngine) Init(_ context.Context, opts InitOpts) error {
	hugoCmd, err := h.cfg.ResolveHugoCommand()
	if err != nil {
		return err
	}

	// 1. hugo new site <path> --force
	cmd := exec.Command(hugoCmd, "new", "site", opts.Path, "--force") //nolint:gosec // resolved binary
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hugo new site: %w", err)
	}

	// 2. Write _config.yaml (bragctl site config)
	siteCfg := Config{
		Title:  opts.Title,
		Author: opts.Author,
		Engine: "hugo",
		AI:     opts.AI,
	}
	if err := writeConfig(opts.Path, &siteCfg); err != nil {
		return err
	}

	// 3. Write hugo.toml (Hugo site config, replaces the one hugo new creates)
	if err := h.deployTemplate("templates/hugo/hugo.toml", filepath.Join(opts.Path, "hugo.toml"), opts.Author, opts.Title); err != nil {
		return err
	}

	// 4. Create content/posts/ directory
	if err := os.MkdirAll(filepath.Join(opts.Path, "content", "posts"), 0o750); err != nil {
		return fmt.Errorf("create content/posts: %w", err)
	}

	// 5. Deploy archetypes
	archetypes := []string{"week.md", "month.md", "year.md"}
	for _, name := range archetypes {
		src := "templates/hugo/archetypes/" + name
		dst := filepath.Join(opts.Path, "archetypes", name)
		if err := h.copyEmbedded(src, dst); err != nil {
			return fmt.Errorf("deploy archetype %s: %w", name, err)
		}
	}

	// 6. Deploy about.md (with author substitution)
	if err := h.deployTemplate("templates/hugo/about.md", filepath.Join(opts.Path, "content", "about.md"), opts.Author, opts.Title); err != nil {
		return err
	}

	// 7. Write Hugo-specific .gitignore
	if err := h.copyEmbedded("templates/hugo/gitignore", filepath.Join(opts.Path, ".gitignore")); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}

	// 8. Install hugo-book theme as git submodule
	if err := h.installTheme(opts.Path); err != nil {
		return err
	}

	// 9. Render context.d/ templates
	if err := ai.RenderContextTemplates(opts.Path, opts.Author, "hugo", opts.Title); err != nil {
		return fmt.Errorf("render context templates: %w", err)
	}

	// 10. Git init + initial commit
	if err := gitInit(opts.Path); err != nil {
		return err
	}

	return nil
}

// Serve runs hugo server for live preview.
func (h *HugoEngine) Serve(ctx context.Context, sitePath string, opts ServeOpts) error {
	hugoCmd, err := h.cfg.ResolveHugoCommand()
	if err != nil {
		return err
	}

	args := []string{"server", "-D", "--port", fmt.Sprintf("%d", opts.Port)}
	cmd := exec.CommandContext(ctx, hugoCmd, args...) //nolint:gosec // resolved binary + user port
	cmd.Dir = sitePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Validate checks that a directory is a valid Hugo site.
func (h *HugoEngine) Validate(sitePath string) error {
	cfg, err := loadConfig(sitePath)
	if err != nil {
		return fmt.Errorf("not a bragctl site: %w", err)
	}
	if cfg.Engine != "hugo" {
		return fmt.Errorf("site engine is %q, not hugo", cfg.Engine)
	}
	if _, err := os.Stat(filepath.Join(sitePath, "hugo.toml")); err != nil {
		return fmt.Errorf("hugo.toml missing: %w", err)
	}
	return nil
}

// deployTemplate reads an embedded template and writes it with AUTHOR/TITLE substitution.
func (h *HugoEngine) deployTemplate(src, dst, author, title string) error {
	data, err := hugoTemplates.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read template %s: %w", src, err)
	}

	content := string(data)
	content = strings.ReplaceAll(content, "%%AUTHOR%%", author)
	content = strings.ReplaceAll(content, "%%TITLE%%", title)

	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create dir for %s: %w", dst, err)
	}
	if err := os.WriteFile(dst, []byte(content), 0o644); err != nil { //nolint:gosec // user content file
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

// copyEmbedded copies an embedded file to disk without substitution.
func (h *HugoEngine) copyEmbedded(src, dst string) error {
	data, err := hugoTemplates.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read embedded %s: %w", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create dir for %s: %w", dst, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil { //nolint:gosec // template file
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func (h *HugoEngine) themeRepo() string {
	if h.cfg.Hugo.ThemeRepo != "" {
		return h.cfg.Hugo.ThemeRepo
	}
	return defaultThemeRepo
}

func (h *HugoEngine) themeCommit() string {
	if h.cfg.Hugo.ThemeCommit != "" {
		return h.cfg.Hugo.ThemeCommit
	}
	return defaultThemeCommit
}

// gitCmd creates a git command with a clean environment.
// This prevents interference from parent GIT_DIR, GIT_WORK_TREE, etc.
// when running git in a subprocess (e.g., during pre-commit hooks).
func gitCmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	// Remove git env vars that could leak from parent process
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "GIT_DIR=") &&
			!strings.HasPrefix(env, "GIT_WORK_TREE=") &&
			!strings.HasPrefix(env, "GIT_INDEX_FILE=") &&
			!strings.HasPrefix(env, "GIT_OBJECT_DIRECTORY=") &&
			!strings.HasPrefix(env, "GIT_ALTERNATE_OBJECT_DIRECTORIES=") {
			cmd.Env = append(cmd.Env, env)
		}
	}
	return cmd
}

// installTheme adds the theme as a git submodule pinned to a specific commit.
func (h *HugoEngine) installTheme(sitePath string) error {
	// Need a git repo first for submodule to work
	if err := gitCmd(sitePath, "init").Run(); err != nil {
		return fmt.Errorf("git init (for submodule): %w", err)
	}

	repo := h.themeRepo()
	if err := gitCmd(sitePath, "submodule", "add", repo, "themes/hugo-book").Run(); err != nil { //nolint:gosec // user-configured repo URL
		return fmt.Errorf("install theme: %w", err)
	}

	// Pin to specific commit
	commit := h.themeCommit()
	themePath := filepath.Join(sitePath, "themes", "hugo-book")

	if err := gitCmd(themePath, "checkout", commit).Run(); err != nil { //nolint:gosec // default or configured hash
		// Commit not in clone — fetch it first (shallow clone case)
		if fetchErr := gitCmd(themePath, "fetch", "--depth=1", "origin", commit).Run(); fetchErr != nil { //nolint:gosec // same
			return fmt.Errorf("pin theme to %s: checkout failed (%w), fetch also failed (%w)", commit[:12], err, fetchErr)
		}
		if err := gitCmd(themePath, "checkout", commit).Run(); err != nil { //nolint:gosec // retry after fetch
			return fmt.Errorf("pin theme to %s: %w", commit[:12], err)
		}
	}

	return nil
}
