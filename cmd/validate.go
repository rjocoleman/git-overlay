package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validatePath ensures a path does not escape its parent directory
func validatePath(base, path string) error {
	// Check if path is absolute
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths are not allowed: %s", path)
	}

	// Clean paths to normalize them
	cleanBase := filepath.Clean(base)
	cleanPath := filepath.Clean(filepath.Join(base, path))

	// Ensure the path starts with the base directory
	rel, err := filepath.Rel(cleanBase, cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if the path tries to escape using ../
	if strings.HasPrefix(rel, "..") || strings.Contains(path, "../") {
		return fmt.Errorf("path attempts to escape base directory: %s", path)
	}

	return nil
}
