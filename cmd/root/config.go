package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/bragctl/bragctl/internal/ai"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/config"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/site"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage bragctl configuration",
	}

	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configSetDefaultCmd())
	cmd.AddCommand(configClearDefaultCmd())
	cmd.AddCommand(configSetAICmd())

	return cmd
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			fmt.Printf("Config:    %s\n", config.Path())
			fmt.Printf("Base dir:  %s\n", config.BaseDir())
			fmt.Printf("Sites:     %s\n", config.SitesDir())
			fmt.Println()

			if cfg.DefaultSite != "" {
				fmt.Printf("Default site:   %s\n", cfg.DefaultSite)
			} else {
				fmt.Printf("Default site:   (not set)\n")
			}

			if cfg.DefaultAI != "" {
				fmt.Printf("Default AI:     %s\n", cfg.DefaultAI)
			} else {
				fmt.Printf("Default AI:     (auto-detect)\n")
			}

			fmt.Printf("Default engine: %s\n", cfg.DefaultEngine)

			if cfg.MCP.Server != "" {
				fmt.Printf("MCP server:     %s\n", cfg.MCP.Server)
			} else {
				fmt.Printf("MCP server:     what-the-mcp (from PATH)\n")
			}

			return nil
		},
	}
}

func configSetDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "set-default <site-name>",
		Short:             "Set the default site",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeSiteNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Verify site exists
			mgr := site.NewManager(cfg)
			if _, err := mgr.Resolve(args[0]); err != nil {
				return fmt.Errorf("site %q not found", args[0])
			}

			cfg.DefaultSite = args[0]
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Default site set to %q\n", args[0])
			return nil
		},
	}
}

func configClearDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear-default",
		Short: "Clear the default site",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.DefaultSite = ""
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println("Default site cleared")
			return nil
		},
	}
}

func configSetAICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-ai <assistant>",
		Short: "Set the default AI assistant",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return ai.AssistantNames(), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if _, err := ai.ByName(args[0]); err != nil {
				return err
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.DefaultAI = args[0]
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Default AI set to %q\n", args[0])
			return nil
		},
	}
}
