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

func TestHugoEngineName(t *testing.T) {
	e := NewHugoEngine(&config.Config{})
	if e.Name() != "hugo" {
		t.Errorf("Name() = %q, want %q", e.Name(), "hugo")
	}
}

func TestHugoTemplatesEmbedded(t *testing.T) {
	// Verify all expected templates are embedded
	files := []string{
		"templates/hugo/hugo.toml",
		"templates/hugo/about.md",
		"templates/hugo/gitignore",
		"templates/hugo/archetypes/week.md",
		"templates/hugo/archetypes/month.md",
		"templates/hugo/archetypes/year.md",
		"templates/hugo/static/favicon.svg",
		"templates/hugo/static/images/logo.svg",
	}
	for _, f := range files {
		data, err := hugoTemplates.ReadFile(f)
		if err != nil {
			t.Errorf("embedded file %s not found: %v", f, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("embedded file %s is empty", f)
		}
	}
}

func TestHugoDeployTemplate(t *testing.T) {
	dir := t.TempDir()
	e := NewHugoEngine(&config.Config{})

	dst := filepath.Join(dir, "hugo.toml")
	if err := e.deployTemplate("templates/hugo/hugo.toml", dst, "Alice", "Alice's Brags"); err != nil {
		t.Fatalf("deployTemplate: %v", err)
	}

	data, err := os.ReadFile(dst) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("read deployed file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `author = "Alice"`) {
		t.Error("hugo.toml missing author substitution")
	}
	if !strings.Contains(content, `title = "Alice's Brags"`) {
		t.Error("hugo.toml missing title substitution")
	}
	if strings.Contains(content, "%%"+"AUTHOR%%") {
		t.Error("hugo.toml still contains author placeholder")
	}
	if strings.Contains(content, "%%"+"TITLE%%") {
		t.Error("hugo.toml still contains title placeholder")
	}
}

func TestHugoDeployAboutMd(t *testing.T) {
	dir := t.TempDir()
	e := NewHugoEngine(&config.Config{})

	dst := filepath.Join(dir, "content", "about.md")
	if err := e.deployTemplate("templates/hugo/about.md", dst, "Alice", "Alice's Brags"); err != nil {
		t.Fatalf("deployTemplate about.md: %v", err)
	}

	data, err := os.ReadFile(dst) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("read about.md: %v", err)
	}

	content := string(data)

	// Should have author substituted
	if !strings.Contains(content, "About Alice") {
		t.Error("about.md missing author substitution")
	}

	// Should have a real date, not a placeholder
	if strings.Contains(content, "%%"+"DATE%%") {
		t.Error("about.md still contains date placeholder")
	}
	if strings.Contains(content, "%%"+"YEAR%%") {
		t.Error("about.md still contains year placeholder")
	}

	// Must NOT contain Hugo template syntax (it's a content file, not an archetype)
	if strings.Contains(content, "{{") {
		t.Errorf("about.md contains Hugo template syntax:\n%s",
			firstLineContaining(content, "{{"))
	}

	// Should have a valid YAML date in frontmatter
	if !strings.Contains(content, "date: 20") {
		t.Error("about.md missing valid date in frontmatter")
	}
}

func TestHugoCopyEmbedded(t *testing.T) {
	dir := t.TempDir()
	e := NewHugoEngine(&config.Config{})

	dst := filepath.Join(dir, "archetypes", "week.md")
	if err := e.copyEmbedded("templates/hugo/archetypes/week.md", dst); err != nil {
		t.Fatalf("copyEmbedded: %v", err)
	}

	data, err := os.ReadFile(dst) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}

	// Should contain Hugo template syntax verbatim
	if !strings.Contains(string(data), "{{ .Date }}") {
		t.Error("archetype missing Hugo template syntax {{ .Date }}")
	}
}

func TestHugoValidate(t *testing.T) {
	dir := t.TempDir()
	e := NewHugoEngine(&config.Config{})

	// Should fail — no _config.yaml
	if err := e.Validate(dir); err == nil {
		t.Error("Validate should fail on empty dir")
	}

	// Write _config.yaml with engine: hugo
	cfg := &Config{Title: "Test", Author: "Alice", Engine: "hugo"}
	if err := writeConfig(dir, cfg); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}

	// Should still fail — no hugo.toml
	if err := e.Validate(dir); err == nil {
		t.Error("Validate should fail without hugo.toml")
	}

	// Create hugo.toml
	if err := os.WriteFile(filepath.Join(dir, "hugo.toml"), []byte("title = 'test'"), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	// Should pass now
	if err := e.Validate(dir); err != nil {
		t.Errorf("Validate should pass: %v", err)
	}
}

