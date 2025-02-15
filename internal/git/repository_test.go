package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "git-overlay-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cleanup := func() {
		os.Chdir(originalDir)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func setupUpstreamRepo(t *testing.T, dir string) string {
	t.Helper()

	// Create and initialize upstream repository
	upstreamDir := filepath.Join(dir, "upstream")
	if err := os.MkdirAll(upstreamDir, 0755); err != nil {
		t.Fatalf("Failed to create upstream dir: %v", err)
	}

	repo, err := git.PlainInit(upstreamDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize upstream repository: %v", err)
	}

	// Configure Git to allow file protocol
	cfg, err := repo.Config()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}
	cfg.Raw.Section("protocol").SetOption("file", "allow")
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Create test file
	testFile := filepath.Join(upstreamDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add and commit file
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	_, err = wt.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}

	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit test file: %v", err)
	}

	// Create and checkout main branch
	wt, err = repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("main"),
		Create: true,
	})
	if err != nil {
		t.Fatalf("Failed to create main branch: %v", err)
	}

	return upstreamDir
}

func TestInitMainRepository(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Test initialization of new repository
	repo, err := InitMainRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	if repo.mainRepo == nil {
		t.Error("Expected mainRepo to be initialized")
	}

	// Test opening existing repository
	repo2, err := InitMainRepository()
	if err != nil {
		t.Fatalf("Failed to open existing repository: %v", err)
	}

	if repo2.mainRepo == nil {
		t.Error("Expected mainRepo to be initialized")
	}
}

func TestAddUpstreamSubmodule(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Setup upstream repository
	upstreamDir := setupUpstreamRepo(t, tmpDir)

	// Initialize main repository
	repo, err := InitMainRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Add upstream submodule
	if err := repo.AddUpstreamSubmodule(upstreamDir); err != nil {
		t.Fatalf("Failed to add upstream submodule: %v", err)
	}

	// Verify .upstream directory exists
	if _, err := os.Stat(".upstream"); os.IsNotExist(err) {
		t.Error("Expected .upstream directory to exist")
	}

	// Verify test file exists in .upstream
	if _, err := os.Stat(filepath.Join(".upstream", "test.txt")); os.IsNotExist(err) {
		t.Error("Expected test.txt to exist in .upstream")
	}

	// Verify submodule configuration
	cfg, err := repo.mainRepo.Config()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if cfg.Submodules["upstream"] == nil {
		t.Error("Expected upstream submodule configuration to exist")
	}

	if cfg.Submodules["upstream"].URL != upstreamDir {
		t.Errorf("Expected submodule URL to be %s, got %s", upstreamDir, cfg.Submodules["upstream"].URL)
	}
}

func TestSyncUpstream(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Setup upstream repository
	upstreamDir := setupUpstreamRepo(t, tmpDir)

	// Initialize main repository and add submodule
	repo, err := InitMainRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	if err := repo.AddUpstreamSubmodule(upstreamDir); err != nil {
		t.Fatalf("Failed to add upstream submodule: %v", err)
	}

	// Create new file in upstream
	newFile := filepath.Join(upstreamDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Add and commit new file in upstream
	upstreamRepo, err := git.PlainOpen(upstreamDir)
	if err != nil {
		t.Fatalf("Failed to open upstream repository: %v", err)
	}

	wt, err := upstreamRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	_, err = wt.Add("new.txt")
	if err != nil {
		t.Fatalf("Failed to add new file: %v", err)
	}

	_, err = wt.Commit("Add new file", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit new file: %v", err)
	}

	// Sync upstream
	if err := repo.SyncUpstream("main"); err != nil {
		t.Fatalf("Failed to sync upstream: %v", err)
	}

	// Verify new file exists in .upstream
	if _, err := os.Stat(filepath.Join(".upstream", "new.txt")); os.IsNotExist(err) {
		t.Error("Expected new.txt to exist in .upstream")
	}
}
