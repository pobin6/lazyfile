package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lazyfile/internal/config"
	"lazyfile/internal/filesystem"

	"github.com/clipperhouse/displaywidth"
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

func TestEscapeClearsFileSelectionWithoutQuitting(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one.txt", "two.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	directory, err := filesystem.Load(root, root)
	if err != nil {
		t.Fatal(err)
	}
	app := App{
		current:       directory,
		focusedCol:    1,
		itemSelected:  1,
		previewOffset: 2,
	}

	if shouldQuit := app.handleKey(tcell.NewEventKey(tcell.KeyEscape, "", tcell.ModNone)); shouldQuit {
		t.Fatal("Escape requested application quit")
	}
	if app.itemSelected != len(app.current.Items) {
		t.Fatalf("item selection = %d, want empty selection sentinel %d", app.itemSelected, len(app.current.Items))
	}
	if app.previewOffset != 0 {
		t.Fatalf("preview offset = %d, want 0 after clearing selection", app.previewOffset)
	}
	if got := app.previewLines(); len(got) != 0 {
		t.Fatalf("preview lines = %q, want none after clearing selection", got)
	}
	if got := app.selectedItemInfo(); len(got) != 1 || got[0] != "No file selected" {
		t.Fatalf("selected item info = %q, want no file selected", got)
	}

	app.selectItem(1)
	if app.itemSelected != 0 {
		t.Fatalf("item selection after j = %d, want first item (0)", app.itemSelected)
	}
}

func TestSearchFilesByPartialNameAndNavigateMatches(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"report-one.txt", "report-two.txt", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	directory, err := filesystem.Load(root, root)
	if err != nil {
		t.Fatal(err)
	}
	app := App{current: directory, focusedCol: 1}

	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "/", tcell.ModNone))
	for _, character := range "REPORT-" {
		app.handleKey(tcell.NewEventKey(tcell.KeyRune, string(character), tcell.ModNone))
	}
	if len(app.searchMatches) != 2 {
		t.Fatalf("search matches = %v, want two case-insensitive matches", app.searchMatches)
	}
	first, second := app.searchMatches[0], app.searchMatches[1]
	if !app.isCurrentSearchMatch(1, first) || !app.isSearchMatch(1, second) {
		t.Fatalf("search match highlighting not set for first and subsequent results: %v", app.searchMatches)
	}
	if app.isSearchMatch(0, first) {
		t.Fatal("Files search incorrectly matched the first column")
	}

	app.handleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if app.searchInput || app.itemSelected != first {
		t.Fatalf("search enter state: input=%t selected=%d, want false and %d", app.searchInput, app.itemSelected, first)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "n", tcell.ModNone))
	if app.itemSelected != second {
		t.Fatalf("n selected item %d, want next match %d", app.itemSelected, second)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "N", tcell.ModNone))
	if app.itemSelected != first {
		t.Fatalf("N selected item %d, want previous match %d", app.itemSelected, first)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyEscape, "", tcell.ModNone))
	if app.searchQuery != "" || len(app.searchMatches) != 0 {
		t.Fatalf("Escape did not clear search: query=%q matches=%v", app.searchQuery, app.searchMatches)
	}
}

