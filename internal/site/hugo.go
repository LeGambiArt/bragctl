package site

import (
	"context"
	"embed"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// sanitizeTOMLValue escapes special characters in a string value for safe TOML substitution.
// It removes or escapes characters that could break TOML structure.
func sanitizeTOMLValue(s string) string {
	// Replace backslash first (to avoid double-escaping)
	s = strings.ReplaceAll(s, `\`, `\\`)
	// Escape double quotes (TOML strings use double quotes)
	s = strings.ReplaceAll(s, `"`, `\"`)
	// Remove or escape newlines (TOML strings can't contain literal newlines)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	// Remove tabs (or could escape as \t)
	s = strings.ReplaceAll(s, "\t", " ")
	return s
}

// deployTemplate reads an embedded template and writes it with AUTHOR/TITLE substitution.
// Values are sanitized to prevent TOML/YAML injection.
func (h *HugoEngine) deployTemplate(src, dst, author, title string) error {
	data, err := hugoTemplates.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read template %s: %w", src, err)
	}

	now := time.Now()
	content := string(data)
	// Sanitize author and title to prevent TOML/YAML injection
	content = strings.ReplaceAll(content, "%%AUTHOR%%", sanitizeTOMLValue(author))
	content = strings.ReplaceAll(content, "%%TITLE%%", sanitizeTOMLValue(title))
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

var gitCommitPattern = regexp.MustCompile(`^[0-9a-f]{7,64}$`)

// validateThemeURL validates that a theme repository URL is safe.
// Only HTTPS URLs to public repositories are allowed.
func validateThemeURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("theme URL cannot be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Require HTTPS scheme only
	if u.Scheme != "https" {
		return fmt.Errorf("theme URL must use https:// scheme, got %q", u.Scheme)
	}

	// Reject URLs with embedded credentials
	if u.User != nil {
		return fmt.Errorf("theme URL cannot contain credentials")
	}

	// Require a host
	if u.Host == "" {
		return fmt.Errorf("theme URL must have a host")
	}

	// Reject localhost and loopback addresses (SSRF prevention)
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "[::1]" {
		return fmt.Errorf("theme URL cannot use localhost")
	}

	// Reject private IP ranges (basic SSRF prevention)
	if strings.HasPrefix(host, "10.") || strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "172.16.") || strings.HasPrefix(host, "172.17.") ||
		strings.HasPrefix(host, "172.18.") || strings.HasPrefix(host, "172.19.") ||
		strings.HasPrefix(host, "172.20.") || strings.HasPrefix(host, "172.21.") ||
		strings.HasPrefix(host, "172.22.") || strings.HasPrefix(host, "172.23.") ||
		strings.HasPrefix(host, "172.24.") || strings.HasPrefix(host, "172.25.") ||
		strings.HasPrefix(host, "172.26.") || strings.HasPrefix(host, "172.27.") ||
		strings.HasPrefix(host, "172.28.") || strings.HasPrefix(host, "172.29.") ||
		strings.HasPrefix(host, "172.30.") || strings.HasPrefix(host, "172.31.") {
		return fmt.Errorf("theme URL cannot use private IP address")
	}

	// Reject non-standard ports (legitimate git hosting uses standard HTTPS port 443)
	if u.Port() != "" {
		return fmt.Errorf("theme URL cannot specify a custom port")
	}

	return nil
}

// validateThemeCommit validates that a commit string is a valid git SHA.
func validateThemeCommit(commit string) error {
	if commit == "" {
		return fmt.Errorf("commit cannot be empty")
	}
	if !gitCommitPattern.MatchString(commit) {
		return fmt.Errorf("commit must be a valid git SHA (7-64 hex characters): %q", commit)
	}
	return nil
}

func (h *HugoEngine) themeRepo() string {
	if h.cfg.Hugo.ThemeRepo != "" {
		// Validate custom theme repo
		if err := validateThemeURL(h.cfg.Hugo.ThemeRepo); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid theme_repo in config (%v), using default\n", err)
			return defaultThemeRepo
		}
		return h.cfg.Hugo.ThemeRepo
	}
	return defaultThemeRepo
}

func (h *HugoEngine) themeCommit() string {
	if h.cfg.Hugo.ThemeCommit != "" {
		// Validate custom theme commit
		if err := validateThemeCommit(h.cfg.Hugo.ThemeCommit); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid theme_commit in config (%v), using default\n", err)
			return defaultThemeCommit
		}
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
