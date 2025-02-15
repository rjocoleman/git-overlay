package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rjocoleman/git-overlay/internal/config"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove managed files and links",
	Long: `Remove files and links managed by git-overlay in the overlay directory.
This only removes files that are configured in .git-overlay.yml.
Custom files and directories are preserved.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if overlay directory exists
		if _, err := os.Stat("overlay"); os.IsNotExist(err) {
			return fmt.Errorf("overlay directory does not exist")
		}

		// Load state
		state, err := config.LoadState()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		// Create lookup map of managed paths
		managedPaths := make(map[string]struct{})
		for _, mf := range state.ManagedFiles {
			managedPaths[mf.Path] = struct{}{}
		}

		// Sort managed paths by depth (deepest first)
		var sortedPaths []string
		for path := range managedPaths {
			sortedPaths = append(sortedPaths, path)
		}
		sort.Slice(sortedPaths, func(i, j int) bool {
			iDepth := strings.Count(sortedPaths[i], "/")
			jDepth := strings.Count(sortedPaths[j], "/")
			if iDepth == jDepth {
				return sortedPaths[i] > sortedPaths[j] // alphabetical fallback
			}
			return iDepth > jDepth
		})

		removed := 0

		// Process each managed path
		for _, relPath := range sortedPaths {
			fullPath := filepath.Join("overlay", relPath)

			// Check if path exists
			info, err := os.Lstat(fullPath)
			if os.IsNotExist(err) {
				state.RemoveManagedFile(relPath)
				continue
			}

			// Handle files and symlinks
			if !info.IsDir() {
				if err := os.Remove(fullPath); err == nil {
					removed++
				}
				state.RemoveManagedFile(relPath)
				continue
			}

			// Handle directories
			if isFullyManaged(fullPath, managedPaths) {
				if err := os.RemoveAll(fullPath); err == nil {
					removed++
				}
				state.RemoveManagedFile(relPath)
			}
		}

		// Final cleanup: ensure all managed paths are removed from state
		for path := range managedPaths {
			state.RemoveManagedFile(path)
		}

		// Clean up any empty directories
		if err := removeEmptyDirs("overlay"); err != nil {
			return fmt.Errorf("failed to clean up empty directories: %w", err)
		}

		// Save state and print results
		if err := state.SaveState(); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}
		fmt.Printf("Removed %d managed files and directories\n", removed)
		return nil
	},
}

// isFullyManaged checks if a directory and all its contents are managed
func isFullyManaged(path string, managedPaths map[string]struct{}) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		relPath, err := filepath.Rel("overlay", entryPath)
		if err != nil {
			return false
		}

		// Check if this entry is managed
		if _, ok := managedPaths[relPath]; !ok {
			return false
		}

		// Recursively check directories
		if entry.IsDir() {
			if !isFullyManaged(entryPath, managedPaths) {
				return false
			}
		}
	}
	return true
}

// removeEmptyDirs recursively traverses the directory tree starting at 'dir'.
// After processing children, it checks if the directory is empty and removes it.
func removeEmptyDirs(dir string) error {
	// List directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory %q: %w", dir, err)
	}

	// Process subdirectories recursively
	for _, entry := range entries {
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())
			if err := removeEmptyDirs(subdir); err != nil {
				return err
			}
		}
	}

	// Re-read directory entries after possibly removing subdirectories
	entries, err = os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("re-reading directory %q: %w", dir, err)
	}

	// If the directory is empty and not the root overlay directory, remove it
	if len(entries) == 0 && dir != "overlay" {
		if err := os.Remove(dir); err != nil {
			return fmt.Errorf("removing directory %q: %w", dir, err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
