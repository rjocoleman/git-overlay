package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rjocoleman/git-overlay/internal/config"
	"github.com/spf13/cobra"
)

func TestCleanCommand(t *testing.T) {
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

	type testSetup struct {
		managedFiles []config.ManagedFile
		setupFunc    func(t *testing.T)
	}

	tests := []struct {
		name            string
		config          *config.Config
		setup           testSetup
		wantError       bool
		verifyPreserved func(t *testing.T)
	}{
		{
			name: "clean empty overlay directory",
			config: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "test.txt"},
				},
			},
			setup: testSetup{
				managedFiles: []config.ManagedFile{},
				setupFunc: func(t *testing.T) {
					if err := os.Mkdir("overlay", 0755); err != nil {
						t.Fatalf("Failed to create overlay directory: %v", err)
					}
				},
			},
			wantError: false,
		},
		{
			name: "clean managed symlinks only",
			config: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "managed.txt"},
				},
			},
			setup: testSetup{
				managedFiles: []config.ManagedFile{
					{Path: "managed.txt", LinkMode: "symlink", Source: "managed.txt"},
				},
				setupFunc: func(t *testing.T) {
					// Create overlay directory
					if err := os.MkdirAll("overlay", 0755); err != nil {
						t.Fatalf("Failed to create overlay directory: %v", err)
					}

					// Create source files
					if err := os.WriteFile(".upstream/managed.txt", []byte("managed"), 0644); err != nil {
						t.Fatalf("Failed to create managed file: %v", err)
					}

					// Create custom file
					if err := os.WriteFile("overlay/custom.txt", []byte("custom"), 0644); err != nil {
						t.Fatalf("Failed to create custom file: %v", err)
					}

					// Create managed symlink
					if err := os.Symlink(filepath.Join("..", ".upstream", "managed.txt"), filepath.Join("overlay", "managed.txt")); err != nil {
						t.Fatalf("Failed to create managed symlink: %v", err)
					}
				},
			},
			wantError: false,
			verifyPreserved: func(t *testing.T) {
				// Verify custom file still exists
				if _, err := os.Stat("overlay/custom.txt"); os.IsNotExist(err) {
					t.Error("Custom file was removed")
				}

				// Verify managed symlink was removed
				if _, err := os.Stat("overlay/managed.txt"); !os.IsNotExist(err) {
					t.Error("Managed symlink was not removed")
				}
			},
		},
		{
			name: "clean managed directory with nested structure",
			config: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "dir"},
				},
			},
			setup: testSetup{
				managedFiles: []config.ManagedFile{
					{Path: "dir/managed.txt", LinkMode: "symlink", Source: "dir/managed.txt"},
					{Path: "dir/empty", LinkMode: "symlink", Source: "dir/empty"},
				},
				setupFunc: func(t *testing.T) {
					// Create directories
					if err := os.MkdirAll("overlay/dir/empty", 0755); err != nil {
						t.Fatalf("Failed to create empty directory: %v", err)
					}
					if err := os.MkdirAll("overlay/dir/keep", 0755); err != nil {
						t.Fatalf("Failed to create directory to keep: %v", err)
					}
					if err := os.MkdirAll(".upstream/dir", 0755); err != nil {
						t.Fatalf("Failed to create upstream directory: %v", err)
					}

					// Create managed symlink in directory
					if err := os.WriteFile(".upstream/dir/managed.txt", []byte("managed"), 0644); err != nil {
						t.Fatalf("Failed to create managed file: %v", err)
					}
					if err := os.Symlink(filepath.Join("..", "..", ".upstream", "dir", "managed.txt"), filepath.Join("overlay", "dir", "managed.txt")); err != nil {
						t.Fatalf("Failed to create managed symlink: %v", err)
					}

					// Create custom file in keep directory
					if err := os.WriteFile("overlay/dir/keep/custom.txt", []byte("custom"), 0644); err != nil {
						t.Fatalf("Failed to create custom file: %v", err)
					}
				},
			},
			wantError: false,
			verifyPreserved: func(t *testing.T) {
				// Verify custom file still exists
				if _, err := os.Stat("overlay/dir/keep/custom.txt"); os.IsNotExist(err) {
					t.Error("Custom file in directory was removed")
				}

				// Verify managed symlink was removed
				if _, err := os.Stat("overlay/dir/managed.txt"); !os.IsNotExist(err) {
					t.Error("Managed symlink was not removed")
				}

				// Verify empty directory was removed
				if _, err := os.Stat("overlay/dir/empty"); !os.IsNotExist(err) {
					t.Error("Empty directory was not removed")
				}

				// Verify directory with content was preserved
				if _, err := os.Stat("overlay/dir/keep"); os.IsNotExist(err) {
					t.Error("Directory with content was removed")
				}
			},
		},
		{
			name: "clean dotfiles",
			config: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: ".config"},
				},
			},
			setup: testSetup{
				managedFiles: []config.ManagedFile{
					{Path: ".config/managed", LinkMode: "symlink", Source: ".config/managed"},
				},
				setupFunc: func(t *testing.T) {
					// Create directories
					if err := os.MkdirAll("overlay/.config", 0755); err != nil {
						t.Fatalf("Failed to create overlay directory: %v", err)
					}
					if err := os.MkdirAll(".upstream/.config", 0755); err != nil {
						t.Fatalf("Failed to create upstream directory: %v", err)
					}

					// Create managed symlink
					if err := os.WriteFile(".upstream/.config/managed", []byte("managed"), 0644); err != nil {
						t.Fatalf("Failed to create managed file: %v", err)
					}
					if err := os.Symlink(filepath.Join("..", "..", ".upstream", ".config", "managed"), filepath.Join("overlay", ".config", "managed")); err != nil {
						t.Fatalf("Failed to create managed symlink: %v", err)
					}

					// Create custom file
					if err := os.WriteFile("overlay/.config/custom", []byte("custom"), 0644); err != nil {
						t.Fatalf("Failed to create custom file: %v", err)
					}
				},
			},
			wantError: false,
			verifyPreserved: func(t *testing.T) {
				// Verify custom file still exists
				if _, err := os.Stat("overlay/.config/custom"); os.IsNotExist(err) {
					t.Error("Custom dotfile was removed")
				}

				// Verify managed symlink was removed
				if _, err := os.Stat("overlay/.config/managed"); !os.IsNotExist(err) {
					t.Error("Managed dotfile symlink was not removed")
				}
			},
		},
		{
			name: "clean non-existent overlay directory",
			config: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "test.txt"},
				},
			},
			setup: testSetup{
				managedFiles: []config.ManagedFile{},
				setupFunc:    func(t *testing.T) {},
			},
			wantError: true,
		},
		{
			name: "clean managed hardlinks",
			config: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "managed.txt"},
				},
			},
			setup: testSetup{
				managedFiles: []config.ManagedFile{
					{Path: "managed.txt", LinkMode: "hardlink", Source: "managed.txt"},
				},
				setupFunc: func(t *testing.T) {
					// Create overlay directory
					if err := os.MkdirAll("overlay", 0755); err != nil {
						t.Fatalf("Failed to create overlay directory: %v", err)
					}

					// Create source file
					if err := os.WriteFile(".upstream/managed.txt", []byte("managed"), 0644); err != nil {
						t.Fatalf("Failed to create managed file: %v", err)
					}

					// Create custom file
					if err := os.WriteFile("overlay/custom.txt", []byte("custom"), 0644); err != nil {
						t.Fatalf("Failed to create custom file: %v", err)
					}

					// Create managed hardlink
					if err := os.Link(".upstream/managed.txt", "overlay/managed.txt"); err != nil {
						t.Fatalf("Failed to create managed hardlink: %v", err)
					}
				},
			},
			wantError: false,
			verifyPreserved: func(t *testing.T) {
				// Verify custom file still exists
				if _, err := os.Stat("overlay/custom.txt"); os.IsNotExist(err) {
					t.Error("Custom file was removed")
				}

				// Verify managed hardlink was removed
				if _, err := os.Stat("overlay/managed.txt"); !os.IsNotExist(err) {
					t.Error("Managed hardlink was not removed")
				}
			},
		},
		{
			name: "clean managed copies",
			config: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "managed.txt"},
				},
			},
			setup: testSetup{
				managedFiles: []config.ManagedFile{
					{Path: "managed.txt", LinkMode: "copy", Source: "managed.txt"},
				},
				setupFunc: func(t *testing.T) {
					// Create overlay directory
					if err := os.MkdirAll("overlay", 0755); err != nil {
						t.Fatalf("Failed to create overlay directory: %v", err)
					}

					// Create source file
					if err := os.WriteFile(".upstream/managed.txt", []byte("managed"), 0644); err != nil {
						t.Fatalf("Failed to create managed file: %v", err)
					}

					// Create custom file
					if err := os.WriteFile("overlay/custom.txt", []byte("custom"), 0644); err != nil {
						t.Fatalf("Failed to create custom file: %v", err)
					}

					// Create managed copy
					if err := copyFile(".upstream/managed.txt", "overlay/managed.txt"); err != nil {
						t.Fatalf("Failed to create managed copy: %v", err)
					}
				},
			},
			wantError: false,
			verifyPreserved: func(t *testing.T) {
				// Verify custom file still exists
				if _, err := os.Stat("overlay/custom.txt"); os.IsNotExist(err) {
					t.Error("Custom file was removed")
				}

				// Verify managed copy was removed
				if _, err := os.Stat("overlay/managed.txt"); !os.IsNotExist(err) {
					t.Error("Managed copy was not removed")
				}
			},
		},
		{
			name: "clean empty directories",
			config: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "dir"},
				},
			},
			setup: testSetup{
				managedFiles: []config.ManagedFile{
					{Path: "dir/a/b/c/file.txt", LinkMode: "symlink", Source: "dir/a/b/c/file.txt"},
				},
				setupFunc: func(t *testing.T) {
					// Create nested directory structure
					if err := os.MkdirAll("overlay/dir/a/b/c", 0755); err != nil {
						t.Fatalf("Failed to create directories: %v", err)
					}
					if err := os.MkdirAll(".upstream/dir/a/b/c", 0755); err != nil {
						t.Fatalf("Failed to create upstream directories: %v", err)
					}

					// Create source file
					if err := os.WriteFile(".upstream/dir/a/b/c/file.txt", []byte("managed"), 0644); err != nil {
						t.Fatalf("Failed to create source file: %v", err)
					}

					// Create managed symlink
					if err := os.Symlink(filepath.Join("..", "..", "..", "..", ".upstream", "dir", "a", "b", "c", "file.txt"), filepath.Join("overlay", "dir", "a", "b", "c", "file.txt")); err != nil {
						t.Fatalf("Failed to create managed symlink: %v", err)
					}

					// Create some empty directories that should be cleaned up
					emptyDirs := []string{
						"overlay/dir/empty1",
						"overlay/dir/empty2/nested",
						"overlay/dir/a/empty3",
					}
					for _, dir := range emptyDirs {
						if err := os.MkdirAll(dir, 0755); err != nil {
							t.Fatalf("Failed to create empty directory %s: %v", dir, err)
						}
					}
				},
			},
			wantError: false,
			verifyPreserved: func(t *testing.T) {
				// Verify managed file was removed
				if _, err := os.Stat("overlay/dir/a/b/c/file.txt"); !os.IsNotExist(err) {
					t.Error("Managed file was not removed")
				}

				// Verify empty directories were removed
				emptyDirs := []string{
					"overlay/dir/empty1",
					"overlay/dir/empty2",
					"overlay/dir/empty2/nested",
					"overlay/dir/a/empty3",
					"overlay/dir/a/b/c",
					"overlay/dir/a/b",
					"overlay/dir/a",
				}
				for _, dir := range emptyDirs {
					if _, err := os.Stat(dir); !os.IsNotExist(err) {
						t.Errorf("Empty directory %s was not removed", dir)
					}
				}
			},
		},
		{
			name: "clean complex nested structure with multiple runs",
			config: &config.Config{
				Symlinks: []config.SymlinkSpec{
					{String: "dir"},
				},
			},
			setup: testSetup{
				managedFiles: []config.ManagedFile{
					// Branch 1: Deep nested files
					{Path: "dir/a/b/c/d/file1.txt", LinkMode: "symlink", Source: "dir/a/b/c/d/file1.txt"},
					{Path: "dir/a/b/c/file2.txt", LinkMode: "symlink", Source: "dir/a/b/c/file2.txt"},
					{Path: "dir/a/b/file3.txt", LinkMode: "symlink", Source: "dir/a/b/file3.txt"},
					{Path: "dir/a/file4.txt", LinkMode: "symlink", Source: "dir/a/file4.txt"},
					// Branch 2: Mixed managed and unmanaged
					{Path: "dir/x/y/managed1.txt", LinkMode: "symlink", Source: "dir/x/y/managed1.txt"},
					{Path: "dir/x/managed2.txt", LinkMode: "symlink", Source: "dir/x/managed2.txt"},
					// Branch 3: Different link modes
					{Path: "dir/p/q/hardlink.txt", LinkMode: "hardlink", Source: "dir/p/q/hardlink.txt"},
					{Path: "dir/p/copy.txt", LinkMode: "copy", Source: "dir/p/copy.txt"},
					// Directories
					{Path: "dir/a/b/c/d", LinkMode: "symlink", Source: "dir/a/b/c/d"},
					{Path: "dir/a/b/c", LinkMode: "symlink", Source: "dir/a/b/c"},
					{Path: "dir/a/b", LinkMode: "symlink", Source: "dir/a/b"},
					{Path: "dir/a", LinkMode: "symlink", Source: "dir/a"},
					{Path: "dir/x/y", LinkMode: "symlink", Source: "dir/x/y"},
					{Path: "dir/x", LinkMode: "symlink", Source: "dir/x"},
					{Path: "dir/p/q", LinkMode: "symlink", Source: "dir/p/q"},
					{Path: "dir/p", LinkMode: "symlink", Source: "dir/p"},
				},
				setupFunc: func(t *testing.T) {
					// Create directories
					dirs := []string{
						// Branch 1
						"overlay/dir/a/b/c/d",
						// Branch 2
						"overlay/dir/x/y",
						"overlay/dir/x/keep",
						// Branch 3
						"overlay/dir/p/q",
						// Upstream
						".upstream/dir/a/b/c/d",
						".upstream/dir/x/y",
						".upstream/dir/p/q",
					}
					for _, dir := range dirs {
						if err := os.MkdirAll(dir, 0755); err != nil {
							t.Fatalf("Failed to create directory %s: %v", dir, err)
						}
					}

					// Create source files
					sources := []string{
						// Branch 1
						".upstream/dir/a/b/c/d/file1.txt",
						".upstream/dir/a/b/c/file2.txt",
						".upstream/dir/a/b/file3.txt",
						".upstream/dir/a/file4.txt",
						// Branch 2
						".upstream/dir/x/y/managed1.txt",
						".upstream/dir/x/managed2.txt",
						// Branch 3
						".upstream/dir/p/q/hardlink.txt",
						".upstream/dir/p/copy.txt",
					}
					for _, src := range sources {
						if err := os.WriteFile(src, []byte("managed"), 0644); err != nil {
							t.Fatalf("Failed to create source file %s: %v", src, err)
						}
					}

					// Create managed symlinks
					links := map[string]string{
						// Branch 1
						"overlay/dir/a/b/c/d/file1.txt": "../../../../../.upstream/dir/a/b/c/d/file1.txt",
						"overlay/dir/a/b/c/file2.txt":   "../../../../.upstream/dir/a/b/c/file2.txt",
						"overlay/dir/a/b/file3.txt":     "../../../.upstream/dir/a/b/file3.txt",
						"overlay/dir/a/file4.txt":       "../../.upstream/dir/a/file4.txt",
						// Branch 2
						"overlay/dir/x/y/managed1.txt": "../../../.upstream/dir/x/y/managed1.txt",
						"overlay/dir/x/managed2.txt":   "../../.upstream/dir/x/managed2.txt",
					}
					for dst, src := range links {
						if err := os.Symlink(src, dst); err != nil {
							t.Fatalf("Failed to create symlink from %s to %s: %v", src, dst, err)
						}
					}

					// Create hardlink
					if err := os.Link(".upstream/dir/p/q/hardlink.txt", "overlay/dir/p/q/hardlink.txt"); err != nil {
						t.Fatalf("Failed to create hardlink: %v", err)
					}

					// Create copy
					if err := copyFile(".upstream/dir/p/copy.txt", "overlay/dir/p/copy.txt"); err != nil {
						t.Fatalf("Failed to create copy: %v", err)
					}

					// Create unmanaged files
					unmanaged := []string{
						"overlay/dir/x/y/custom1.txt",
						"overlay/dir/x/keep/custom2.txt",
					}
					for _, file := range unmanaged {
						if err := os.WriteFile(file, []byte("custom"), 0644); err != nil {
							t.Fatalf("Failed to create unmanaged file %s: %v", file, err)
						}
					}
				},
			},
			wantError: false,
			verifyPreserved: func(t *testing.T) {
				// First run should remove everything in one pass
				firstRunExpected := []string{
					// Files
					"overlay/dir/a/b/c/d/file1.txt",
					"overlay/dir/a/b/c/file2.txt",
					"overlay/dir/a/b/file3.txt",
					"overlay/dir/a/file4.txt",
					"overlay/dir/x/y/managed1.txt",
					"overlay/dir/x/managed2.txt",
					"overlay/dir/p/q/hardlink.txt",
					"overlay/dir/p/copy.txt",
					// Directories
					"overlay/dir/a/b/c/d",
					"overlay/dir/a/b/c",
					"overlay/dir/a/b",
					"overlay/dir/a",
					"overlay/dir/p/q",
					"overlay/dir/p",
				}

				// Verify all managed files and directories were removed
				for _, path := range firstRunExpected {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Errorf("Path %s was not removed in first run", path)
					}
				}

				// Verify unmanaged files still exist
				unmanaged := []string{
					"overlay/dir/x/y/custom1.txt",
					"overlay/dir/x/keep/custom2.txt",
				}
				for _, file := range unmanaged {
					if _, err := os.Stat(file); os.IsNotExist(err) {
						t.Errorf("Unmanaged file %s was removed", file)
					}
				}

				// Verify directories with content were preserved
				preserved := []string{
					"overlay/dir/x/y",    // Contains custom1.txt
					"overlay/dir/x",      // Contains keep directory
					"overlay/dir/x/keep", // Contains custom2.txt
				}
				for _, dir := range preserved {
					if _, err := os.Stat(dir); os.IsNotExist(err) {
						t.Errorf("Directory with content %s was removed", dir)
					}
				}

				// Run clean command again to verify idempotency
				cmd := &cobra.Command{
					Use:   "clean",
					Short: cleanCmd.Short,
					Long:  cleanCmd.Long,
					RunE:  cleanCmd.RunE,
				}
				cmd.Flags().String("config", ".git-overlay.yml", "")

				// Second run should not remove anything
				err := cmd.RunE(cmd, []string{})
				if err != nil {
					t.Errorf("Second clean run failed: %v", err)
				}

				// Verify state is consistent
				state, err := config.LoadState()
				if err != nil {
					t.Errorf("Failed to load state after second run: %v", err)
				}

				// State should be empty since all managed files were removed
				if len(state.ManagedFiles) > 0 {
					t.Errorf("State still contains %d files after cleanup", len(state.ManagedFiles))
				}

				// Verify unmanaged files still exist after second run
				for _, file := range unmanaged {
					if _, err := os.Stat(file); os.IsNotExist(err) {
						t.Errorf("Unmanaged file %s was removed after second run", file)
					}
				}

				// Verify preserved directories still exist after second run
				for _, dir := range preserved {
					if _, err := os.Stat(dir); os.IsNotExist(err) {
						t.Errorf("Directory with content %s was removed after second run", dir)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up from previous test
			os.RemoveAll("overlay")
			os.RemoveAll(".upstream")

			// Create .upstream directory
			if err := os.MkdirAll(".upstream", 0755); err != nil {
				t.Fatalf("Failed to create .upstream directory: %v", err)
			}

			// Setup test
			tt.setup.setupFunc(t)

			// Create config file with test configuration
			configContent := fmt.Sprintf(`upstream:
  url: "https://example.com/repo.git"
  ref: "main"
symlinks:
  - %s
`, tt.config.Symlinks[0].String)
			if err := os.WriteFile(".git-overlay.yml", []byte(configContent), 0644); err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			// Create state file
			state := &config.State{}
			for _, file := range tt.setup.managedFiles {
				state.AddManagedFile(file.Path, file.LinkMode, file.Source)
			}
			if err := state.SaveState(); err != nil {
				t.Fatalf("Failed to save state: %v", err)
			}

			// Create new command instance for each test
			cmd := &cobra.Command{
				Use:   "clean",
				Short: cleanCmd.Short,
				Long:  cleanCmd.Long,
				RunE:  cleanCmd.RunE,
			}
			cmd.Flags().String("config", ".git-overlay.yml", "")

			// Run clean command
			err := cmd.RunE(cmd, []string{})

			// Check error
			if (err != nil) != tt.wantError {
				t.Errorf("clean() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// Run verification if provided
			if tt.verifyPreserved != nil {
				tt.verifyPreserved(t)
			}
		})
	}
}