func TestHugoValidateWrongEngine(t *testing.T) {
	dir := t.TempDir()
	e := NewHugoEngine(&config.Config{})

	cfg := &Config{Title: "Test", Author: "Alice", Engine: "markdown"}
	if err := writeConfig(dir, cfg); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hugo.toml"), []byte("title = 'test'"), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	if err := e.Validate(dir); err == nil {
		t.Error("Validate should fail for wrong engine")
	}
}

func TestResolveHugoCommand(t *testing.T) {
	// With explicit config, should return that
	cfg := &config.Config{}
	cfg.Hugo.Command = "/usr/local/bin/my-hugo"
	cmd, err := cfg.ResolveHugoCommand()
	if err != nil {
		t.Fatalf("ResolveHugoCommand: %v", err)
	}
	if cmd != "/usr/local/bin/my-hugo" {
		t.Errorf("got %q, want /usr/local/bin/my-hugo", cmd)
	}

	// Without config, should find hugo-bragctl or hugo in PATH (if available)
	cfg = &config.Config{}
	resolved, err := cfg.ResolveHugoCommand()
	_, hasBragctl := exec.LookPath("hugo-bragctl")
	_, hasHugo := exec.LookPath("hugo")
	if hasBragctl == nil || hasHugo == nil {
		if err != nil {
			t.Errorf("unexpected error with hugo in PATH: %v", err)
		}
		if resolved == "" {
			t.Error("resolved command is empty")
		}
	} else if err == nil {
		t.Error("expected error when neither hugo nor hugo-bragctl in PATH")
	}
}

// hugoTestSite creates a minimal Hugo site with our archetypes deployed.
// Returns the site path and the resolved hugo command.
func hugoTestSite(t *testing.T) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	cfg := &config.Config{}
	hugoCmd, err := cfg.ResolveHugoCommand()
	if err != nil {
		t.Skip("hugo not available")
	}

	dir := t.TempDir()
	cmd := exec.Command(hugoCmd, "new", "site", dir, "--force") //nolint:gosec // test
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hugo new site: %v\n%s", err, out)
	}

	// Deploy our archetypes
	e := NewHugoEngine(cfg)
	for _, name := range []string{"week.md", "month.md", "year.md"} {
		src := "templates/hugo/archetypes/" + name
		dst := filepath.Join(dir, "archetypes", name)
		if err := e.copyEmbedded(src, dst); err != nil {
			t.Fatalf("deploy archetype %s: %v", name, err)
		}
	}

	return dir, hugoCmd
}

func TestArchetypeWeek(t *testing.T) {
	sitePath, hugoCmd := hugoTestSite(t)

	cmd := exec.Command(hugoCmd, "new", "-k", "week", "content/posts/week-10-03-26.md") //nolint:gosec // test
	cmd.Dir = sitePath
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hugo new week: %v\n%s", err, out)
	}

	data, err := os.ReadFile(filepath.Join(sitePath, "content", "posts", "week-10-03-26.md")) //nolint:gosec // test
	if err != nil {
		t.Fatalf("read week file: %v", err)
	}

	content := string(data)

	// Frontmatter should have resolved values (not raw template syntax)
	if strings.Contains(content, "{{ .Date }}") {
		t.Error("week archetype has unresolved {{ .Date }}")
	}
	if strings.Contains(content, "{{ .Name }}") {
		t.Error("week archetype has unresolved {{ .Name }}")
	}

	// Should contain the resolved title with the file name
	if !strings.Contains(content, "week-10-03-26") {
		t.Error("week archetype missing resolved .Name in title")
	}

	// Should have a real date in frontmatter
	if !strings.Contains(content, "date: ") {
		t.Error("week archetype missing date field")
	}

	// Should have dateFormat-resolved tags (year and month name)
	if !strings.Contains(content, "accomplishments") {
		t.Error("week archetype missing expected tag")
	}
}

