// Package mcp handles MCP configuration generation for AI assistants.
package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the MCP server configuration for an AI assistant.
type Config struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// ServerConfig defines a single MCP server.
type ServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// Setup writes MCP configuration for an AI assistant at a site directory.
// The mcpBinary is the path to the what-the-mcp binary.
// The workdir is the what-the-mcp workdir for --workdir flag.
func Setup(assistant, sitePath, mcpBinary, workdir string) error {
	if mcpBinary == "" {
		// Try to find it in PATH
		mcpBinary = "what-the-mcp"
	}

	cfg := Config{
		MCPServers: map[string]ServerConfig{
			"what-the-mcp": {
				Command: mcpBinary,
				Args:    []string{"--workdir", workdir},
			},
		},
	}

	switch assistant {
	case "claude":
		return writeJSON(filepath.Join(sitePath, ".mcp.json"), cfg)
	case "cursor":
		dir := filepath.Join(sitePath, ".cursor")
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
		return writeJSON(filepath.Join(dir, "mcp.json"), cfg)
	case "gemini":
		return writeGeminiConfig(sitePath, cfg)
	default:
		return fmt.Errorf("unknown assistant for MCP setup: %s", assistant)
	}
}

// Remove removes MCP configuration for an assistant from a site.
func Remove(assistant, sitePath string) error {
	switch assistant {
	case "claude":
		return os.Remove(filepath.Join(sitePath, ".mcp.json"))
	case "cursor":
		return os.Remove(filepath.Join(sitePath, ".cursor", "mcp.json"))
	case "gemini":
		return os.Remove(filepath.Join(sitePath, ".gemini", "settings.json"))
	default:
		return fmt.Errorf("unknown assistant: %s", assistant)
	}
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644) //nolint:gosec // config file
}

func writeGeminiConfig(sitePath string, cfg Config) error {
	dir := filepath.Join(sitePath, ".gemini")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	// Gemini uses a different format
	geminiCfg := map[string]any{
		"mcpServers": cfg.MCPServers,
	}
	return writeJSON(filepath.Join(dir, "settings.json"), geminiCfg)
}
