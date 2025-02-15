package main

import (
	"fmt"
	"os"

	"github.com/rjocoleman/git-overlay/cmd"
)

var (
	// Version information (set by goreleaser)
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
