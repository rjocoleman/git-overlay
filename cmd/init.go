package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rjocoleman/git-overlay/internal/config"
	"github.com/rjocoleman/git-overlay/internal/git"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new overlay repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Remove existing .upstream directory if it exists
		if err := os.RemoveAll(".upstream"); err != nil {
			return fmt.Errorf("failed to remove existing .upstream directory: %w", err)
		}

		// Create overlay directory
		if err := os.MkdirAll("overlay", 0755); err != nil {
			return fmt.Errorf("failed to create overlay directory: %w", err)
		}

		// Initialize Git repository and add upstream submodule
		repo, err := git.InitMainRepository()
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		if err := repo.AddUpstreamSubmodule(cfg.Upstream.URL); err != nil {
			return fmt.Errorf("failed to add upstream submodule: %w", err)
		}

		// Sync to the specified ref
		if err := repo.SyncUpstream(cfg.Upstream.Ref); err != nil {
			return fmt.Errorf("failed to sync upstream: %w", err)
		}

		// Create initial links
		if err := CreateLinks(cmd, cfg); err != nil {
			return fmt.Errorf("failed to create links: %w", err)
		}

		fmt.Println("Git overlay repository initialized successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func updateGitignore(cfg *config.Config, createdLinks []string) error {
	// Create initial gitignore content
	content := "# BEGIN GIT-OVERLAY MANAGED BLOCK - DO NOT EDIT\n"

	// Add each created link to gitignore
	for _, link := range createdLinks {
		content += link + "\n"
	}

	content += "# END GIT-OVERLAY MANAGED BLOCK"

	// Check if .gitignore exists
	if _, err := os.Stat(".gitignore"); os.IsNotExist(err) {
		return os.WriteFile(".gitignore", []byte(content), 0644)
	}

	// Read existing .gitignore
	existing, err := os.ReadFile(".gitignore")
	if err != nil {
		return err
	}

	// Remove old managed block if it exists
	lines := strings.Split(string(existing), "\n")
	var newLines []string
	inManagedBlock := false
	for _, line := range lines {
		if line == "# BEGIN GIT-OVERLAY MANAGED BLOCK - DO NOT EDIT" {
			inManagedBlock = true
			continue
		}
		if line == "# END GIT-OVERLAY MANAGED BLOCK" {
			inManagedBlock = false
			continue
		}
		if !inManagedBlock {
			newLines = append(newLines, line)
		}
	}

	// Add new managed block
	if len(newLines) > 0 && newLines[len(newLines)-1] != "" {
		newLines = append(newLines, "")
	}
	newLines = append(newLines, content)

	// Write back to file
	return os.WriteFile(".gitignore", []byte(strings.Join(newLines, "\n")), 0644)
}
