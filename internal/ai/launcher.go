// Package ai handles launching AI assistants with proper context
// and MCP configuration for brag document sites.
package ai

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
func Launch(assistant Assistant, sitePath string) error {
	path, err := exec.LookPath(assistant.Command)
	if err != nil {
		return fmt.Errorf("%s not found in PATH", assistant.Command)
	}

	cmd := exec.Command(path) //nolint:gosec // assistant command from LookPath
	cmd.Dir = sitePath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// autogenFile is the single source context file that AI-specific
// files symlink to.
const autogenFile = "autogen-context.md"

// WriteContext generates autogen-context.md and creates symlinks
// for all AI assistants (CLAUDE.md, GEMINI.md, .cursorrules).
func WriteContext(_ Assistant, sitePath, siteName, engineName string) error {
	content := generateContext(siteName, engineName)
	autogenPath := filepath.Join(sitePath, autogenFile)
	if err := os.WriteFile(autogenPath, []byte(content), 0o644); err != nil { //nolint:gosec // generated context
		return fmt.Errorf("write %s: %w", autogenFile, err)
	}

	// Create symlinks for all assistants
	for _, a := range AllAssistants() {
		if err := ensureSymlink(sitePath, a.ContextFile, autogenFile); err != nil {
			return fmt.Errorf("symlink %s: %w", a.ContextFile, err)
		}
	}
	return nil
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

func generateContext(siteName, engineName string) string {
	var b strings.Builder

	b.WriteString("# Brag Document Assistant\n\n")
	b.WriteString(fmt.Sprintf("Site: %s | Engine: %s\n\n", siteName, engineName))

	b.WriteString("## Site Structure\n\n")
	switch engineName {
	case "hugo":
		b.WriteString("Posts are in `content/posts/`.\n")
		b.WriteString("Use Hugo frontmatter format.\n")
		b.WriteString("Run `hugo server -D` to preview.\n")
	case "markdown":
		b.WriteString("Posts are in `posts/`.\n")
		b.WriteString("Use YAML frontmatter: title, date, tags, impact.\n")
		b.WriteString("Files are plain markdown — no build step needed.\n")
	}

	b.WriteString("\n## Writing Brag Entries\n\n")
	b.WriteString("Each post should capture a professional accomplishment:\n")
	b.WriteString("- What you did (the action)\n")
	b.WriteString("- Why it matters (the impact)\n")
	b.WriteString("- Who was involved (collaboration)\n")
	b.WriteString("- Quantify when possible (metrics, numbers)\n")

	b.WriteString("\n## Additional Context\n\n")
	b.WriteString("Read all files in `context.d/` for workflow preferences, team\n")
	b.WriteString("conventions, and custom instructions specific to this site.\n")

	b.WriteString("\n## MCP Tools\n\n")
	b.WriteString("This project has MCP tools available via the what-the-mcp server.\n")
	b.WriteString("Before using a plugin's tools for the first time, read its MCP\n")
	b.WriteString("resource for tool-specific guidelines and usage patterns.\n")

	return b.String()
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
