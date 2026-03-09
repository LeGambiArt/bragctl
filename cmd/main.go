// bragctl is a CLI tool for managing brag document sites.
package main

import (
	"os"

	"gitlab.cee.redhat.com/bragctl/bragctl/cmd/root"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/config"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/ui"
)

// Version and BuildDate are set via ldflags at build time.
var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	if err := config.EnsureDirs(); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
	rootCmd := root.New(Version, BuildDate)
	if err := rootCmd.Execute(); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
}
