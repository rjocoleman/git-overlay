package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rjocoleman/git-overlay/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// getGitCommandEnv returns a properly configured environment for git commands
func getGitCommandEnv(name, email string) []string {
	return append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		fmt.Sprintf("GIT_AUTHOR_NAME=%s", name),
		fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", email),
		fmt.Sprintf("GIT_COMMITTER_NAME=%s", name),
		fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", email),
	)
}

// runGitCommand executes a git command with the proper environment
func runGitCommand(dir string, args []string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = getGitCommandEnv("test", "test@example.com")
	return cmd.Run()
}

// loadConfig loads and validates the configuration file
func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if cfg.Upstream.URL == "" {
		return nil, fmt.Errorf("upstream.url is required")
	}
	if cfg.Upstream.Ref == "" {
		return nil, fmt.Errorf("upstream.ref is required")
	}

	return &cfg, nil
}

// createLink creates a single link (symlink, hardlink, or copy) from src to dst
func createLink(src, dst string, linkMode string, force bool, createdLinks *[]string, state *config.State) error {
	// Validate paths
	if err := validatePath("overlay", strings.TrimPrefix(dst, "overlay/")); err != nil {
		return fmt.Errorf("invalid target path: %w", err)
	}

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(dst)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", dst, err)
	}

	// Handle existing target
	if _, err := os.Stat(dst); err == nil {
		if !force {
			return fmt.Errorf("target already exists: %s", dst)
		}
		// Remove existing file or link
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("failed to remove existing target %s: %w", dst, err)
		}
	}

	// Special handling for .gitignore
	if strings.HasSuffix(dst, ".gitignore") {
		fmt.Println("Note: .gitignore is being copied for compatibility")
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy .gitignore: %w", err)
		}
		// Track created link and state
		*createdLinks = append(*createdLinks, dst)
		relPath := strings.TrimPrefix(dst, "overlay/")
		relSrc := strings.TrimPrefix(src, ".upstream/")
		state.AddManagedFile(relPath, "copy", relSrc)
		return nil
	}

	switch linkMode {
	case "symlink":
		// For symlinks, we need to use relative paths
		relPath, err := filepath.Rel(filepath.Dir(dst), src)
		if err != nil {
			return fmt.Errorf("failed to create relative path from %s to %s: %w", src, dst, err)
		}
		if err := os.Symlink(relPath, dst); err != nil {
			return fmt.Errorf("failed to create symlink from %s to %s: %w", src, dst, err)
		}
	case "hardlink":
		if err := os.Link(src, dst); err != nil {
			return fmt.Errorf("failed to create hardlink from %s to %s: %w", src, dst, err)
		}
	case "copy":
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy from %s to %s: %w", src, dst, err)
		}
	default:
		return fmt.Errorf("unsupported link mode: %s", linkMode)
	}

	// Track created link for gitignore and state
	*createdLinks = append(*createdLinks, dst)

	// Track in state
	relPath := strings.TrimPrefix(dst, "overlay/")
	relSrc := strings.TrimPrefix(src, ".upstream/")
	state.AddManagedFile(relPath, linkMode, relSrc)

	return nil
}

// CreateLinks creates symlinks according to the configuration
func CreateLinks(cmd *cobra.Command, cfg *config.Config) error {
	linkMode, err := cmd.Flags().GetString("link-mode")
	if err != nil {
		return err
	}

	// Override link mode from config if set
	if cfg.LinkMode != "" {
		linkMode = cfg.LinkMode
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	// Load state
	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Track all created symlinks for gitignore
	var createdLinks []string

	for _, link := range cfg.Symlinks {
		var pattern, targetBase string
		if link.String != "" {
			pattern = link.String
			targetBase = link.String
		} else {
			pattern = link.From
			targetBase = link.To
		}

		// Calculate source and target paths
		from := filepath.Join(".upstream", pattern)
		to := filepath.Join("overlay", targetBase)

		// Check if source exists
		info, err := os.Stat(from)
		if err != nil {
			return fmt.Errorf("source does not exist: %s", from)
		}

		// Handle directories
		if info.IsDir() {
			// Walk the directory and create links for each file
			err := filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Skip directories themselves
				if info.IsDir() {
					return nil
				}

				// Calculate relative path from source base
				relPath, err := filepath.Rel(from, path)
				if err != nil {
					return fmt.Errorf("failed to get relative path: %w", err)
				}

				// Calculate target path preserving directory structure
				targetPath := filepath.Join("overlay", targetBase, relPath)

				return createLink(path, targetPath, linkMode, force, &createdLinks, state)
			})
			if err != nil {
				return fmt.Errorf("failed to process directory %s: %w", pattern, err)
			}
		} else {
			// Handle single file
			if err := createLink(from, to, linkMode, force, &createdLinks, state); err != nil {
				return fmt.Errorf("failed to process file %s: %w", pattern, err)
			}
		}
	}

	// Update gitignore with all created links
	if err := updateGitignore(cfg, createdLinks); err != nil {
		return fmt.Errorf("failed to update gitignore: %w", err)
	}

	// Save state
	if err := state.SaveState(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// copyPath copies a file or directory from src to dst
func copyPath(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir recursively copies a directory from src to dst
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
