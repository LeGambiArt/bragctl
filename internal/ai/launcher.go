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

// WriteContext generates the AI context file for a site.
func WriteContext(assistant Assistant, sitePath, siteName, engineName string) error {
	content := generateContext(siteName, engineName)
	path := filepath.Join(sitePath, assistant.ContextFile)
	return os.WriteFile(path, []byte(content), 0o644) //nolint:gosec // user content file
}

func generateContext(siteName, engineName string) string {
	var b strings.Builder

	b.WriteString("# Brag Document Assistant\n\n")
	b.WriteString(fmt.Sprintf("Site: %s\n", siteName))
	b.WriteString(fmt.Sprintf("Engine: %s\n\n", engineName))

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

	// Check for custom instructions
	b.WriteString("\n## Custom Instructions\n\n")
	b.WriteString("Add an `INSTRUCTIONS.md` file to this site for custom rules.\n")

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
