package root

import "github.com/spf13/cobra"

func setupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up integrations and credentials",
		Long:  `Configure credentials for third-party services like Google, Microsoft, etc.`,
	}

	cmd.AddCommand(setupGoogleCmd())

	return cmd
}
