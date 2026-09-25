package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResolvesRelativePathAndReadsItems(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	directory, err := Load(".", root)
	if err != nil {
		t.Fatal(err)
	}

	if directory.Path != root {
		t.Fatalf("path = %q, want %q", directory.Path, root)
	}
	if directory.Name != filepath.Base(root) {
		t.Fatalf("name = %q, want %q", directory.Name, filepath.Base(root))
	}
	if len(directory.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(directory.Items))
	}
}

func TestLoadRejectsFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(file, root); err == nil {
		t.Fatal("Load(file) returned nil error")
	}
}

func TestLoadExpandsHomeAndEnvironmentVariables(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LAZYFILE_TEST_HOME", root)
	if err := os.Mkdir(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"$LAZYFILE_TEST_HOME", "${LAZYFILE_TEST_HOME}", "~"} {
		t.Run(path, func(t *testing.T) {
			if path == "~" {
				t.Setenv("HOME", root)
			}
			directory, err := Load(path, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if directory.Path != root {
				t.Fatalf("path = %q, want %q", directory.Path, root)
			}
		})
	}
}

func TestLoadExpandsHomeSubdirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}

	directory, err := Load("~/child", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if directory.Path != child {
		t.Fatalf("path = %q, want %q", directory.Path, child)
	}
}
