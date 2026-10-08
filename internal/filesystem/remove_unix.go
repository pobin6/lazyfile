//go:build !windows

package filesystem

import "os"

func removeAll(path string) error {
	return os.RemoveAll(path)
}
