package root

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/bragctl/bragctl/internal/config"
)

func contextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage site context files",
		Long:  `List, enable, disable, or edit context.d/ files for a site.`,
	}

	cmd.AddCommand(contextListCmd())
	cmd.AddCommand(contextEnableCmd())
	cmd.AddCommand(contextDisableCmd())
	cmd.AddCommand(contextEditCmd())

	return cmd
}

func contextListCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "list [site]",
		Short:             "List context files with enabled/disabled status",
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

			ctxDir := filepath.Join(s.Path, "context.d")
			entries, err := os.ReadDir(ctxDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No context.d/ directory found.")
					return nil
				}
				return err
			}

			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				if strings.HasSuffix(name, ".md") {
					fmt.Printf("  %-30s enabled\n", strings.TrimSuffix(name, ".md"))
				} else if strings.HasSuffix(name, ".md.disabled") {
					fmt.Printf("  %-30s disabled\n", strings.TrimSuffix(name, ".md.disabled"))
				}
			}
			return nil
		},
	}
}

func contextEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "enable <name> [site]",
		Short:             "Enable a context file",
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: completeSiteNames,
		RunE: func(_ *cobra.Command, args []string) error {
			return toggleContext(args, true)
		},
	}
}

func contextDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "disable <name> [site]",
		Short:             "Disable a context file",
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: completeSiteNames,
		RunE: func(_ *cobra.Command, args []string) error {
			return toggleContext(args, false)
		},
	}
}

func contextEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "edit <name> [site]",
		Short:             "Edit a context file in $EDITOR",
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: completeSiteNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			name := args[0]
			siteArgs := args[1:]
			s, err := resolveSite(cfg, siteArgs)
			if err != nil {
				return err
			}

			// Find the file (enabled or disabled)
			ctxDir := filepath.Join(s.Path, "context.d")
			filePath := filepath.Join(ctxDir, name+".md")
			if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
				filePath = filepath.Join(ctxDir, name+".md.disabled")
				if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
					return fmt.Errorf("context %q not found in %s", name, ctxDir)
				}
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			cmd := exec.Command(editor, filePath) //nolint:gosec // editor from env
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}
}

func toggleContext(args []string, enable bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	name := args[0]
	siteArgs := args[1:]
	s, err := resolveSite(cfg, siteArgs)
	if err != nil {
		return err
	}

	ctxDir := filepath.Join(s.Path, "context.d")

	if enable {
		src := filepath.Join(ctxDir, name+".md.disabled")
		dst := filepath.Join(ctxDir, name+".md")
		if _, statErr := os.Stat(src); os.IsNotExist(statErr) {
			if _, statErr := os.Stat(dst); statErr == nil {
				fmt.Printf("%s is already enabled\n", name)
				return nil
			}
			return fmt.Errorf("context %q not found", name)
		}
		if err := os.Rename(src, dst); err != nil {
			return err
		}
		fmt.Printf("Enabled %s\n", name)
	} else {
		src := filepath.Join(ctxDir, name+".md")
		dst := filepath.Join(ctxDir, name+".md.disabled")
		if _, statErr := os.Stat(src); os.IsNotExist(statErr) {
			if _, statErr := os.Stat(dst); statErr == nil {
				fmt.Printf("%s is already disabled\n", name)
				return nil
			}
			return fmt.Errorf("context %q not found", name)
		}
		if err := os.Rename(src, dst); err != nil {
			return err
		}
		fmt.Printf("Disabled %s\n", name)
	}
	return nil
}
