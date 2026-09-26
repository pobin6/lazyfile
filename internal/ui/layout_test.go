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

func TestSelectedPath(t *testing.T) {
	directory := filesystem.Directory{Path: "/tmp/目录"}
	app := App{current: directory}
	if got := app.selectedPath(); got != "/tmp/目录" {
		t.Fatalf("selected path = %q", got)
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

func TestNumericFocusNavigation(t *testing.T) {
	app := App{}

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "0", tcell.ModNone))
	if app.focusedCol != -1 {
		t.Fatalf("focused target = %d, want path bar (-1)", app.focusedCol)
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "1", tcell.ModNone))
	if app.focusedCol != 0 {
		t.Fatalf("focused column = %d, want 0", app.focusedCol)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "2", tcell.ModNone))
	if app.focusedCol != 1 {
		t.Fatalf("focused column = %d, want 1", app.focusedCol)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "3", tcell.ModNone))
	if app.focusedCol != 2 {
		t.Fatalf("focused column = %d, want 2 after 3", app.focusedCol)
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyRight, "", tcell.ModNone))
	if app.focusedCol != 2 {
		t.Fatalf("right arrow changed focus to %d", app.focusedCol)
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyLeft, "", tcell.ModNone))
	if app.focusedCol != 2 {
		t.Fatalf("left arrow changed focus to %d", app.focusedCol)
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

func TestCollectionNavigationAndPageSwitching(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := App{
		collections: []config.Collection{
			{Name: "one"},
			{Name: "two"},
		},
		collectionPage: true,
		focusedCol:     0,
	}

	app.syncCollectionEntries()

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "j", tcell.ModNone))
	if app.selectedCollection != 1 {
		t.Fatalf("selected collection = %d, want 1", app.selectedCollection)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "]", tcell.ModNone))
	if app.collectionPage {
		t.Fatal("expected entry page after ]")
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "[", tcell.ModNone))
	if !app.collectionPage {
		t.Fatal("expected collection page after [")
	}
}

func TestSecondColumnDirectoryNavigationPersistsState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}

	app := App{
		entries:    []config.Entry{{Name: "root", Path: root, CurrentPath: root}},
		selected:   0,
		focusedCol: 1,
	}
	app.current, _ = filesystem.Load(root, root)
	for index, item := range app.current.Items {
		if item.Name() == "child" {
			app.itemSelected = index
		}
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "l", tcell.ModNone))
	if app.current.Path != child || app.entries[0].CurrentPath != child {
		t.Fatalf("child navigation = %q, entry state = %q", app.current.Path, app.entries[0].CurrentPath)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "h", tcell.ModNone))
	if app.current.Path != root || app.current.Items[app.itemSelected].Name() != "child" {
		t.Fatalf("parent navigation = %q, want %q", app.current.Path, root)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "h", tcell.ModNone))
	if app.current.Path != root {
		t.Fatalf("parent navigation crossed entry root: %q", app.current.Path)
	}
}

func TestCopyCutAndPasteItems(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	if err := os.WriteFile(source, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	destination := t.TempDir()
	cutDestination := t.TempDir()
	app := App{
		current:    filesystem.Directory{Path: root},
		focusedCol: 1,
	}
	app.current, _ = filesystem.Load(root, root)
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "y", tcell.ModNone))
	if app.clipboardPath != source || app.clipboardCut {
		t.Fatalf("yank state = %q, cut=%t", app.clipboardPath, app.clipboardCut)
	}
	app.current, _ = filesystem.Load(destination, destination)
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "p", tcell.ModNone))
	if data, err := os.ReadFile(filepath.Join(destination, "source.txt")); err != nil || string(data) != "content" {
		t.Fatalf("pasted file = %q, err=%v", data, err)
	}

	app.current, _ = filesystem.Load(root, root)
	app.itemSelected = 0
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "x", tcell.ModNone))
	if app.clipboardPath != source || !app.clipboardCut {
		t.Fatalf("cut state = %q, cut=%t", app.clipboardPath, app.clipboardCut)
	}
	app.current, _ = filesystem.Load(cutDestination, cutDestination)
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "p", tcell.ModNone))
	if _, err := os.Stat(filepath.Join(root, "source.txt")); !os.IsNotExist(err) {
		t.Fatalf("cut source still exists, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(cutDestination, "source.txt")); err != nil {
		t.Fatalf("cut destination missing: %v", err)
	}
}

func TestCreateAndDeleteFilesModuleItems(t *testing.T) {
	root := t.TempDir()
	app := App{focusedCol: 1}
	app.current, _ = filesystem.Load(root, root)

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "a", tcell.ModNone))
	for _, key := range []string{"n", "o", "t", "e", ".", "t", "x", "t"} {
		app.handleKey(tcell.NewEventKey(tcell.KeyRune, key, tcell.ModNone))
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if _, err := os.Stat(filepath.Join(root, "note.txt")); err != nil {
		t.Fatalf("file was not created: %v", err)
	}

	app.createItem("a/b")
	if info, err := os.Stat(filepath.Join(root, "a", "b")); err != nil || !info.IsDir() {
		t.Fatalf("directory was not created: info=%v err=%v message=%q", info, err, app.errorMessage)
	}

	for index, item := range app.current.Items {
		if item.Name() == "note.txt" {
			app.itemSelected = index
		}
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "d", tcell.ModNone))
	if !app.confirmItemDelete {
		t.Fatalf("delete confirmation did not open: focus=%d items=%d selected=%d", app.focusedCol, len(app.current.Items), app.itemSelected)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "y", tcell.ModNone))
	if _, err := os.Stat(filepath.Join(root, "note.txt")); !os.IsNotExist(err) {
		t.Fatalf("file was not deleted: %v", err)
	}
}
