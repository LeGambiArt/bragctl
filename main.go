// bragctl is a CLI tool for managing brag document sites.
package main

import "fmt"

// Version and BuildDate are set via ldflags at build time.
var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	fmt.Printf("bragctl %s (built %s)\n", Version, BuildDate)
}
