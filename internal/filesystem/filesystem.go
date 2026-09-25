package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Directory struct {
	Path  string
	Name  string
	Items []os.DirEntry
}

func Load(path, baseDir string) (Directory, error) {
	resolved, err := resolvePath(path, baseDir)
	if err != nil {
		return Directory{}, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return Directory{}, fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return Directory{}, fmt.Errorf("path is not a directory: %s", resolved)
	}

	items, err := os.ReadDir(resolved)
	if err != nil {
		return Directory{}, fmt.Errorf("read directory: %w", err)
	}

	return Directory{
		Path:  resolved,
		Name:  filepath.Base(resolved),
		Items: items,
	}, nil
}

func resolvePath(path, baseDir string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	path = os.ExpandEnv(path)
	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = homeDir
	} else if strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	resolved, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return resolved, nil
}
