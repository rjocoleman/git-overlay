package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rjocoleman/git-overlay/internal/config"
	"github.com/spf13/cobra"
)

func TestCreateLinks(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "git-overlay-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create test structure
	if err := os.MkdirAll(".upstream/src/lib", 0755); err != nil {
		t.Fatalf("Failed to create upstream directory: %v", err)
	}
	if err := os.MkdirAll("overlay", 0755); err != nil {
		t.Fatalf("Failed to create overlay directory: %v", err)
	}

	// Create test files
	if err := os.WriteFile(".upstream/src/lib/test.txt", []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(".upstream/.test-file", []byte("dotfile content"), 0644); err != nil {
		t.Fatalf("Failed to create dotfile: %v", err)
	}

	tests := []struct {
		name      string
		cfg       *config.Config
		linkMode  string
		force     bool
		wantError bool
	}{
		{
			name: "simple symlink",
			cfg: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "src/lib/test.txt"},
				},
			},
			linkMode:  "symlink",
			wantError: false,
		},
		{
			name: "hardlink file",
			cfg: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "src/lib/test.txt"},
				},
			},
			linkMode:  "hardlink",
			wantError: false,
		},
		{
			name: "copy file",
			cfg: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "src/lib/test.txt"},
				},
			},
			linkMode:  "copy",
			wantError: false,
		},
		{
			name: "custom target path",
			cfg: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{
						From: "src/lib/test.txt",
						To:   "custom/path/test.txt",
					},
				},
			},
			linkMode:  "symlink",
			wantError: false,
		},
		{
			name: "invalid link mode",
			cfg: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "src/lib/test.txt"},
				},
			},
			linkMode:  "invalid",
			wantError: true,
		},
		{
			name: "dotfile symlink",
			cfg: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: ".test-file"},
				},
			},
			linkMode:  "symlink",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean overlay directory and recreate it empty
			if err := os.RemoveAll("overlay"); err != nil {
				t.Fatalf("Failed to remove overlay directory: %v", err)
			}
			if err := os.MkdirAll("overlay", 0755); err != nil {
				t.Fatalf("Failed to recreate overlay directory: %v", err)
			}

			// Create test command
			cmd := &cobra.Command{}
			cmd.Flags().String("link-mode", tt.linkMode, "")
			cmd.Flags().Bool("force", true, "") // Always use force in tests to handle existing files

			err := CreateLinks(cmd, tt.cfg)
			if (err != nil) != tt.wantError {
				t.Errorf("CreateLinks() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				return
			}

			// Verify link/copy was created
			for _, link := range tt.cfg.Symlinks {
				var targetPath string
				if link.String != "" {
					targetPath = filepath.Join("overlay", link.String)
				} else {
					targetPath = filepath.Join("overlay", link.To)
				}

				if _, err := os.Stat(targetPath); err != nil {
					t.Errorf("Target file not created: %v", err)
				}

				// For copy mode, verify content
				if tt.linkMode == "copy" {
					content, err := os.ReadFile(targetPath)
					if err != nil {
						t.Errorf("Failed to read target file: %v", err)
					}
					expectedContent := "test content"
					if link.String == ".test-file" {
						expectedContent = "dotfile content"
					}
					if string(content) != expectedContent {
						t.Errorf("Copy content mismatch, got %q, want %q", string(content), expectedContent)
					}
				}

				// For symlink mode, verify link target
				if tt.linkMode == "symlink" {
					target, err := os.Readlink(targetPath)
					if err != nil {
						t.Errorf("Failed to read symlink target: %v", err)
					}
					// For custom target paths, we need to calculate the relative path differently
					var expectedTarget string
					if link.String != "" {
						// For simple paths, calculate relative path from target to source
						sourcePath := filepath.Join(".upstream", link.String)
						targetDir := filepath.Dir(targetPath)
						expectedTarget, err = filepath.Rel(targetDir, sourcePath)
						if err != nil {
							t.Errorf("Failed to calculate relative path: %v", err)
						}
					} else {
						// For custom paths, calculate relative path from target to source
						sourcePath := filepath.Join(".upstream", link.From)
						targetDir := filepath.Dir(targetPath)
						expectedTarget, err = filepath.Rel(targetDir, sourcePath)
						if err != nil {
							t.Errorf("Failed to calculate relative path: %v", err)
						}
					}

					if target != expectedTarget {
						t.Errorf("Symlink target mismatch, got %q, want %q", target, expectedTarget)
					}
				}
			}
		})
	}
}
