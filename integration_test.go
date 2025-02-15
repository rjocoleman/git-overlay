package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupGitRepo(t *testing.T, path string) {
	t.Helper()

	// Initialize git repo with initial branch
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = path
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Create test file
	testFile := filepath.Join(path, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add and commit file
	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = path
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit test file: %v", err)
	}
}

func TestEndToEnd(t *testing.T) {
	// Create temporary test directories
	tmpDir, err := os.MkdirTemp("", "git-overlay-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create upstream repository
	upstreamDir := filepath.Join(tmpDir, "upstream")
	if err := os.MkdirAll(upstreamDir, 0755); err != nil {
		t.Fatalf("Failed to create upstream dir: %v", err)
	}
	setupGitRepo(t, upstreamDir)

	// Create overlay repository
	overlayDir := filepath.Join(tmpDir, "overlay")
	if err := os.MkdirAll(overlayDir, 0755); err != nil {
		t.Fatalf("Failed to create overlay dir: %v", err)
	}

	// Change to overlay directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(overlayDir); err != nil {
		t.Fatalf("Failed to change to overlay directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create config file
	configContent := []byte(fmt.Sprintf(`upstream:
  url: "%s"
  ref: "main"
symlinks:
  - test.txt
`, filepath.Join(upstreamDir)))
	if err := os.WriteFile(".git-overlay.yml", configContent, 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Initialize overlay repository with initial branch
	overlayCmd := exec.Command("git", "init", "-b", "main")
	overlayCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if err := overlayCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize overlay repository: %v", err)
	}

	// Configure Git to allow file protocol (local to repository)
	gitCmd := exec.Command("git", "config", "protocol.file.allow", "always")
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("Failed to configure Git protocol: %v", err)
	}

	// Clean up any existing .upstream directory
	if err := os.RemoveAll(".upstream"); err != nil {
		t.Fatalf("Failed to clean up .upstream directory: %v", err)
	}

	// Initialize Go module
	cmd := exec.Command("go", "mod", "init", "test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize Go module: %v", err)
	}

	// Copy project files to test directory
	files := []string{"main.go", "go.mod", "go.sum"}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(originalDir, file))
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("Failed to read %s: %v", file, err)
		}
		if err == nil {
			if err := os.WriteFile(file, data, 0644); err != nil {
				t.Fatalf("Failed to write %s: %v", file, err)
			}
		}
	}

	// Copy cmd directory
	if err := filepath.Walk(filepath.Join(originalDir, "cmd"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(filepath.Join(originalDir, "cmd"), path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join("cmd", relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0644)
	}); err != nil {
		t.Fatalf("Failed to copy cmd directory: %v", err)
	}

	// Copy internal directory
	if err := filepath.Walk(filepath.Join(originalDir, "internal"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(filepath.Join(originalDir, "internal"), path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join("internal", relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0644)
	}); err != nil {
		t.Fatalf("Failed to copy internal directory: %v", err)
	}

	// Test init command with force flag
	cmd = exec.Command("go", "run", ".")
	cmd.Args = append(cmd.Args, "init", "--force")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run init command: %v", err)
	}

	// Verify .upstream directory and submodule were created
	if _, err := os.Stat(".upstream"); os.IsNotExist(err) {
		t.Error("Expected .upstream directory to exist")
	}
	if _, err := os.Stat(".upstream/test.txt"); os.IsNotExist(err) {
		t.Error("Expected upstream test file to exist")
	}

	// Verify overlay symlink was created
	if _, err := os.Stat("overlay/test.txt"); os.IsNotExist(err) {
		t.Error("Expected overlay symlink to exist")
	}

	// Create new file in upstream
	newFile := filepath.Join(upstreamDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Add and commit new file in upstream
	cmd = exec.Command("git", "add", "new.txt")
	cmd.Dir = upstreamDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add new file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add new file")
	cmd.Dir = upstreamDir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit new file: %v", err)
	}

	// Update config to include new file
	configContent = []byte(fmt.Sprintf(`upstream:
  url: "%s"
  ref: "main"
symlinks:
  - test.txt
  - new.txt
`, filepath.Join(upstreamDir)))
	if err := os.WriteFile(".git-overlay.yml", configContent, 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// Test sync command
	cmd = exec.Command("go", "run", ".")
	cmd.Args = append(cmd.Args, "sync", "--force")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run sync command: %v", err)
	}

	// Verify new file was synced
	if _, err := os.Stat(".upstream/new.txt"); os.IsNotExist(err) {
		t.Error("Expected new upstream file to exist")
	}
	if _, err := os.Stat("overlay/new.txt"); os.IsNotExist(err) {
		t.Error("Expected new overlay symlink to exist")
	}
}
