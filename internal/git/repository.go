package git

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// Repository manages Git operations for both main and upstream repositories
type Repository struct {
	mainRepo     *git.Repository
	upstreamRepo *git.Repository
}

// InitMainRepository initializes the main repository if it doesn't exist
func InitMainRepository() (*Repository, error) {
	repo, err := git.PlainOpen(".")
	if err == git.ErrRepositoryNotExists {
		repo, err = git.PlainInit(".", false)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize repository: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Configure Git to allow file protocol
	cfg, err := repo.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	cfg.Raw.Section("protocol").SetOption("file", "allow")
	if err := repo.SetConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to set config: %w", err)
	}

	return &Repository{mainRepo: repo}, nil
}

// AddUpstreamSubmodule adds the upstream repository as a submodule
func (r *Repository) AddUpstreamSubmodule(url string) error {
	// Get worktree for main repository
	wt, err := r.mainRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Configure submodule
	cfg, err := r.mainRepo.Config()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	cfg.Submodules = make(map[string]*config.Submodule)
	cfg.Submodules["upstream"] = &config.Submodule{
		Name: "upstream",
		URL:  url,
		Path: ".upstream",
	}

	if err := r.mainRepo.SetConfig(cfg); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	// Create .gitmodules file
	gitmodulesContent := fmt.Sprintf(`[submodule "upstream"]
	path = .upstream
	url = %s
	ignore = all
`, url)
	if err := os.WriteFile(".gitmodules", []byte(gitmodulesContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitmodules: %w", err)
	}

	// Stage .gitmodules
	if _, err := wt.Add(".gitmodules"); err != nil {
		return fmt.Errorf("failed to stage .gitmodules: %w", err)
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "git-overlay-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone into temporary directory
	repo, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:        url,
		RemoteName: "origin",
		Progress:   nil,
		Tags:       git.AllTags,
	})
	if err != nil {
		return fmt.Errorf("failed to clone upstream repository: %w", err)
	}

	// Remove existing .upstream directory if it exists
	if err := os.RemoveAll(".upstream"); err != nil {
		return fmt.Errorf("failed to remove existing .upstream directory: %w", err)
	}

	// Move repository to .upstream
	if err := os.Rename(tmpDir, ".upstream"); err != nil {
		return fmt.Errorf("failed to move repository: %w", err)
	}

	// Open the repository again
	repo, err = git.PlainOpen(".upstream")
	if err != nil {
		return fmt.Errorf("failed to open upstream repository: %w", err)
	}

	// Configure Git to allow file protocol
	cfg, err = repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	cfg.Raw.Section("protocol").SetOption("file", "allow")
	if err := repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	r.upstreamRepo = repo

	// Stage the submodule and .gitmodules
	if _, err := wt.Add(".upstream"); err != nil {
		return fmt.Errorf("failed to stage submodule: %w", err)
	}

	return nil
}

// SyncUpstream updates the upstream repository to the specified ref
func (r *Repository) SyncUpstream(ref string) error {
	if r.upstreamRepo == nil {
		var err error
		r.upstreamRepo, err = git.PlainOpen(".upstream")
		if err != nil {
			return fmt.Errorf("failed to open upstream repository: %w", err)
		}
	}

	// Fetch all refs and tags
	var err error
	err = r.upstreamRepo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Force:      true,
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/remotes/origin/*",
			"+refs/tags/*:refs/tags/*",
		},
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch upstream: %w", err)
	}

	// Get worktree
	wt, err := r.upstreamRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Try to resolve the reference
	var hash plumbing.Hash

	// First try as a tag
	if tagRef, err := r.upstreamRepo.Reference(plumbing.NewTagReferenceName(ref), true); err == nil {
		hash = tagRef.Hash()
	} else if branchRef, err := r.upstreamRepo.Reference(plumbing.NewRemoteReferenceName("origin", ref), true); err == nil {
		// If not a tag, try as a branch
		hash = branchRef.Hash()
	} else if len(ref) == 40 {
		// If still not found, try as a commit hash
		hash = plumbing.NewHash(ref)
		if _, err := r.upstreamRepo.CommitObject(hash); err != nil {
			return fmt.Errorf("reference not found: %s", ref)
		}
	} else {
		return fmt.Errorf("reference not found: %s", ref)
	}

	// Checkout the reference
	err = wt.Checkout(&git.CheckoutOptions{
		Hash:  hash,
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout ref %s: %w", ref, err)
	}


	return nil
}
