// Package ai handles launching AI assistants with proper context
// and MCP configuration for brag document sites.
package ai

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/ai-spec.md
var aiSpecFS embed.FS

//go:embed templates/context.d/*.md
var contextTemplatesFS embed.FS

// ContextTemplate describes a context.d template with its default state.
type ContextTemplate struct {
	Name    string // e.g. "persona"
	File    string // e.g. "persona.md"
	Enabled bool   // default enabled/disabled
}

// DefaultContextTemplates returns the list of context templates
// shipped with bragctl and their default enabled state.
func DefaultContextTemplates() []ContextTemplate {
	return []ContextTemplate{
		{Name: "persona", File: "persona.md", Enabled: true},
		{Name: "brag-rules", File: "brag-rules.md", Enabled: true},
		{Name: "startup", File: "startup.md", Enabled: true},
		{Name: "shutdown", File: "shutdown.md", Enabled: true},
		{Name: "notes", File: "notes.md", Enabled: true},
		{Name: "adhd", File: "adhd.md", Enabled: true},
	}
}

// contextData is the template rendering context.
type contextData struct {
	Author string
	Engine string
	Title  string
}

// validateTemplateValue validates that a value is safe for text/template rendering.
// It rejects values containing Go template delimiters to prevent template injection.
func validateTemplateValue(value, fieldName string) error {
	if strings.Contains(value, "{{") || strings.Contains(value, "}}") {
		return fmt.Errorf("%s cannot contain template delimiters ({{ or }}): %q", fieldName, value)
	}
	return nil
}

