package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lazyfile/internal/config"
	"lazyfile/internal/filesystem"

	"github.com/gdamore/tcell/v3"
)

func TestPreviewLines(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("first\nsecond"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "child", "nested.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	directory, err := filesystem.Load(root, root)
	if err != nil {
		t.Fatal(err)
	}
	app := App{current: directory}

	for index, item := range directory.Items {
		if item.Name() == "note.txt" {
			app.itemSelected = index
			lines := app.previewLines()
			if len(lines) < 3 || lines[1] != "first" || lines[2] != "second" {
				t.Fatalf("file preview = %q", lines)
			}
		}

		if item.Name() == "child" {
			app.itemSelected = index
			lines := app.previewLines()
			if len(lines) != 2 || lines[1] != "nested.txt" {
				t.Fatalf("directory preview = %q", lines)
			}
		}
	}
}

func TestPreviewLinesLimitsFileReadsToOneKiB(t *testing.T) {
	root := t.TempDir()
	content := strings.Repeat("x", maxPreviewBytes+100)
	if err := os.WriteFile(filepath.Join(root, "large.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	directory, err := filesystem.Load(root, root)
	if err != nil {
		t.Fatal(err)
	}
	app := App{current: directory, itemSelected: 0}

	lines := app.previewLines()
	if lines[len(lines)-1] != "[Preview truncated after 1 KiB]" {
		t.Fatalf("last preview line = %q, want truncation notice", lines[len(lines)-1])
	}
	previewContent := strings.Join(lines[1:len(lines)-1], "\n")
	if len(previewContent) != maxPreviewBytes {
		t.Fatalf("preview bytes = %d, want %d", len(previewContent), maxPreviewBytes)
	}
}

func TestCalculateLayout(t *testing.T) {
	layout := calculateLayout(100)
	if layout.LeftWidth != 20 || layout.MiddleWidth != 40 || layout.RightWidth != 40 {
		t.Fatalf("layout = %+v, want 20/40/40", layout)
	}
}

func TestEnsureItemVisibleKeepsSelectionAwayFromViewportEdges(t *testing.T) {
	app := App{
		current: filesystem.Directory{
			Items: make([]os.DirEntry, 20),
		},
		itemSelected: 8,
	}

	app.ensureItemVisible(10)
	if app.itemScroll != 2 {
		t.Fatalf("item scroll = %d, want 2", app.itemScroll)
	}

	app.itemSelected = 19
	app.ensureItemVisible(10)
	if app.itemScroll != 10 {
		t.Fatalf("item scroll = %d, want 10", app.itemScroll)
	}

	app.itemSelected = 0
	app.ensureItemVisible(10)
	if app.itemScroll != 0 {
		t.Fatalf("item scroll = %d, want 0", app.itemScroll)
	}
}

func TestMoveFocus(t *testing.T) {
	app := App{}

	app.moveFocus(-1)
	if app.focusedCol != 0 {
		t.Fatalf("focused column = %d, want 0", app.focusedCol)
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "l", tcell.ModNone))
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "l", tcell.ModNone))
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "l", tcell.ModNone))
	if app.focusedCol != 2 {
		t.Fatalf("focused column = %d, want 2", app.focusedCol)
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyRight, "", tcell.ModNone))
	if app.focusedCol != 2 {
		t.Fatalf("focused column = %d, want 2 after right clamp", app.focusedCol)
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyLeft, "", tcell.ModNone))
	if app.focusedCol != 1 {
		t.Fatalf("focused column = %d, want 1 after left", app.focusedCol)
	}
}

func TestEntryNavigationAndDeleteConfirmation(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := App{
		entries: []config.Entry{
			{Name: "one", Path: "/one"},
			{Name: "two", Path: "/two"},
		},
		focusedCol: 0,
		selected:   0,
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "j", tcell.ModNone))
	if app.selected != 1 {
		t.Fatalf("selected = %d, want 1 after j", app.selected)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "k", tcell.ModNone))
	if app.selected != 0 {
		t.Fatalf("selected = %d, want 0 after k", app.selected)
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "d", tcell.ModNone))
	if !app.confirmDelete {
		t.Fatal("delete did not open confirmation")
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "n", tcell.ModNone))
	if app.confirmDelete || len(app.entries) != 2 {
		t.Fatalf("cancel changed delete state: confirm=%t entries=%d", app.confirmDelete, len(app.entries))
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "d", tcell.ModNone))
	app.handleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if app.confirmDelete || len(app.entries) != 1 || app.entries[0].Name != "two" {
		t.Fatalf("confirmed delete state: confirm=%t entries=%+v", app.confirmDelete, app.entries)
	}
}
