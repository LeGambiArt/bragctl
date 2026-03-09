// Package root defines the bragctl root command and global flags.
package root

import (
	"context"
	"fmt"
	"os"

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
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return config.EnsureDirs()
		},
	}

	rootCmd.AddCommand(versionCmd(version, buildDate))
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(aiCmd())
	rootCmd.AddCommand(claudeCmd())
	rootCmd.AddCommand(cursorCmd())
	rootCmd.AddCommand(geminiCmd())
	rootCmd.AddCommand(mcpSetupCmd())
	rootCmd.AddCommand(configCmd())

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
	var engine, title, author string

	cmd := &cobra.Command{
		Use:   "init <site-name>",
		Short: "Create a new brag document site",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			if engine == "" {
				engine = "markdown"
			}
			if title == "" {
				title = "My Brag Document"
			}
			if author == "" {
				author = currentUser()
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			mgr := site.NewManager(cfg)

			s, err := mgr.Create(context.Background(), site.InitOpts{
				Name:   name,
				Title:  title,
				Author: author,
				Engine: engine,
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

	cmd.Flags().StringVarP(&engine, "engine", "e", "", "Site engine: markdown (default: markdown)")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Site title")
	cmd.Flags().StringVarP(&author, "author", "a", "", "Author name")

	_ = cmd.RegisterFlagCompletionFunc("engine", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"markdown"}, cobra.ShellCompDirectiveNoFileComp
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

func currentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "Unknown"
}
