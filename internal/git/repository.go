package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

const gitmodTemplate = `[submodule "{{.Name}}"]
	path = {{.Path}}
	url = {{.URL}}
	ignore = all
`

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
	// Create submodule spec
	spec := config.Submodule{
		Name: "upstream",
		Path: ".upstream",
		URL:  url,
	}

	// Get worktree
	wt, err := r.mainRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Create/update .gitmodules file
	gitmodulesFile := filepath.Join(wt.Filesystem.Root(), ".gitmodules")
	f, err := os.OpenFile(gitmodulesFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create .gitmodules: %w", err)
	}

	// Write submodule config using template
	t := template.Must(template.New("gitmodule").Parse(gitmodTemplate))
	if err := t.Execute(f, spec); err != nil {
		return fmt.Errorf("failed to write .gitmodules: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close .gitmodules: %w", err)
	}

	// Get submodule
	sub, err := wt.Submodule("upstream")
	if err != nil {
		return fmt.Errorf("failed to get submodule: %w", err)
	}

	// Initialize submodule
	if err := sub.Init(); err != nil {
		return fmt.Errorf("failed to init submodule: %w", err)
	}

	// Get submodule repo
	r.upstreamRepo, err = sub.Repository()
	if err != nil {
		return fmt.Errorf("failed to get submodule repository: %w", err)
	}

	// Get submodule worktree
	subwt, err := r.upstreamRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get submodule worktree: %w", err)
	}

	// Pull changes
	if err := subwt.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull submodule: %w", err)
	}

	head, err := r.upstreamRepo.Head()
	if err != nil {
		return fmt.Errorf("failed to get submodule head: %w", err)
	}
	commitHash := head.Hash().String()

	// Update the parent index with the gitlink for .upstream
	cmd := exec.Command("git", "update-index", "--add", "--cacheinfo", "160000", commitHash, ".upstream")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update index: %v, output: %s", err, output)
	}

	// Ensure .gitignore from upstream is copied, breaking any symlink
	upstreamGitIgnore := ".upstream/.gitignore"
	if stat, err := os.Lstat(upstreamGitIgnore); err == nil {
		data, err := os.ReadFile(upstreamGitIgnore)
		if err != nil {
			return fmt.Errorf("failed to read upstream .gitignore: %w", err)
		}
		if stat.Mode()&os.ModeSymlink != 0 {
			// Remove the symlink and write a fresh copy as a regular file
			if err := os.Remove(upstreamGitIgnore); err != nil {
				return fmt.Errorf("failed to remove symlink for upstream .gitignore: %w", err)
			}
			if err := os.WriteFile(upstreamGitIgnore, data, 0644); err != nil {
				return fmt.Errorf("failed to copy upstream .gitignore: %w", err)
			}
		} else {
			// Always recopy even if it's a regular file
			if err := os.WriteFile(upstreamGitIgnore, data, 0644); err != nil {
				return fmt.Errorf("failed to recopy upstream .gitignore: %w", err)
			}
		}
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

	// Get worktree
	wt, err := r.upstreamRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Fetch all refs
	err = r.upstreamRepo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Force:      true,
		Progress:   os.Stdout,
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/remotes/origin/*",
			"+refs/tags/*:refs/tags/*",
		},
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch upstream: %w", err)
	}

	// Pull changes
	err = wt.Pull(&git.PullOptions{
		RemoteName: "origin",
		Force:      true,
		Progress:   os.Stdout,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull upstream: %w", err)
	}

	// Get remote reference first
	remoteRef, err := r.upstreamRepo.Reference(plumbing.NewRemoteReferenceName("origin", ref), true)
	if err == nil {
		// Found as remote branch
		err = wt.Checkout(&git.CheckoutOptions{
			Hash:  remoteRef.Hash(),
			Force: true,
		})
		return err
	}

	// Try as tag
	tagRef, err := r.upstreamRepo.Reference(plumbing.NewTagReferenceName(ref), true)
	if err == nil {
		err = wt.Checkout(&git.CheckoutOptions{
			Hash:  tagRef.Hash(),
			Force: true,
		})
		return err
	}

	// Try as hash
	hash := plumbing.NewHash(ref)
	return wt.Checkout(&git.CheckoutOptions{
		Hash:  hash,
		Force: true,
	})
}
