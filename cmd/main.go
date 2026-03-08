// bragctl is a CLI tool for managing brag document sites.
package main

import (
	"fmt"
	"os"

	"gitlab.cee.redhat.com/bragctl/bragctl/cmd/root"
)

// Version and BuildDate are set via ldflags at build time.
var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	rootCmd := root.New(Version, BuildDate)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
