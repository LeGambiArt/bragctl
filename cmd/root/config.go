package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LeGambiArt/bragctl/internal/ai"
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
	cmd.AddCommand(configSetDefaultSiteCmd())
	cmd.AddCommand(configClearDefaultSiteCmd())
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

			dflt := cfg.DefaultSite
			if dflt == "" {
				dflt = "(not set)"
			}

			// Build full MCP command line
			mcpFull := cfg.MCPCommand()
			for _, arg := range cfg.MCPArgs() {
				mcpFull += " " + arg
			}

			rows := [][]string{
				{"Config", config.Path()},
				{"Base dir", config.BaseDir()},
				{"Sites", config.SitesDir()},
				{"Default site", dflt},
				{"MCP", mcpFull},
			}
			ui.PrintKeyValueTable(rows)

			return nil
		},
	}
}

func configSetDefaultSiteCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "set-default-site <site-name>",
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

func configClearDefaultSiteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear-default-site",
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

func configSetAICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-ai <assistant> [site]",
		Short: "Set the preferred AI assistant for a site",
		Long: `Set which AI assistant bragctl ai launches for a site.
Supported: claude, cursor, gemini, auto (detect from PATH).`,
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: completeAIAndSite,
		RunE: func(_ *cobra.Command, args []string) error {
			assistant := args[0]

			// Validate assistant name
			valid := ai.AssistantNames()
			if assistant != "auto" {
				found := false
				for _, v := range valid {
					if v == assistant {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("unknown assistant %q (supported: %v, auto)", assistant, valid)
				}
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			siteArgs := args[1:]
			s, err := resolveSite(cfg, siteArgs)
			if err != nil {
				return err
			}

			siteCfg, err := site.LoadConfig(s.Path)
			if err != nil {
				return fmt.Errorf("load site config: %w", err)
			}

			if assistant == "auto" {
				siteCfg.AI = ""
			} else {
				siteCfg.AI = assistant
			}

			if err := site.SaveConfig(s.Path, siteCfg); err != nil {
				return fmt.Errorf("save site config: %w", err)
			}

			if assistant == "auto" {
				ui.Success("AI assistant set to auto-detect for %s", s.Name)
			} else {
				ui.Success("AI assistant set to %s for %s", assistant, s.Name)
			}
			return nil
		},
	}
}

func completeAIAndSite(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		names := append(ai.AssistantNames(), "auto")
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 1 {
		cfg, err := config.Load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		mgr := site.NewManager(cfg)
		siteNames, _ := mgr.ListNames()
		return siteNames, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
