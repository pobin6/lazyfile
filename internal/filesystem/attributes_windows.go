//go:build windows

package filesystem

import (
	"os"
	"strings"
	"syscall"
)

func FormatFileAttributes(info os.FileInfo) string {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok || data == nil {
		return "Attributes: unavailable"
	}

	attributes := make([]string, 0, 3)
	if data.FileAttributes&syscall.FILE_ATTRIBUTE_READONLY != 0 {
		attributes = append(attributes, "Read-only")
	}
	if data.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0 {
		attributes = append(attributes, "Hidden")
	}
	if data.FileAttributes&syscall.FILE_ATTRIBUTE_SYSTEM != 0 {
		attributes = append(attributes, "System")
	}
	if len(attributes) == 0 {
		attributes = append(attributes, "Normal")
	}
	return "Attributes: " + strings.Join(attributes, ", ")
}
