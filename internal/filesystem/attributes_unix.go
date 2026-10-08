//go:build !windows

package filesystem

import (
	"fmt"
	"os"
)

func FormatFileAttributes(info os.FileInfo) string {
	return fmt.Sprintf("Permissions: %s (%04o)", info.Mode().String(), info.Mode().Perm())
}
