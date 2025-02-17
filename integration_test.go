package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rjocoleman/git-overlay/cmd"
	"github.com/rjocoleman/git-overlay/internal/config"
	igit "github.com/rjocoleman/git-overlay/internal/git" // alias to avoid conflict
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func setupTestRepo(t *testing.T, path string) {
	t.Helper()

	// Initialize git repository
	repo, err := git.PlainInit(path, false)
	if err != nil {
		t.Fatalf("failed to initialize git repo: %v", err)
	}

	// Create test file
	testFile := filepath.Join(path, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Stage and commit file
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := wt.Add("test.txt"); err != nil {
		t.Fatalf("failed to stage test file: %v", err)
	}

	if _, err := wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		t.Fatalf("failed to commit test file: %v", err)
	}
}

func TestEndToEnd(t *testing.T) {
	// Create test directories
	tmpDir := t.TempDir()
	upstreamDir := filepath.Join(tmpDir, "upstream")
	overlayDir := filepath.Join(tmpDir, "overlay")

	// Setup test repositories
	if err := os.MkdirAll(upstreamDir, 0755); err != nil {
		t.Fatalf("failed to create upstream dir: %v", err)
	}
	setupTestRepo(t, upstreamDir)

	// Create and change to overlay directory
	if err := os.MkdirAll(overlayDir, 0755); err != nil {
		t.Fatalf("failed to create overlay dir: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err := os.Chdir(overlayDir); err != nil {
		t.Fatalf("failed to change to overlay dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create test config
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: upstreamDir,
			Ref: "main",
		},
		Symlinks: []config.SymlinkSpec{
			{String: "test.txt"},
		},
	}
	configBytes, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(".git-overlay.yml", configBytes, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create cobra command for flags
	command := &cobra.Command{}
	command.Flags().String("config", ".git-overlay.yml", "")
	command.Flags().Bool("force", true, "")
	command.Flags().String("link-mode", "symlink", "")

	// Initialize repository
	repo, err := igit.InitMainRepository()
	if err != nil {
		t.Fatalf("failed to initialize repository: %v", err)
	}

	// Add upstream submodule
	if err := repo.AddUpstreamSubmodule(cfg.Upstream.URL); err != nil {
		t.Fatalf("failed to add upstream submodule: %v", err)
	}

	// Sync to ref
	if err := repo.SyncUpstream(cfg.Upstream.Ref); err != nil {
		t.Fatalf("failed to sync upstream: %v", err)
	}

	// Create links
	if err := cmd.CreateLinks(command, cfg); err != nil {
		t.Fatalf("failed to create links: %v", err)
	}

	// Verify state
	if _, err := os.Stat(".upstream"); os.IsNotExist(err) {
		t.Error("expected .upstream directory to exist")
	}
	if _, err := os.Stat(".upstream/test.txt"); os.IsNotExist(err) {
		t.Error("expected upstream test file to exist")
	}
	if _, err := os.Stat("overlay/test.txt"); os.IsNotExist(err) {
		t.Error("expected overlay symlink to exist")
	}

	// Create new file in upstream
	newFile := filepath.Join(upstreamDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("failed to create new file: %v", err)
	}

	// Stage and commit new file in upstream
	upstreamRepo, err := git.PlainOpen(upstreamDir)
	if err != nil {
		t.Fatalf("failed to open upstream repo: %v", err)
	}
	wt, err := upstreamRepo.Worktree()
	if err != nil {
		t.Fatalf("failed to get upstream worktree: %v", err)
	}
	if _, err := wt.Add("new.txt"); err != nil {
		t.Fatalf("failed to stage new file: %v", err)
	}
	if _, err := wt.Commit("Add new file", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		t.Fatalf("failed to commit new file: %v", err)
	}

	// Sync changes first
	if err := repo.SyncUpstream(cfg.Upstream.Ref); err != nil {
		t.Fatalf("failed to sync upstream: %v", err)
	}

	// Verify new file exists in .upstream
	if _, err := os.Stat(".upstream/new.txt"); os.IsNotExist(err) {
		t.Error("expected new upstream file to exist")
	}

	// Now update config and create links
	cfg.Symlinks = append(cfg.Symlinks, config.SymlinkSpec{String: "new.txt"})
	configBytes, err = yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal updated config: %v", err)
	}
	if err := os.WriteFile(".git-overlay.yml", configBytes, 0644); err != nil {
		t.Fatalf("failed to write updated config: %v", err)
	}

	if err := cmd.CreateLinks(command, cfg); err != nil {
		t.Fatalf("failed to create links: %v", err)
	}

	// Verify symlink was created
	if _, err := os.Stat("overlay/new.txt"); os.IsNotExist(err) {
		t.Error("expected new overlay symlink to exist")
	}
}
