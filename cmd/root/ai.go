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

			assistant, err := resolveAssistant(cfg)
			if err != nil {
				return err
			}

			return launchForSite(cfg, assistant, siteName(args))
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
			return launchForSite(cfg, ai.Claude, siteName(args))
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
			return launchForSite(cfg, ai.Cursor, siteName(args))
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
			return launchForSite(cfg, ai.Gemini, siteName(args))
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
			mgr := site.NewManager(cfg)

			s, err := mgr.Resolve(siteName(args))
			if err != nil {
				return err
			}

			// Determine which assistants to set up
			assistants := ai.AssistantNames()
			if assistant != "" {
				assistants = []string{assistant}
			}

			mcpBinary := cfg.MCP.Server
			workdir := config.BaseDir()

			for _, name := range assistants {
				if err := mcp.Setup(name, s.Path, mcpBinary, workdir); err != nil {
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

func launchForSite(cfg *config.Config, assistant ai.Assistant, name string) error {
	mgr := site.NewManager(cfg)

	s, err := mgr.Resolve(name)
	if err != nil {
		return err
	}

	// Write/update context file
	if err := ai.WriteContext(assistant, s.Path, s.Name, s.Engine.Name()); err != nil {
		return fmt.Errorf("write context: %w", err)
	}

	// Set up MCP config — what-the-mcp uses bragctl's base dir as workdir
	mcpBinary := cfg.MCP.Server
	workdir := config.BaseDir()
	if err := mcp.Setup(assistant.Name, s.Path, mcpBinary, workdir); err != nil {
		return fmt.Errorf("mcp setup: %w", err)
	}

	fmt.Printf("Launching %s for site %q...\n", assistant.Name, s.Name)
	return ai.Launch(assistant, s.Path)
}

func resolveAssistant(cfg *config.Config) (ai.Assistant, error) {
	if cfg.DefaultAI != "" {
		return ai.ByName(cfg.DefaultAI)
	}
	return ai.Detect()
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
