package cmd

import (
	"github.com/spf13/cobra"
)

var (
	version string // Set by SetVersion

	rootCmd = &cobra.Command{
		Use:     "git-overlay",
		Short:   "Git Overlay - Manage overlay repositories that extend upstream Git repositories",
		Version: version,
	}
)

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version string for the root command
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", ".git-overlay.yml", "Path to config file")
	rootCmd.PersistentFlags().BoolP("force", "f", false, "Force overwrite of existing files/links")
	rootCmd.PersistentFlags().String("link-mode", "symlink", "Link mode (symlink|hardlink|copy)")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
}
