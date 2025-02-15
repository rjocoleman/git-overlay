package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// State represents the git-overlay state
type State struct {
	ManagedFiles []ManagedFile `json:"managed_files"`
}

// ManagedFile represents a file managed by git-overlay
type ManagedFile struct {
	Path     string `json:"path"`     // Path relative to overlay directory
	LinkMode string `json:"linkMode"` // Link mode used (symlink, hardlink, copy)
	Source   string `json:"source"`   // Source path in .upstream
}

// LoadState loads the state file
func LoadState() (*State, error) {
	data, err := os.ReadFile(".git-overlay.state.json")
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// SaveState saves the state file
func (s *State) SaveState() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(".git-overlay.state.json", data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// AddManagedFile adds a file to the managed files list
func (s *State) AddManagedFile(path, linkMode, source string) {
	// Remove any existing entry for this path
	for i := len(s.ManagedFiles) - 1; i >= 0; i-- {
		if s.ManagedFiles[i].Path == path {
			s.ManagedFiles = append(s.ManagedFiles[:i], s.ManagedFiles[i+1:]...)
		}
	}

	s.ManagedFiles = append(s.ManagedFiles, ManagedFile{
		Path:     path,
		LinkMode: linkMode,
		Source:   source,
	})
}

// RemoveManagedFile removes a file from the managed files list
func (s *State) RemoveManagedFile(path string) {
	for i := len(s.ManagedFiles) - 1; i >= 0; i-- {
		if s.ManagedFiles[i].Path == path {
			s.ManagedFiles = append(s.ManagedFiles[:i], s.ManagedFiles[i+1:]...)
		}
	}
}

// IsManagedFile checks if a file is managed by git-overlay
func (s *State) IsManagedFile(path string) (bool, *ManagedFile) {
	for _, f := range s.ManagedFiles {
		if f.Path == path {
			return true, &f
		}
	}
	return false, nil
}

// GetManagedFilesInDir returns all managed files in a directory
func (s *State) GetManagedFilesInDir(dir string) []ManagedFile {
	var files []ManagedFile
	for _, f := range s.ManagedFiles {
		if filepath.Dir(f.Path) == dir || f.Path == dir {
			files = append(files, f)
		}
	}
	return files
}
