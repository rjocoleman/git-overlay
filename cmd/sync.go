package cmd

import (
	"fmt"

	"github.com/rjocoleman/git-overlay/internal/git"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Update upstream code and rebuild links",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Open repository and sync upstream
		repo, err := git.InitMainRepository()
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}

		if err := repo.SyncUpstream(cfg.Upstream.Ref); err != nil {
			return fmt.Errorf("failed to sync upstream: %w", err)
		}

		// Update gitignore and rebuild links
		if err := updateGitignore(cfg, nil); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}

		if err := createLinks(cmd, cfg); err != nil {
			return fmt.Errorf("failed to rebuild links: %w", err)
		}

		fmt.Println("Git overlay repository synchronized successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
