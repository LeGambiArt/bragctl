package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/bragctl/bragctl/internal/ai"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/config"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/mcp"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/site"
)

func aiCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "ai [site-name]",
		Short:             "Launch the default AI assistant for a site",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeSiteNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			s, err := resolveSite(cfg, args)
			if err != nil {
				return err
			}

			assistant, err := resolveAssistant(s)
			if err != nil {
				return err
			}

			return launchForSite(cfg, assistant, s)
		},
	}
}

func claudeCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "claude [site-name]",
		Short:             "Launch Claude Code for a site",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeSiteNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			s, err := resolveSite(cfg, args)
			if err != nil {
				return err
			}
			return launchForSite(cfg, ai.Claude, s)
		},
	}
}

func cursorCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "cursor [site-name]",
		Short:             "Launch Cursor for a site",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeSiteNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			s, err := resolveSite(cfg, args)
			if err != nil {
				return err
			}
			return launchForSite(cfg, ai.Cursor, s)
		},
	}
}

func geminiCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "gemini [site-name]",
		Short:             "Launch Gemini CLI for a site",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeSiteNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			s, err := resolveSite(cfg, args)
			if err != nil {
				return err
			}
			return launchForSite(cfg, ai.Gemini, s)
		},
	}
}

func mcpSetupCmd() *cobra.Command {
	var assistant string

	cmd := &cobra.Command{
		Use:               "mcp-setup [site-name]",
		Short:             "Configure MCP for an AI assistant",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeSiteNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			s, err := resolveSite(cfg, args)
			if err != nil {
				return err
			}

			assistants := ai.AssistantNames()
			if assistant != "" {
				assistants = []string{assistant}
			}

			for _, name := range assistants {
				if err := setupMCP(cfg, name, s.Path); err != nil {
					return fmt.Errorf("mcp setup for %s: %w", name, err)
				}
				fmt.Printf("MCP configured for %s at %s\n", name, s.Path)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&assistant, "assistant", "", "Configure for specific assistant (claude, cursor, gemini)")
	_ = cmd.RegisterFlagCompletionFunc("assistant", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return ai.AssistantNames(), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func launchForSite(cfg *config.Config, assistant ai.Assistant, s *site.Site) error {
	if err := ai.WriteContext(assistant, s.Path, s.Name, s.Engine.Name()); err != nil {
		return fmt.Errorf("write context: %w", err)
	}

	if err := setupMCP(cfg, assistant.Name, s.Path); err != nil {
		return fmt.Errorf("mcp setup: %w", err)
	}

	fmt.Printf("Launching %s for site %q...\n", assistant.Name, s.Name)
	return ai.Launch(assistant, s.Path)
}

func setupMCP(cfg *config.Config, assistant, sitePath string) error {
	return mcp.Setup(assistant, sitePath, cfg.MCPCommand(), cfg.MCPArgs())
}

// resolveAssistant picks the AI assistant: site preference, then auto-detect.
func resolveAssistant(s *site.Site) (ai.Assistant, error) {
	if s.Config.AI != "" {
		return ai.ByName(s.Config.AI)
	}
	return ai.Detect()
}

func resolveSite(cfg *config.Config, args []string) (*site.Site, error) {
	mgr := site.NewManager(cfg)
	return mgr.Resolve(siteName(args))
}

func completeSiteNames(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	mgr := site.NewManager(cfg)
	names, _ := mgr.ListNames()
	return names, cobra.ShellCompDirectiveNoFileComp
}

func siteName(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}