func TestArchetypeMonth(t *testing.T) {
	sitePath, hugoCmd := hugoTestSite(t)

	cmd := exec.Command(hugoCmd, "new", "-k", "month", "content/2026/March/_index.md") //nolint:gosec // test
	cmd.Dir = sitePath
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hugo new month: %v\n%s", err, out)
	}

	data, err := os.ReadFile(filepath.Join(sitePath, "content", "2026", "March", "_index.md")) //nolint:gosec // test
	if err != nil {
		t.Fatalf("read month file: %v", err)
	}

	content := string(data)

	// No unresolved template syntax should remain
	if strings.Contains(content, "{{") {
		t.Errorf("month archetype has unresolved template syntax:\n%s",
			firstLineContaining(content, "{{"))
	}

	// Should have resolved month name and year
	for _, want := range []string{
		`month: "`,         // month field populated
		`year: "`,          // year field populated
		"Monthly Overview", // title structure
		"Monthly Highlights",
		"Previous Month",
		"Next Month",
		"Year Overview",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("month archetype missing %q", want)
		}
	}
}

func TestArchetypeYear(t *testing.T) {
	sitePath, hugoCmd := hugoTestSite(t)

	cmd := exec.Command(hugoCmd, "new", "-k", "year", "content/2026/_index.md") //nolint:gosec // test
	cmd.Dir = sitePath
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hugo new year: %v\n%s", err, out)
	}

	data, err := os.ReadFile(filepath.Join(sitePath, "content", "2026", "_index.md")) //nolint:gosec // test
	if err != nil {
		t.Fatalf("read year file: %v", err)
	}

	content := string(data)

	// No unresolved template syntax should remain
	if strings.Contains(content, "{{") {
		t.Errorf("year archetype has unresolved template syntax:\n%s",
			firstLineContaining(content, "{{"))
	}

	// Should have resolved year values
	for _, want := range []string{
		`year: "`,              // year field populated
		"Annual Brag Document", // title structure
		"Professional Growth",
		"Previous Year",
		"Next Year",
		"Current Month",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("year archetype missing %q", want)
		}
	}
}

// firstLineContaining returns the first line containing substr, for error messages.
func firstLineContaining(s, substr string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, substr) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func TestHugoInit(t *testing.T) {
	// Skip if hugo or git not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	cfg := &config.Config{}
	if _, err := cfg.ResolveHugoCommand(); err != nil {
		t.Skip("hugo not available")
	}

	dir := t.TempDir()
	sitePath := filepath.Join(dir, "test-hugo-site")
	e := NewHugoEngine(cfg)

	err := e.Init(context.Background(), InitOpts{
		Name:   "test-hugo-site",
		Path:   sitePath,
		Title:  "Test Brags",
		Author: "Alice",
		Engine: "hugo",
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Check _config.yaml
	siteCfg, err := loadConfig(sitePath)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if siteCfg.Engine != "hugo" {
		t.Errorf("engine = %q, want hugo", siteCfg.Engine)
	}
	if siteCfg.Author != "Alice" {
		t.Errorf("author = %q, want Alice", siteCfg.Author)
	}

	// Check hugo.toml exists with substituted values
	hugoToml, err := os.ReadFile(filepath.Join(sitePath, "hugo.toml")) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("read hugo.toml: %v", err)
	}
	if !strings.Contains(string(hugoToml), `author = "Alice"`) {
		t.Error("hugo.toml missing author")
	}
	if !strings.Contains(string(hugoToml), `title = "Test Brags"`) {
		t.Error("hugo.toml missing title")
	}

	// Check content/posts/ exists
	if _, err := os.Stat(filepath.Join(sitePath, "content", "posts")); err != nil {
		t.Errorf("content/posts/ missing: %v", err)
	}

	// Check archetypes
	for _, name := range []string{"week.md", "month.md", "year.md"} {
		if _, err := os.Stat(filepath.Join(sitePath, "archetypes", name)); err != nil {
			t.Errorf("archetype %s missing: %v", name, err)
		}
	}

	// Check about.md deployed with substitution
	aboutData, err := os.ReadFile(filepath.Join(sitePath, "content", "about.md")) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("read about.md: %v", err)
	}
	if !strings.Contains(string(aboutData), "About Alice") {
		t.Error("about.md missing author substitution")
	}

	// Check .gitignore
	if _, err := os.Stat(filepath.Join(sitePath, ".gitignore")); err != nil {
		t.Errorf(".gitignore missing: %v", err)
	}

	// Check theme submodule
	if _, err := os.Stat(filepath.Join(sitePath, "themes", "hugo-book")); err != nil {
		t.Errorf("themes/hugo-book missing: %v", err)
	}

	// Check git repo with initial commit
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = sitePath
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if !strings.Contains(string(out), "Initial bragctl site") {
		t.Errorf("unexpected commit message: %s", out)
	}

	// Validate should pass
	if err := e.Validate(sitePath); err != nil {
		t.Errorf("Validate after Init: %v", err)
	}
}
