package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
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

	for _, path := range []string{"$LAZYFILE_TEST_HOME", "${LAZYFILE_TEST_HOME}", "%LAZYFILE_TEST_HOME%", "~"} {
		t.Run(path, func(t *testing.T) {
			if path == "~" {
				t.Setenv("HOME", root)
				if runtime.GOOS == "windows" {
					t.Setenv("USERPROFILE", root)
				}
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
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", root)
	}
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"~/child", `~\child`} {
		t.Run(path, func(t *testing.T) {
			directory, err := Load(path, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if directory.Path != child {
				t.Fatalf("path = %q, want %q", directory.Path, child)
			}
		})
	}
}

func TestMoveAcrossDevicesCopiesThenRemovesSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	destination := filepath.Join(root, "destination.txt")
	if err := os.WriteFile(source, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := moveAcrossDevices(source, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists after move: %v", err)
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "content" {
		t.Fatalf("destination = %q, err=%v", data, err)
	}
}

func TestRemoveAllRemovesReadOnlyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "readonly.txt")
	if err := os.WriteFile(path, []byte("content"), 0o444); err != nil {
		t.Fatal(err)
	}

	if err := RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("read-only file still exists: %v", err)
	}
}

func TestResolvePathWindowsAbsoluteForms(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows absolute path forms are only valid on Windows")
	}
	for _, path := range []string{`C:\Users\example\folder`, `\\server\share\folder`} {
		t.Run(path, func(t *testing.T) {
			resolved, err := resolvePath(path, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if !filepath.IsAbs(resolved) || filepath.VolumeName(resolved) == "" {
				t.Fatalf("resolved path = %q, want an absolute path with a volume", resolved)
			}
		})
	}
}
