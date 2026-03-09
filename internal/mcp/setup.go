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
// command is the what-the-mcp binary path, args are the full argument list
// (including --workdir and any extras).
func Setup(assistant, sitePath, command string, args []string) error {
	cfg := Config{
		MCPServers: map[string]ServerConfig{
			"what-the-mcp": {
				Command: command,
				Args:    args,
			},
		},
	}

	switch assistant {
	case "claude":
		if err := writeJSON(filepath.Join(sitePath, ".mcp.json"), cfg); err != nil {
			return err
		}
		return writeClaudeSettings(sitePath)
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
		_ = os.Remove(filepath.Join(sitePath, ".claude", "settings.local.json"))
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

func writeClaudeSettings(sitePath string) error {
	dir := filepath.Join(sitePath, ".claude")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	settings := map[string]any{
		"enabledMcpjsonServers":      []string{"what-the-mcp"},
		"enableAllProjectMcpServers": true,
	}
	return writeJSON(filepath.Join(dir, "settings.local.json"), settings)
}

func writeGeminiConfig(sitePath string, cfg Config) error {
	dir := filepath.Join(sitePath, ".gemini")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	geminiCfg := map[string]any{
		"mcpServers": cfg.MCPServers,
	}
	return writeJSON(filepath.Join(dir, "settings.json"), geminiCfg)
}
