// Package root defines the bragctl root command and global flags.
package root

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/LeGambiArt/bragctl/internal/ai"
	"github.com/LeGambiArt/bragctl/internal/config"
	"github.com/LeGambiArt/bragctl/internal/mcp"
	"github.com/LeGambiArt/bragctl/internal/site"
	"github.com/LeGambiArt/bragctl/internal/ui"
)

// New creates the root cobra command with all subcommands.
func New(version, buildDate string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bragctl",
		Short: "Manage brag document sites",
		Long: `bragctl is a CLI tool for managing brag document sites.
It supports Hugo and plain Markdown engines, and integrates
with AI assistants (Claude, Cursor, Gemini) via MCP.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(versionCmd(version, buildDate))
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(aiCmd())
	rootCmd.AddCommand(claudeCmd())
	rootCmd.AddCommand(cursorCmd())
	rootCmd.AddCommand(geminiCmd())
	rootCmd.AddCommand(mcpSetupCmd())
	rootCmd.AddCommand(contextCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(newCmd())
	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(setupCmd())

	return rootCmd
}

func versionCmd(version, buildDate string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("bragctl %s (built %s)\n", version, buildDate)
		},
	}
}

func initCmd() *cobra.Command {
	var engine, title, author, aiPref string
	var force bool

	cmd := &cobra.Command{
		Use:   "init <site-name>",
		Short: "Create a new brag document site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Check if site already exists BEFORE prompting
			sitePath := filepath.Join(config.SitesDir(), name)
			if _, err := os.Stat(sitePath); err == nil && !force {
				return fmt.Errorf("site %q already exists at %s (use --force to re-initialize)", name, sitePath)
			}

			// Prompt for engine if not provided
			if engine == "" && ui.IsTerminal() && !cmd.Flags().Changed("engine") {
				engine = ui.PromptSelect("Site engine", []string{"markdown", "hugo"}, "markdown")
			} else if engine == "" {
				engine = "markdown"
			}

			if title == "" {
				title = "My Brag Document"
			}

			// Prompt for author if not provided and stdin is a terminal
			if author == "" {
				dflt := currentUser()
				if ui.IsTerminal() && !cmd.Flags().Changed("author") {
					author = ui.PromptInput("Author name", dflt)
				} else {
					author = dflt
				}
			}

			// Prompt for AI preference if not provided
			if aiPref == "" && ui.IsTerminal() && !cmd.Flags().Changed("ai") {
				aiPref = ui.PromptSelect("AI assistant", []string{"auto", "claude", "cursor", "gemini"}, "auto")
				if aiPref == "auto" {
					aiPref = ""
				}
			}

			// Remove existing site if --force was used
			if force {
				if err := os.RemoveAll(sitePath); err != nil {
					return fmt.Errorf("remove existing site: %w", err)
				}
			}

			mgr := site.NewManager(cfg)

			s, err := mgr.Create(context.Background(), site.InitOpts{
				Name:   name,
				Title:  title,
				Author: author,
				Engine: engine,
				AI:     aiPref,
			})
			if err != nil {
				return err
			}

			ui.Success("Created site %q at %s", s.Name, s.Path)
			ui.KeyValue("Engine:", s.Engine.Name())

			// Set as default if it's the first site or no default is set
			if cfg.DefaultSite == "" {
				cfg.DefaultSite = name
				if err := config.Save(cfg); err != nil {
					return fmt.Errorf("save config: %w", err)
				}
				ui.Info("Set as default site")
			}

			// Auto MCP setup
			mcpAssistant := aiPref
			if mcpAssistant == "auto" || mcpAssistant == "" {
				if detected, err := ai.Detect(); err == nil {
					mcpAssistant = detected.Name
				} else {
					mcpAssistant = ""
				}
			}
			if mcpAssistant != "" {
				if err := mcp.Setup(mcpAssistant, s.Path, cfg.MCPCommand(), cfg.MCPArgs()); err != nil {
					ui.Dim("MCP setup skipped: %v", err)
					ui.Dim("  Run 'bragctl mcp-setup %s' later to configure", name)
				} else {
					ui.Success("MCP configured for %s", mcpAssistant)
				}
			}

			fmt.Println()
			ui.Dim("To start writing:")
			ui.Dim("  bragctl ai %s", s.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&engine, "engine", "e", "", "Site engine: markdown, hugo (default: markdown)")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Site title")
	cmd.Flags().StringVarP(&author, "author", "a", "", "Author name")
	cmd.Flags().StringVar(&aiPref, "ai", "", "Preferred AI assistant (claude, cursor, gemini)")
	cmd.Flags().BoolVar(&force, "force", false, "Re-initialize existing site")

	_ = cmd.RegisterFlagCompletionFunc("engine", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"markdown", "hugo"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all managed sites",
		Aliases: []string{"ls"},
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			mgr := site.NewManager(cfg)

			sites, err := mgr.List()
			if err != nil {
				return err
			}

			if len(sites) == 0 {
				ui.Dim("No sites found. Create one with: bragctl init <name>")
				return nil
			}

			rows := make([]ui.SiteRow, len(sites))
			for i, s := range sites {
				status := "stopped"
				port := "-"
				if state := site.ReadServerState(s.Path); state.IsRunning() {
					status = "running"
					port = fmt.Sprintf("%d", state.Port)
				}

				rows[i] = ui.SiteRow{
					Name:      s.Name,
					Engine:    s.Config.Engine,
					AI:        s.Config.AI,
					Status:    status,
					Port:      port,
					IsDefault: s.Name == cfg.DefaultSite,
				}
			}
			ui.PrintSiteTable(rows)
			return nil
		},
	}
}

func newCmd() *cobra.Command {
	var kind string

	cmd := &cobra.Command{
		Use:   "new [site-name] [title]",
		Short: "Create a new brag entry",
		Long: `Create a new brag entry for a site.

For Hugo sites, creates entries using archetypes:
  bragctl new              # current bi-weekly entry (default)
  bragctl new --kind month # monthly overview
  bragctl new --kind year  # yearly overview

For markdown sites, creates dated posts:
  bragctl new              # current bi-weekly entry
  bragctl new "my topic"   # freeform post`,
		Args:              cobra.MaximumNArgs(2),
		ValidArgsFunction: completeSiteNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Parse args: [site-name] [title]
			var siteName, title string
			switch len(args) {
			case 2:
				siteName = args[0]
				title = args[1]
			case 1:
				// Could be site name or title — try as site first
				mgr := site.NewManager(cfg)
				if _, err := mgr.Resolve(args[0]); err == nil {
					siteName = args[0]
				} else {
					title = args[0]
				}
			}

			s, err := resolveSite(cfg, []string{siteName})
			if err != nil {
				return err
			}

			k := kind
			if k == "" && title != "" {
				k = "post"
			}

			path, err := s.Engine.New(cmd.Context(), s.Path, site.NewOpts{
				Kind:  k,
				Title: title,
			})
			if err != nil {
				return err
			}

			ui.Success("Entry: %s", path)
			return nil
		},
	}

	cmd.Flags().StringVarP(&kind, "kind", "k", "", "Entry kind: week (default), month, year, post")
	_ = cmd.RegisterFlagCompletionFunc("kind", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"week", "month", "year", "post"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func serveCmd() *cobra.Command {
	var port int
	var bind string
	var foreground bool

	cmd := &cobra.Command{
		Use:               "serve [site-name]",
		Short:             "Start a dev server to preview a site",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeSiteNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			s, err := resolveSite(cfg, args)
			if err != nil {
				return err
			}

			opts := site.ServeOpts{
				Port:       port,
				Bind:       bind,
				Foreground: foreground,
			}

			if foreground {
				return s.Engine.Serve(cmd.Context(), s.Path, opts)
			}
			return site.StartBackground(s.Name, s.Path, opts)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to serve on (default: auto-detect from 1313)")
	cmd.Flags().StringVar(&bind, "bind", "127.0.0.1", "Address to bind to (use 0.0.0.0 for all interfaces)")
	cmd.Flags().BoolVarP(&foreground, "foreground", "f", false, "Run in foreground (blocking)")
	return cmd
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "stop [site-name]",
		Short:             "Stop a running dev server",
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
			return site.StopServer(s.Name, s.Path)
		},
	}
}

func currentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "Unknown"
}
