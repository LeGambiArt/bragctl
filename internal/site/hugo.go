package site

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/LeGambiArt/bragctl/internal/ai"
	"github.com/LeGambiArt/bragctl/internal/config"
	"github.com/LeGambiArt/bragctl/internal/ui"
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
	if err := ui.Spin("Creating Hugo site", func() error {
		cmd := exec.Command(hugoCmd, "new", "site", opts.Path, "--force") //nolint:gosec // resolved binary
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	}); err != nil {
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

	// 7. Deploy static assets (logo, favicon)
	staticFiles := map[string]string{
		"templates/hugo/static/favicon.svg":     filepath.Join(opts.Path, "static", "favicon.svg"),
		"templates/hugo/static/images/logo.svg": filepath.Join(opts.Path, "static", "images", "logo.svg"),
	}
	for src, dst := range staticFiles {
		if err := h.copyEmbedded(src, dst); err != nil {
			return fmt.Errorf("deploy static %s: %w", filepath.Base(src), err)
		}
	}

	// 8. Write Hugo-specific .gitignore
	if err := h.copyEmbedded("templates/hugo/gitignore", filepath.Join(opts.Path, ".gitignore")); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}

	// 8. Install hugo-book theme (copy from shared cache)
	if err := ui.Spin("Installing theme", func() error {
		return h.installTheme(opts.Path)
	}); err != nil {
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

	args := []string{"server", "-D", "--port", fmt.Sprintf("%d", opts.Port), "--bind", opts.Bind}
	cmd := exec.CommandContext(ctx, hugoCmd, args...) //nolint:gosec // resolved binary + user args
	cmd.Dir = sitePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// New creates a new Hugo content entry using archetypes.
func (h *HugoEngine) New(_ context.Context, sitePath string, opts NewOpts) (string, error) {
	hugoCmd, err := h.cfg.ResolveHugoCommand()
	if err != nil {
		return "", err
	}

	kind := opts.Kind
	if kind == "" {
		kind = "week"
	}

	var contentPath string

	switch kind {
	case "week":
		period := CurrentBiWeeklyPeriod()
		contentPath = filepath.Join("content", period.Dir, period.Filename)

		// Auto-create year index if missing
		yearIndex := filepath.Join("content", period.Year, "_index.md")
		if _, err := os.Stat(filepath.Join(sitePath, yearIndex)); os.IsNotExist(err) {
			if err := h.hugoNew(hugoCmd, sitePath, "year", yearIndex); err != nil {
				return "", fmt.Errorf("create year index: %w", err)
			}
		}

		// Auto-create month index if missing
		monthIndex := filepath.Join("content", period.Dir, "_index.md")
		if _, err := os.Stat(filepath.Join(sitePath, monthIndex)); os.IsNotExist(err) {
			if err := h.hugoNew(hugoCmd, sitePath, "month", monthIndex); err != nil {
				return "", fmt.Errorf("create month index: %w", err)
			}
		}

	case "month":
		period := CurrentBiWeeklyPeriod()
		contentPath = filepath.Join("content", period.Dir, "_index.md")

		// Auto-create year index if missing
		yearIndex := filepath.Join("content", period.Year, "_index.md")
		if _, err := os.Stat(filepath.Join(sitePath, yearIndex)); os.IsNotExist(err) {
			if err := h.hugoNew(hugoCmd, sitePath, "year", yearIndex); err != nil {
				return "", fmt.Errorf("create year index: %w", err)
			}
		}

	case "year":
		period := CurrentBiWeeklyPeriod()
		contentPath = filepath.Join("content", period.Year, "_index.md")

	default:
		return "", fmt.Errorf("unknown kind: %s (use week, month, or year)", kind)
	}

	// Check if file already exists
	fullPath := filepath.Join(sitePath, contentPath)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil
	}

	if err := h.hugoNew(hugoCmd, sitePath, kind, contentPath); err != nil {
		return "", err
	}

	return filepath.Join(sitePath, contentPath), nil
}

// hugoNew runs hugo new -k <kind> <path> in the site directory.
func (h *HugoEngine) hugoNew(hugoCmd, sitePath, kind, contentPath string) error {
	cmd := exec.Command(hugoCmd, "new", "-k", kind, contentPath) //nolint:gosec // resolved binary
	cmd.Dir = sitePath
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hugo new -k %s %s: %w", kind, contentPath, err)
	}
	return nil
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

	now := time.Now()
	content := string(data)
	content = strings.ReplaceAll(content, "%%AUTHOR%%", author)
	content = strings.ReplaceAll(content, "%%TITLE%%", title)
	content = strings.ReplaceAll(content, "%%DATE%%", now.Format("2006-01-02"))
	content = strings.ReplaceAll(content, "%%YEAR%%", now.Format("2006"))
	content = strings.ReplaceAll(content, "%%LONGDATE%%", now.Format("January 2, 2006"))

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

// themeCacheDir returns the path to the shared theme cache.
func themeCacheDir() string {
	return filepath.Join(config.BaseDir(), "themes", "hugo-book")
}

// ensureThemeCache clones the theme to ~/.bragctl/themes/hugo-book/ if not present,
// pinned to the configured commit.
func (h *HugoEngine) ensureThemeCache() error {
	cacheDir := themeCacheDir()

	if _, err := os.Stat(cacheDir); err == nil {
		return nil // already cached
	}

	repo := h.themeRepo()
	if err := os.MkdirAll(filepath.Dir(cacheDir), 0o750); err != nil {
		return fmt.Errorf("create themes dir: %w", err)
	}

	if err := gitCmd(".", "clone", repo, cacheDir).Run(); err != nil {
		return fmt.Errorf("clone theme: %w", err)
	}

	// Pin to specific commit
	commit := h.themeCommit()
	if err := gitCmd(cacheDir, "checkout", commit).Run(); err != nil {
		if fetchErr := gitCmd(cacheDir, "fetch", "--depth=1", "origin", commit).Run(); fetchErr != nil {
			return fmt.Errorf("pin theme to %s: checkout (%w), fetch (%w)", commit[:12], err, fetchErr)
		}
		if err := gitCmd(cacheDir, "checkout", commit).Run(); err != nil {
			return fmt.Errorf("pin theme to %s: %w", commit[:12], err)
		}
	}

	return nil
}

// installTheme copies the cached theme into the site's themes/ directory.
func (h *HugoEngine) installTheme(sitePath string) error {
	if err := h.ensureThemeCache(); err != nil {
		return err
	}

	dst := filepath.Join(sitePath, "themes", "hugo-book")
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create themes dir: %w", err)
	}

	// Copy theme files (excluding .git) from cache to site
	return copyDir(themeCacheDir(), dst)
}

// copyDir recursively copies src to dst, skipping .git directories.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o750)
		}

		data, err := os.ReadFile(path) //nolint:gosec // copying from known cache dir
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
