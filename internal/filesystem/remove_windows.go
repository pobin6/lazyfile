//go:build windows

package filesystem

import (
	"os"
	"path/filepath"
)

func removeAll(path string) error {
	err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && info.Mode().Perm()&0o200 == 0 {
			if err := os.Chmod(currentPath, info.Mode().Perm()|0o200); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.RemoveAll(path)
}