// RenderContextTemplates renders all context.d templates into the site's
// context.d/ directory. Disabled templates get .md.disabled extension.
// Existing files are NOT overwritten — only missing files are created.
func RenderContextTemplates(sitePath, author, engine, title string) error {
	// Validate template inputs to prevent template injection
	if err := validateTemplateValue(author, "author"); err != nil {
		return err
	}
	if err := validateTemplateValue(engine, "engine"); err != nil {
		return err
	}
	if err := validateTemplateValue(title, "title"); err != nil {
		return err
	}

	ctxDir := filepath.Join(sitePath, "context.d")
	if err := os.MkdirAll(ctxDir, 0o750); err != nil {
		return fmt.Errorf("create context.d: %w", err)
	}

	data := contextData{Author: author, Engine: engine, Title: title}

	for _, ct := range DefaultContextTemplates() {
		outName := ct.File
		if !ct.Enabled {
			outName += ".disabled"
		}
		outPath := filepath.Join(ctxDir, outName)

		// Also check the opposite state — if user toggled it, don't recreate
		altName := ct.File
		if ct.Enabled {
			altName += ".disabled"
		}
		altPath := filepath.Join(ctxDir, altName)

		// Skip if either version exists
		if fileExists(outPath) || fileExists(altPath) {
			continue
		}

		tmplContent, err := contextTemplatesFS.ReadFile("templates/context.d/" + ct.File)
		if err != nil {
			return fmt.Errorf("read template %s: %w", ct.File, err)
		}

		tmpl, err := template.New(ct.Name).Parse(string(tmplContent))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", ct.File, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("render template %s: %w", ct.File, err)
		}

		if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil { //nolint:gosec // user content
			return fmt.Errorf("write %s: %w", outName, err)
		}
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Assistant represents a supported AI coding assistant.
type Assistant struct {
	Name        string
	Command     string
	ContextFile string // CLAUDE.md, GEMINI.md, .cursorrules
}

// Known assistants.
var (
	Claude = Assistant{Name: "claude", Command: "claude", ContextFile: "CLAUDE.md"}
	Cursor = Assistant{Name: "cursor", Command: "cursor", ContextFile: ".cursorrules"}
	Gemini = Assistant{Name: "gemini", Command: "gemini", ContextFile: "GEMINI.md"}
)

// GreetArgs returns the CLI arguments to send an initial "." prompt
// that triggers the persona greeting on session start.
func (a Assistant) GreetArgs() []string {
	switch a.Name {
	case "claude":
		return []string{"."}
	case "gemini":
		return []string{"--prompt-interactive", "."}
	default:
		return []string{"."}
	}
}

// AllAssistants returns the list of supported assistants.
func AllAssistants() []Assistant {
	return []Assistant{Claude, Cursor, Gemini}
}

// ByName returns an assistant by name, or an error if not found.
func ByName(name string) (Assistant, error) {
	switch strings.ToLower(name) {
	case "claude":
		return Claude, nil
	case "cursor":
		return Cursor, nil
	case "gemini":
		return Gemini, nil
	default:
		return Assistant{}, fmt.Errorf("unknown assistant: %s (supported: claude, cursor, gemini)", name)
	}
}

// Detect returns the first installed assistant, or an error if none found.
func Detect() (Assistant, error) {
	for _, a := range AllAssistants() {
		if _, err := exec.LookPath(a.Command); err == nil {
			return a, nil
		}
	}
	return Assistant{}, fmt.Errorf("no AI assistant found in PATH (tried: claude, cursor, gemini)")
}

// Launch starts an AI assistant pointed at a site directory.
// Extra args are passed through to the assistant command.
func Launch(assistant Assistant, sitePath string, extraArgs ...string) error {
	path, err := exec.LookPath(assistant.Command)
	if err != nil {
		return fmt.Errorf("%s not found in PATH", assistant.Command)
	}

	cmd := exec.Command(path, extraArgs...) //nolint:gosec // assistant command from LookPath
	cmd.Dir = sitePath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// aiSpecFile is the rendered template that AI-specific files symlink to.
const aiSpecFile = "ai-spec.md"

// WriteContext renders ai-spec.md from the embedded template and creates
// symlinks for all AI assistants (CLAUDE.md, GEMINI.md, .cursorrules).
func WriteContext(_ Assistant, sitePath, _ /* siteName */, engineName, author string) error {
	content, err := renderAISpec(engineName, author)
	if err != nil {
		return fmt.Errorf("render ai-spec: %w", err)
	}

	// Append enabled context.d/*.md files
	contextContent, err := loadEnabledContext(sitePath)
	if err != nil {
		return fmt.Errorf("load context: %w", err)
	}
	if contextContent != "" {
		content += "\n" + contextContent
	}

	specPath := filepath.Join(sitePath, aiSpecFile)
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil { //nolint:gosec // generated context
		return fmt.Errorf("write %s: %w", aiSpecFile, err)
	}

	// Create symlinks for all assistants
	for _, a := range AllAssistants() {
		if err := ensureSymlink(sitePath, a.ContextFile, aiSpecFile); err != nil {
			return fmt.Errorf("symlink %s: %w", a.ContextFile, err)
		}
	}
	return nil
}

// loadEnabledContext reads all enabled .md files from context.d/ and
// concatenates them with section headers.
func loadEnabledContext(sitePath string) (string, error) {
	ctxDir := filepath.Join(sitePath, "context.d")
	entries, err := os.ReadDir(ctxDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	var buf bytes.Buffer
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue // skip dirs and disabled files (.md.disabled)
		}
		data, err := os.ReadFile(filepath.Join(ctxDir, e.Name())) //nolint:gosec // dir entries from known config path
		if err != nil {
			return "", fmt.Errorf("read %s: %w", e.Name(), err)
		}
		buf.Write(data)
		buf.WriteString("\n")
	}
	return buf.String(), nil
}

// ensureSymlink creates a relative symlink from name → target in dir.
// Removes any existing file or broken symlink at name first.
func ensureSymlink(dir, name, target string) error {
	linkPath := filepath.Join(dir, name)

	// Remove existing file/symlink if present
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("remove existing %s: %w", name, err)
		}
	}

	return os.Symlink(target, linkPath)
}

type aiSpecData struct {
	Engine string
	Author string
}

func renderAISpec(engineName, author string) (string, error) {
	// Validate template inputs to prevent template injection
	if err := validateTemplateValue(engineName, "engine"); err != nil {
		return "", err
	}
	if err := validateTemplateValue(author, "author"); err != nil {
		return "", err
	}

	tmplContent, err := aiSpecFS.ReadFile("templates/ai-spec.md")
	if err != nil {
		return "", fmt.Errorf("read embedded template: %w", err)
	}

	tmpl, err := template.New("ai-spec").Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, aiSpecData{Engine: engineName, Author: author}); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// AssistantNames returns the names of all supported assistants (for completions).
func AssistantNames() []string {
	assistants := AllAssistants()
	names := make([]string, len(assistants))
	for i, a := range assistants {
		names[i] = a.Name
	}
	return names
}
