package git

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repo wraps a go-git repository at a filesystem path.
type Repo struct {
	path string
}

// Open returns a Repo handle for the given directory path.
func Open(path string) *Repo {
	return &Repo{path: path}
}

// Path returns the repository root directory.
func (r *Repo) Path() string {
	return r.path
}

// Init creates a dedicated Substrate document repo with the standard layout.
// Idempotent: if the repo already exists, ensures layout directories exist.
func Init(path string) (*Repo, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("create repo dir: %w", err)
	}

	repo, err := gogit.PlainOpen(path)
	if err == gogit.ErrRepositoryNotExists {
		repo, err = gogit.PlainInit(path, false)
		if err != nil {
			return nil, fmt.Errorf("init git repo: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("open git repo: %w", err)
	}

	r := &Repo{path: path}
	if err := r.ensureLayout(); err != nil {
		return nil, err
	}

	// Initial commit if the repo has no commits yet.
	if _, err := repo.Head(); err != nil {
		if err := r.commitAll(repo, "substrate: init document repository"); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (r *Repo) ensureLayout() error {
	dirs := []string{"specs", "decisions", "gates", "components"}
	for _, d := range dirs {
		dir := filepath.Join(r.path, d)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
		keep := filepath.Join(dir, ".gitkeep")
		if _, err := os.Stat(keep); os.IsNotExist(err) {
			if err := os.WriteFile(keep, []byte{}, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", keep, err)
			}
		}
	}

	readme := filepath.Join(r.path, "README.md")
	if _, err := os.Stat(readme); os.IsNotExist(err) {
		content := "# Substrate Document Repository\n\nSpecs, decisions, gates, and component definitions live here.\n"
		if err := os.WriteFile(readme, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write README: %w", err)
		}
	}

	return nil
}

// Commit stages all changes and creates a commit with the given message.
func (r *Repo) Commit(message string) error {
	repo, err := gogit.PlainOpen(r.path)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}
	return r.commitAll(repo, message)
}

func (r *Repo) commitAll(repo *gogit.Repository, message string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	if _, err := wt.Add("."); err != nil {
		return fmt.Errorf("stage files: %w", err)
	}

	_, err = wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "substrate",
			Email: "substrate@local",
			When:  time.Now().UTC(),
		},
		All: true,
	})
	if err == gogit.ErrEmptyCommit {
		return nil
	}
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
