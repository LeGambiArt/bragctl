package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LeGambiArt/bragctl/internal/config"
	"github.com/LeGambiArt/bragctl/internal/site"
	"github.com/LeGambiArt/bragctl/internal/ui"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage bragctl configuration",
	}

	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configSetDefaultCmd())
	cmd.AddCommand(configClearDefaultCmd())

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

			dflt := cfg.DefaultSite
			if dflt == "" {
				dflt = "(not set)"
			}

			rows := [][]string{
				{"Config", config.Path()},
				{"Base dir", config.BaseDir()},
				{"Sites", config.SitesDir()},
				{"Default site", dflt},
				{"MCP command", cfg.MCPCommand()},
				{"MCP workdir", cfg.MCPWorkdir()},
			}
			if len(cfg.MCP.Args) > 0 {
				rows = append(rows, []string{"MCP extra args", fmt.Sprintf("%v", cfg.MCP.Args)})
			}
			ui.PrintKeyValueTable(rows)

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

			mgr := site.NewManager(cfg)
			if _, err := mgr.Resolve(args[0]); err != nil {
				return fmt.Errorf("site %q not found", args[0])
			}

			cfg.DefaultSite = args[0]
			if err := config.Save(cfg); err != nil {
				return err
			}
			ui.Success("Default site set to %q", args[0])
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
			ui.Success("Default site cleared")
			return nil
		},
	}
}
