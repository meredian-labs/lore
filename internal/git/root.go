package git

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrNotAGitRepo = errors.New("not a git repository")

// FindGitRoot walks parent directories until it finds a .git directory.
func FindGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotAGitRepo
		}
		dir = parent
	}
}
