package root

import (
	"fmt"

	"github.com/spf13/cobra"

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

			fmt.Printf("MCP command:    %s\n", cfg.MCPCommand())
			fmt.Printf("MCP workdir:    %s\n", cfg.MCPWorkdir())
			if len(cfg.MCP.Args) > 0 {
				fmt.Printf("MCP extra args: %v\n", cfg.MCP.Args)
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