func TestSearchFirstColumnAndSearchByFileLineNumber(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := App{
		focusedCol:     0,
		collectionPage: true,
		collections: []config.Collection{
			{Name: "Work"},
			{Name: "Work archive"},
			{Name: "Personal"},
		},
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "/", tcell.ModNone))
	for _, character := range "work" {
		app.handleKey(tcell.NewEventKey(tcell.KeyRune, string(character), tcell.ModNone))
	}
	if len(app.searchMatches) != 2 || app.isSearchMatch(1, 0) {
		t.Fatalf("first-column matches = %v, expected two matches isolated to column one", app.searchMatches)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if app.selectedCollection != 0 {
		t.Fatalf("first matched collection = %d, want 0", app.selectedCollection)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "n", tcell.ModNone))
	if app.selectedCollection != 1 {
		t.Fatalf("next collection match = %d, want 1", app.selectedCollection)
	}

	entryRoot := t.TempDir()
	firstPath := filepath.Join(entryRoot, "docs")
	secondPath := filepath.Join(entryRoot, "docs archive")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	app = App{
		baseDir:    entryRoot,
		focusedCol: 0,
		entries: []config.Entry{
			{Name: "docs", Path: firstPath},
			{Name: "docs archive", Path: secondPath},
		},
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "/", tcell.ModNone))
	for _, character := range "docs" {
		app.handleKey(tcell.NewEventKey(tcell.KeyRune, string(character), tcell.ModNone))
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if app.selected != 0 || app.current.Path != firstPath {
		t.Fatalf("first bookmark match selected=%d path=%q, want index 0 path %q", app.selected, app.current.Path, firstPath)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "n", tcell.ModNone))
	if app.selected != 1 || app.current.Path != secondPath {
		t.Fatalf("next bookmark match selected=%d path=%q, want index 1 path %q", app.selected, app.current.Path, secondPath)
	}

	root := t.TempDir()
	for _, name := range []string{"alpha.txt", "bravo.txt", "charlie.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	directory, err := filesystem.Load(root, root)
	if err != nil {
		t.Fatal(err)
	}
	app = App{current: directory, focusedCol: 1}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "/", tcell.ModNone))
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "2", tcell.ModNone))
	if len(app.searchMatches) != 1 || directory.Items[app.searchMatches[0]].Name() != "bravo.txt" {
		t.Fatalf("search by line number matches = %v", app.searchMatches)
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
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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

func TestMultiSelectCopyAndRangeSelection(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one", "two", "three"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	app := App{focusedCol: 1}
	app.current, _ = filesystem.Load(root, root)
	app.itemSelected = 0
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "y", tcell.ModNone))
	app.itemSelected = 1
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "y", tcell.ModCtrl))
	if len(app.clipboardPaths) != 2 {
		t.Fatalf("ctrl+y selection count = %d, want 2", len(app.clipboardPaths))
	}
	app.itemSelected = 2
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, "Y", tcell.ModShift))
	if len(app.clipboardPaths) != 3 {
		t.Fatalf("shift+y selection count = %d, want 3", len(app.clipboardPaths))
	}
}

func TestFooterPaneData(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.txt")
	if err := os.WriteFile(path, []byte("content"), 0o640); err != nil {
		t.Fatal(err)
	}
	directory, err := filesystem.Load(root, root)
	if err != nil {
		t.Fatal(err)
	}
	app := App{current: directory}
	infoLines := app.selectedItemInfo()
	if len(infoLines) != 3 || !strings.Contains(infoLines[1], "0640") {
		t.Fatalf("file info = %q, expected permissions", infoLines)
	}

	app.addLog("touch %s", path)
	logLines := app.operationLogLines()
	if len(logLines) != 1 || logLines[0] != "touch "+path {
		t.Fatalf("operation log = %q", logLines)
	}

	app.clipboardPaths = []string{path}
	app.clipboardCut = true
	clipboardLines := app.clipboardLines()
	if len(clipboardLines) != 1 || !strings.HasPrefix(clipboardLines[0], "CUT  ") {
		t.Fatalf("clipboard lines = %q", clipboardLines)
	}
}

func TestShortcutHintByFocusedModule(t *testing.T) {
	tests := []struct {
		name           string
		focusedCol     int
		collectionPage bool
		contains       []string
	}{
		{
			name:       "path bar",
			focusedCol: -1,
			contains:   []string{"a 输入路径", "q 退出"},
		},
		{
			name:           "collections",
			focusedCol:     0,
			collectionPage: true,
			contains:       []string{"a 新增集合", "] 目录页"},
		},
		{
			name:       "directories",
			focusedCol: 0,
			contains:   []string{"a 新增条目", "e 编辑", "d 删除"},
		},
		{
			name:       "files",
			focusedCol: 1,
			contains:   []string{"a 新增", "d 删除", "y 复制", "x 剪切", "p 粘贴"},
		},
		{
			name:       "preview",
			focusedCol: 2,
			contains:   []string{"j/k 滚动预览"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := App{focusedCol: test.focusedCol, collectionPage: test.collectionPage}
			hint := app.shortcutHint()
			for _, expected := range test.contains {
				if !strings.Contains(hint, expected) {
					t.Errorf("shortcut hint %q does not contain %q", hint, expected)
				}
			}
		})
	}
}

func TestDrawShortcutHintTruncatesToTerminalWidth(t *testing.T) {
	const width = 12
	hint := truncateShortcutHint((&App{focusedCol: 1}).shortcutHint(), width)
	if displaywidth.String(hint) > width {
		t.Fatalf("truncated hint width = %d, want at most %d", displaywidth.String(hint), width)
	}
	if !strings.HasSuffix(hint, "…") {
		t.Fatalf("truncated hint = %q, want ellipsis suffix", hint)
	}

	if got := truncateShortcutHint("a 新增", width); got != "a 新增" {
		t.Fatalf("short hint = %q, want unchanged", got)
	}
}
