//go:build windows

package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatFileAttributesReportsReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "readonly.txt")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatFileAttributes(info); !strings.Contains(got, "Read-only") {
		t.Fatalf("attributes = %q, want Read-only", got)
	}
}
