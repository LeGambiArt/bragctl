// Package root defines the bragctl root command and global flags.
package root

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/bragctl/bragctl/internal/config"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/site"
)

// New creates the root cobra command with all subcommands.
func New(version, buildDate string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bragctl",
		Short: "Manage brag document sites",
		Long: `bragctl is a CLI tool for managing brag document sites.
It supports Hugo and plain Markdown engines, and integrates
with AI assistants (Claude, Cursor, Gemini) via MCP.`,
		SilenceUsage: true,
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
	rootCmd.AddCommand(serveCmd())

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

			if engine == "" {
				engine = "markdown"
			}
			if title == "" {
				title = "My Brag Document"
			}

			// Prompt for author if not provided and stdin is a terminal
			if author == "" {
				dflt := currentUser()
				if isTerminal() && !cmd.Flags().Changed("author") {
					author = prompt(fmt.Sprintf("Author name [%s]: ", dflt), dflt)
				} else {
					author = dflt
				}
			}

			// Prompt for AI preference if not provided
			if aiPref == "" && isTerminal() && !cmd.Flags().Changed("ai") {
				aiPref = prompt("AI assistant (claude/cursor/gemini) [auto]: ", "")
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

			fmt.Printf("Created site %q at %s\n", s.Name, s.Path)
			fmt.Printf("Engine: %s\n", s.Engine.Name())

			// Set as default if it's the first site or no default is set
			if cfg.DefaultSite == "" {
				cfg.DefaultSite = name
				if err := config.Save(cfg); err != nil {
					return fmt.Errorf("save config: %w", err)
				}
				fmt.Printf("Set as default site\n")
			}

			fmt.Println()
			fmt.Println("To start writing:")
			fmt.Printf("  bragctl ai %s\n", s.Name)
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
				fmt.Println("No sites found. Create one with: bragctl init <name>")
				return nil
			}

			for _, s := range sites {
				marker := " "
				if s.Name == cfg.DefaultSite {
					marker = "*"
				}
				fmt.Printf(" %s %-20s %-10s %s\n", marker, s.Name, s.Config.Engine, s.Path)
			}
			return nil
		},
	}
}

func serveCmd() *cobra.Command {
	var port int

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
			return s.Engine.Serve(cmd.Context(), s.Path, site.ServeOpts{
				Port: port,
			})
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 1313, "Port to serve on")
	return cmd
}

func currentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "Unknown"
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func prompt(question, defaultVal string) string {
	fmt.Print(question)
	var input string
	_, _ = fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}
