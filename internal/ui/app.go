package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"lazyfile/internal/config"
	"lazyfile/internal/filesystem"

	"github.com/gdamore/tcell/v3"
)

const maxPreviewBytes = 1024

type App struct {
	screen             tcell.Screen
	current            filesystem.Directory
	baseDir            string
	input              []rune
	inputOpen          bool
	errorMessage       string
	focusedCol         int
	collectionPage     bool
	collections        []config.Collection
	selectedCollection int
	entries            []config.Entry
	selected           int
	inputMode          inputMode
	confirmDelete      bool
	confirmItemDelete  bool
	itemSelected       int
	itemScroll         int
	previewOffset      int
	clipboardPath      string
	clipboardCut       bool
}

type inputMode int

const (
	inputPath inputMode = iota
	inputAdd
	inputEdit
	inputAddCollection
	inputAddItem
)

func New(screen tcell.Screen, initialDir string) (*App, error) {
	state, err := config.Load()
	if err != nil {
		return nil, err
	}
	app := &App{
		screen:             screen,
		baseDir:            initialDir,
		collections:        state.Collections,
		selectedCollection: state.SelectedCollection,
		collectionPage:     state.CollectionPage,
		entries:            state.Entries,
		selected:           state.SelectedIndex,
	}
	if len(app.collections) == 0 && len(app.entries) > 0 {
		app.collections = []config.Collection{{Name: "Default", Entries: app.entries}}
	}
	app.normalizeCollections()
	app.syncCollectionEntries()
	app.normalizeSelection()
	if !app.collectionPage && len(app.entries) > 0 {
		if err := app.loadSelected(); err != nil {
			app.errorMessage = err.Error()
		}
	}
	return app, nil
}

func (a *App) Run() error {
	for {
		a.draw()
		event := <-a.screen.EventQ()
		switch event := event.(type) {
		case *tcell.EventKey:
			if a.handleKey(event) {
				return a.saveState()
			}
		case *tcell.EventResize:
			a.screen.Sync()
		}
	}
}

func (a *App) handleKey(event *tcell.EventKey) bool {
	if a.confirmDelete {
		return a.handleDeleteConfirmation(event)
	}
	if a.confirmItemDelete {
		return a.handleItemDeleteConfirmation(event)
	}
	if a.inputOpen {
		return a.handleInputKey(event)
	}

	switch event.Key() {
	case tcell.KeyCtrlC, tcell.KeyEscape:
		return true
	case tcell.KeyRune:
		switch event.Str() {
		case "q":
			return true
		case "0":
			a.focusedCol = -1
		case "1":
			a.focusedCol = 0
		case "2":
			a.focusedCol = 1
		case "3":
			a.focusedCol = 2
		case "a":
			if a.focusedCol == 0 {
				if a.collectionPage {
					a.openInput(inputAddCollection)
				} else {
					a.openInput(inputAdd)
				}
			} else if a.focusedCol == 1 {
				a.openInput(inputAddItem)
			} else if a.focusedCol == 2 {
				a.openInput(inputPath)
			}
		case "e":
			if a.focusedCol == 0 && !a.collectionPage && len(a.entries) > 0 {
				a.openInput(inputEdit)
			}
		case "d":
			if a.focusedCol == 0 && !a.collectionPage && len(a.entries) > 0 {
				a.confirmDelete = true
			} else if a.focusedCol == 1 && len(a.current.Items) > 0 && a.itemSelected < len(a.current.Items) {
				a.confirmItemDelete = true
			}
		case "j":
			if a.focusedCol == 0 {
				if a.collectionPage {
					a.selectCollection(1)
				} else {
					a.selectEntry(1)
				}
			} else if a.focusedCol == 1 {
				a.selectItem(1)
			} else if a.focusedCol == 2 {
				a.movePreview(1)
			}
		case "k":
			if a.focusedCol == 0 {
				if a.collectionPage {
					a.selectCollection(-1)
				} else {
					a.selectEntry(-1)
				}
			} else if a.focusedCol == 1 {
				a.selectItem(-1)
			} else if a.focusedCol == 2 {
				a.movePreview(-1)
			}
		case "h":
			if a.focusedCol == 1 {
				a.navigateParent()
			}
		case "l":
			if a.focusedCol == 1 {
				a.navigateChild()
			}
		case "y":
			if a.focusedCol == 1 {
				a.yankItem()
			}
		case "x":
			if a.focusedCol == 1 {
				a.cutItem()
			}
		case "p":
			if a.focusedCol == 1 {
				a.pasteItem()
			}
		case "[":
			if a.focusedCol == 0 {
				a.collectionPage = true
				a.syncCollectionEntries()
			}
		case "]":
			if a.focusedCol == 0 {
				a.collectionPage = false
				a.syncCollectionEntries()
				if len(a.entries) > 0 {
					_ = a.loadSelected()
				}
			}
		}
	}
	return false
}

func (a *App) handleDeleteConfirmation(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyEscape:
		a.confirmDelete = false
	case tcell.KeyEnter:
		a.confirmDelete = false
		a.deleteSelected()
	case tcell.KeyRune:
		switch event.Str() {
		case "y":
			a.confirmDelete = false
			a.deleteSelected()
		case "n":
			a.confirmDelete = false
		}
	}
	return false
}

func (a *App) handleItemDeleteConfirmation(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyEscape:
		a.confirmItemDelete = false
	case tcell.KeyEnter:
		a.confirmItemDelete = false
		a.deleteCurrentItem()
	case tcell.KeyRune:
		switch event.Str() {
		case "y":
			a.confirmItemDelete = false
			a.deleteCurrentItem()
		case "n":
			a.confirmItemDelete = false
		}
	}
	return false
}

func (a *App) openInput(mode inputMode) {
	a.inputOpen = true
	a.inputMode = mode
	a.errorMessage = ""
	a.input = nil
	if mode == inputEdit && len(a.entries) > 0 {
		a.input = []rune(a.entries[a.selected].Path)
	}
}

func (a *App) selectEntry(delta int) {
	if len(a.entries) == 0 {
		return
	}
	a.selected += delta
	a.normalizeSelection()
	if err := a.loadSelected(); err != nil {
		a.errorMessage = err.Error()
	}
	if err := a.saveState(); err != nil {
		a.errorMessage = err.Error()
	}
}

func (a *App) normalizeCollections() {
	if a.selectedCollection < 0 {
		a.selectedCollection = 0
	}
	if len(a.collections) == 0 {
		a.selectedCollection = 0
		return
	}
	if a.selectedCollection >= len(a.collections) {
		a.selectedCollection = len(a.collections) - 1
	}
}

func (a *App) syncCollectionEntries() {
	a.normalizeCollections()
	if len(a.collections) == 0 {
		a.entries = nil
		a.selected = 0
		a.current = filesystem.Directory{}
		return
	}
	a.entries = a.collections[a.selectedCollection].Entries
	a.normalizeSelection()
	if a.collectionPage {
		a.current = filesystem.Directory{}
	}
}

func (a *App) selectCollection(delta int) {
	if len(a.collections) == 0 {
		return
	}
	a.selectedCollection += delta
	a.normalizeCollections()
	a.syncCollectionEntries()
	if !a.collectionPage && len(a.entries) > 0 {
		if err := a.loadSelected(); err != nil {
			a.errorMessage = err.Error()
		}
	}
	if err := a.saveState(); err != nil {
		a.errorMessage = err.Error()
	}
}

func (a *App) normalizeSelection() {
	if len(a.entries) == 0 {
		a.selected = 0
		return
	}
	if a.selected < 0 {
		a.selected = 0
	}
	if a.selected >= len(a.entries) {
		a.selected = len(a.entries) - 1
	}
}

func (a *App) loadSelected() error {
	entry := a.entries[a.selected]
	path := entry.Path
	if entry.CurrentPath != "" {
		path = entry.CurrentPath
	}
	directory, err := filesystem.Load(path, a.baseDir)
	if err != nil {
		return err
	}
	a.current = directory
	a.itemSelected = 0
	a.itemScroll = 0
	a.previewOffset = 0
	return nil
}

func (a *App) navigateChild() {
	if len(a.current.Items) == 0 || a.itemSelected >= len(a.current.Items) {
		return
	}
	item := a.current.Items[a.itemSelected]
	if !item.IsDir() {
		return
	}
	a.navigateTo(filepath.Join(a.current.Path, item.Name()))
}

func (a *App) navigateParent() {
	if len(a.entries) == 0 || a.selected >= len(a.entries) {
		return
	}
	rootPath, err := filepath.Abs(a.entries[a.selected].Path)
	if err != nil || filepath.Clean(a.current.Path) == filepath.Clean(rootPath) {
		return
	}
	parent := filepath.Dir(a.current.Path)
	if !isWithinRoot(rootPath, parent) {
		return
	}
	returnedItem := filepath.Base(a.current.Path)
	a.navigateTo(parent)
	for index, item := range a.current.Items {
		if item.Name() == returnedItem {
			a.itemSelected = index
			break
		}
	}
	a.previewOffset = 0
	if err := a.saveState(); err != nil {
		a.errorMessage = err.Error()
	}
}

func isWithinRoot(rootPath, path string) bool {
	relative, err := filepath.Rel(rootPath, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func (a *App) navigateTo(path string) {
	directory, err := filesystem.Load(path, a.baseDir)
	if err != nil {
		a.errorMessage = err.Error()
		return
	}
	a.current = directory
	a.entries[a.selected].CurrentPath = directory.Path
	a.itemSelected = 0
	a.itemScroll = 0
	a.previewOffset = 0
	a.errorMessage = ""
	if err := a.saveState(); err != nil {
		a.errorMessage = err.Error()
	}
}

func (a *App) yankItem() {
	if len(a.current.Items) == 0 || a.itemSelected >= len(a.current.Items) {
		return
	}
	a.clipboardPath = filepath.Join(a.current.Path, a.current.Items[a.itemSelected].Name())
	a.clipboardCut = false
	a.errorMessage = "Selected: " + filepath.Base(a.clipboardPath)
}

func (a *App) cutItem() {
	if len(a.current.Items) == 0 || a.itemSelected >= len(a.current.Items) {
		return
	}
	a.clipboardPath = filepath.Join(a.current.Path, a.current.Items[a.itemSelected].Name())
	a.clipboardCut = true
	a.errorMessage = "Cut: " + filepath.Base(a.clipboardPath)
}

func (a *App) pasteItem() {
	if a.clipboardPath == "" {
		a.errorMessage = "No item selected"
		return
	}
	source := filepath.Clean(a.clipboardPath)
	destination := filepath.Join(a.current.Path, filepath.Base(source))
	if source == destination {
		a.errorMessage = "Cannot paste an item onto itself"
		return
	}
	if _, err := os.Stat(destination); err == nil {
		a.errorMessage = "Destination already exists: " + filepath.Base(destination)
		return
	} else if !os.IsNotExist(err) {
		a.errorMessage = fmt.Errorf("check destination: %w", err).Error()
		return
	}
	if info, err := os.Stat(source); err != nil {
		a.errorMessage = fmt.Errorf("stat selected item: %w", err).Error()
		return
	} else if info.IsDir() {
		if isWithinRoot(source, destination) {
			a.errorMessage = "Cannot paste a directory into itself"
			return
		}
	}
	var err error
	if a.clipboardCut {
		err = os.Rename(source, destination)
	} else {
		err = copyItem(source, destination)
	}
	if err != nil {
		a.errorMessage = fmt.Errorf("paste item: %w", err).Error()
		return
	}
	if a.clipboardCut {
		a.clipboardPath = ""
		a.clipboardCut = false
	}
	selectedName := filepath.Base(destination)
	if err := a.refreshCurrent(selectedName); err != nil {
		a.errorMessage = err.Error()
		return
	}
	a.errorMessage = ""
}

func copyItem(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.Mkdir(destination, info.Mode().Perm()); err != nil {
			return err
		}
		children, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, child := range children {
			if err := copyItem(filepath.Join(source, child.Name()), filepath.Join(destination, child.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, info.Mode().Perm())
}

func (a *App) refreshCurrent(selectedName string) error {
	directory, err := filesystem.Load(a.current.Path, a.baseDir)
	if err != nil {
		return err
	}
	a.current = directory
	a.itemSelected = 0
	for index, item := range directory.Items {
		if item.Name() == selectedName {
			a.itemSelected = index
			break
		}
	}
	a.itemScroll = 0
	a.previewOffset = 0
	return a.saveState()
}

func (a *App) selectItem(delta int) {
	if len(a.current.Items) == 0 {
		return
	}
	a.itemSelected += delta
	if a.itemSelected < 0 {
		a.itemSelected = 0
	}
	if a.itemSelected >= len(a.current.Items) {
		a.itemSelected = len(a.current.Items) - 1
	}
	a.previewOffset = 0
}

func (a *App) ensureItemVisible(visibleRows int) {
	if len(a.current.Items) == 0 || visibleRows <= 0 {
		return
	}
	maxOffset := len(a.current.Items) - visibleRows
	if maxOffset < 0 {
		maxOffset = 0
	}

	topMargin := 3
	bottomMargin := 3
	if visibleRows <= topMargin+bottomMargin {
		topMargin = 0
		bottomMargin = 0
	}

	if a.itemSelected < a.itemScroll+topMargin {
		a.itemScroll = a.itemSelected - topMargin
	}
	if a.itemSelected > a.itemScroll+visibleRows-1-bottomMargin {
		a.itemScroll = a.itemSelected - (visibleRows - 1 - bottomMargin)
	}
	if a.itemScroll < 0 {
		a.itemScroll = 0
	}
	if a.itemScroll > maxOffset {
		a.itemScroll = maxOffset
	}
}

func (a *App) movePreview(delta int) {
	a.previewOffset += delta
	if a.previewOffset < 0 {
		a.previewOffset = 0
	}
}

func (a *App) deleteSelected() {
	if len(a.entries) == 0 {
		return
	}
	a.entries = append(a.entries[:a.selected], a.entries[a.selected+1:]...)
	if len(a.collections) > 0 {
		a.collections[a.selectedCollection].Entries = a.entries
	}
	a.normalizeSelection()
	a.current = filesystem.Directory{}
	a.errorMessage = ""
	if len(a.entries) > 0 {
		if err := a.loadSelected(); err != nil {
			a.errorMessage = err.Error()
		}
	}

	if err := a.saveState(); err != nil {
		a.errorMessage = err.Error()
	}
}

func (a *App) saveState() error {
	if len(a.collections) > 0 {
		a.collections[a.selectedCollection].Entries = a.entries
	}

	return config.Save(config.State{
		Entries:            a.entries,
		SelectedIndex:      a.selected,
		Collections:        a.collections,
		SelectedCollection: a.selectedCollection,
		CollectionPage:     a.collectionPage,
	})
}

func (a *App) deleteCurrentItem() {
	if len(a.current.Items) == 0 || a.itemSelected >= len(a.current.Items) {
		return
	}
	name := a.current.Items[a.itemSelected].Name()
	if err := os.RemoveAll(filepath.Join(a.current.Path, name)); err != nil {
		a.errorMessage = fmt.Errorf("delete item: %w", err).Error()
		return
	}
	if err := a.refreshCurrent(""); err != nil {
		a.errorMessage = err.Error()
		return
	}
	a.errorMessage = ""
}

func (a *App) handleInputKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyEscape:
		a.inputOpen = false
	case tcell.KeyEnter:
		a.submitPath()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(a.input) > 0 {
			a.input = a.input[:len(a.input)-1]
		}
	case tcell.KeyRune:
		a.input = append(a.input, []rune(event.Str())...)
	}
	return false
}

func (a *App) submitPath() {
	if a.inputMode == inputAddCollection {
		name := strings.TrimSpace(string(a.input))
		if name == "" {
			a.errorMessage = "collection name cannot be empty"
			return
		}
		a.collections = append(a.collections, config.Collection{Name: name})
		a.selectedCollection = len(a.collections) - 1
		a.entries = nil
		a.selected = 0
		a.current = filesystem.Directory{}
		a.collectionPage = true
		a.inputOpen = false
		a.input = nil
		a.errorMessage = ""
		if err := a.saveState(); err != nil {
			a.errorMessage = err.Error()
		}
		return
	}
	if a.inputMode == inputAddItem {
		a.createItem(strings.TrimSpace(string(a.input)))
		return
	}
	directory, err := filesystem.Load(string(a.input), a.baseDir)
	if err != nil {
		a.errorMessage = err.Error()
		return
	}
	switch a.inputMode {
	case inputAdd:
		a.entries = append(a.entries, config.Entry{
			Path:        directory.Path,
			Name:        directory.Name,
			CurrentPath: directory.Path,
		})
		a.selected = len(a.entries) - 1
	case inputEdit:
		a.entries[a.selected].Path = directory.Path
		a.entries[a.selected].Name = directory.Name
		a.entries[a.selected].CurrentPath = directory.Path
	}
	if len(a.collections) > 0 {
		a.collections[a.selectedCollection].Entries = a.entries
	}
	a.current = directory
	a.itemSelected = 0
	a.itemScroll = 0
	a.previewOffset = 0
	a.inputOpen = false
	a.input = nil
	a.errorMessage = ""
	if err := a.saveState(); err != nil {
		a.errorMessage = err.Error()
	}
}

func (a *App) createItem(name string) {
	if name == "" {
		a.errorMessage = "item name cannot be empty"
		return
	}
	relative := filepath.FromSlash(name)
	if filepath.IsAbs(relative) || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		a.errorMessage = "item name must stay inside the current directory"
		return
	}
	destination := filepath.Join(a.current.Path, relative)
	if _, err := os.Stat(destination); err == nil {
		a.errorMessage = "item already exists: " + name
		return
	} else if !os.IsNotExist(err) {
		a.errorMessage = fmt.Errorf("check item: %w", err).Error()
		return
	}
	var err error
	if strings.Contains(name, "/") {
		err = os.MkdirAll(destination, 0o755)
	} else {
		err = os.WriteFile(destination, nil, 0o644)
	}
	if err != nil {
		a.errorMessage = fmt.Errorf("create item: %w", err).Error()
		return
	}
	if err := a.refreshCurrent(filepath.Base(destination)); err != nil {
		a.errorMessage = err.Error()
		return
	}
	a.inputOpen = false
	a.input = nil
	a.errorMessage = ""
}

func (a *App) draw() {
	a.screen.Clear()
	width, height := a.screen.Size()
	layout := calculateLayout(width)
	a.drawPathBar(width)

	a.drawColumnBorders(layout, height)

	firstColumnEntries := a.entries
	if a.collectionPage {
		firstColumnEntries = nil
	}
	for index, entry := range firstColumnEntries {
		if index >= height-3 {
			break
		}
		style := tcell.StyleDefault
		if a.focusedCol == 0 && index == a.selected {
			style = style.Reverse(true)
		}
		a.drawStyledText(1, index+4, layout.LeftWidth-2, entry.Name, style)
	}
	if a.collectionPage {
		for index, collection := range a.collections {
			if index >= height-3 {
				break
			}
			style := tcell.StyleDefault
			if a.focusedCol == 0 && index == a.selectedCollection {
				style = style.Reverse(true)
			}
			a.drawStyledText(1, index+4, layout.LeftWidth-2, collection.Name, style)
		}
	}
	visibleItemRows := height - 5
	a.ensureItemVisible(visibleItemRows)
	for index := a.itemScroll; index < len(a.current.Items); index++ {
		if index-a.itemScroll >= visibleItemRows {
			break
		}
		item := a.current.Items[index]
		style := tcell.StyleDefault
		if a.focusedCol == 1 && index == a.itemSelected {
			style = style.Reverse(true)
		}
		a.drawStyledText(layout.LeftWidth+1, index-a.itemScroll+4, layout.MiddleWidth-2, item.Name(), style)
	}
	a.drawPreview(layout, height)

	if a.errorMessage != "" {
		a.drawText(1, height-2, width-2, "Error: "+a.errorMessage)
	}

	if a.inputOpen {
		a.drawInputDialog(width, height)
	}
	if a.confirmDelete {
		a.drawDeleteConfirmation(width, height)
	}
	if a.confirmItemDelete {
		a.drawItemDeleteConfirmation(width, height)
	}
	a.screen.Show()
}

func (a *App) drawPreview(layout Layout, height int) {
	lines := a.previewLines()
	offset := a.previewOffset
	maxOffset := len(lines) - (height - 5)
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	for index := offset; index < len(lines) && index-offset < height-5; index++ {
		a.drawText(layout.LeftWidth+layout.MiddleWidth+1, index-offset+4,
			layout.RightWidth-2, lines[index])
	}
}

func (a *App) drawPathBar(width int) {
	if width <= 0 {
		return
	}
	style := tcell.StyleDefault
	if a.focusedCol == -1 {
		style = style.Foreground(tcell.ColorPurple).Bold(true)
	}
	for column := 0; column < width; column++ {
		a.screen.SetContent(column, 0, '─', nil, style)
		a.screen.SetContent(column, 2, '─', nil, style)
	}
	a.screen.SetContent(0, 0, '┌', nil, style)
	a.screen.SetContent(width-1, 0, '┐', nil, style)
	a.screen.SetContent(0, 2, '└', nil, style)
	a.screen.SetContent(width-1, 2, '┘', nil, style)
	a.drawStyledText(2, 1, width-4, a.selectedPath(), style)
}

func (a *App) selectedPath() string {
	if len(a.current.Items) > 0 && a.itemSelected < len(a.current.Items) {
		return filepath.Join(a.current.Path, a.current.Items[a.itemSelected].Name())
	}
	if a.current.Path != "" {
		return a.current.Path
	}
	return "No directory selected"
}

func (a *App) previewLines() []string {
	if len(a.current.Items) == 0 || a.itemSelected >= len(a.current.Items) {
		return nil
	}
	item := a.current.Items[a.itemSelected]
	path := filepath.Join(a.current.Path, item.Name())
	if item.IsDir() {
		children, err := os.ReadDir(path)
		if err != nil {
			return []string{"Preview error: " + err.Error()}
		}
		lines := make([]string, 0, len(children)+1)
		lines = append(lines, "Directory: "+item.Name())
		for _, child := range children {
			lines = append(lines, child.Name())
		}
		return lines
	}
	file, err := os.Open(path)
	if err != nil {
		return []string{"Preview error: " + err.Error()}
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxPreviewBytes+1))
	if err != nil {
		return []string{"Preview error: " + err.Error()}
	}
	truncated := len(data) > maxPreviewBytes
	if truncated {
		data = data[:maxPreviewBytes]
	}
	lines := append([]string{"File: " + item.Name()}, strings.Split(string(data), "\n")...)
	if truncated {
		lines = append(lines, "[Preview truncated after 1 KiB]")
	}
	return lines
}

func (a *App) drawColumnBorders(layout Layout, height int) {
	if height <= 3 {
		return
	}
	borders := []struct {
		x       int
		width   int
		focused bool
	}{
		{x: 0, width: layout.LeftWidth, focused: a.focusedCol == 0},
		{x: layout.LeftWidth, width: layout.MiddleWidth, focused: a.focusedCol == 1},
		{x: layout.LeftWidth + layout.MiddleWidth, width: layout.RightWidth, focused: a.focusedCol == 2},
	}
	for _, border := range borders {
		style := tcell.StyleDefault
		horizontal := '─'
		vertical := '│'
		topLeft := '┌'
		topRight := '┐'
		bottomLeft := '└'
		bottomRight := '┘'
		if border.focused {
			style = style.Foreground(tcell.ColorPurple).Bold(true)
			horizontal = '━'
			vertical = '┃'
			topLeft = '┏'
			topRight = '┓'
			bottomLeft = '┗'
			bottomRight = '┛'
		}
		right := border.x + border.width - 1
		if border.width < 2 {
			continue
		}
		for column := border.x; column <= right; column++ {
			a.screen.SetContent(column, 3, horizontal, nil, style)
			a.screen.SetContent(column, height-1, horizontal, nil, style)
		}
		for row := 4; row < height-1; row++ {
			a.screen.SetContent(border.x, row, vertical, nil, style)
			a.screen.SetContent(right, row, vertical, nil, style)
		}
		a.screen.SetContent(border.x, 3, topLeft, nil, style)
		a.screen.SetContent(right, 3, topRight, nil, style)
		a.screen.SetContent(border.x, height-1, bottomLeft, nil, style)
		a.screen.SetContent(right, height-1, bottomRight, nil, style)
	}

	titles := []string{"Directories", "Files", "Preview"}
	for index, border := range borders {
		if border.width < 4 {
			continue
		}
		style := tcell.StyleDefault
		if border.focused {
			style = style.Foreground(tcell.ColorPurple).Bold(true)
		}
		a.drawStyledText(border.x+2, 3, border.width-4, titles[index], style)
	}
}

func (a *App) drawInputDialog(width, height int) {
	dialogWidth := min(70, max(30, width-4))
	dialogHeight := 5
	x := (width - dialogWidth) / 2
	y := (height - dialogHeight) / 2

	for row := 0; row < dialogHeight; row++ {
		for column := 0; column < dialogWidth; column++ {
			char := ' '
			if row == 0 || row == dialogHeight-1 {
				char = '─'
			} else if column == 0 || column == dialogWidth-1 {
				char = '│'
			}
			if (row == 0 || row == dialogHeight-1) && (column == 0 || column == dialogWidth-1) {
				char = '┼'
			}
			a.screen.SetContent(x+column, y+row, char, nil, tcell.StyleDefault)
		}
	}
	title := "Directory path (Enter submit, Esc cancel)"
	if a.inputMode == inputAdd {
		title = "Add directory (Enter submit, Esc cancel)"
	} else if a.inputMode == inputEdit {
		title = "Edit directory (Enter submit, Esc cancel)"
	} else if a.inputMode == inputAddCollection {
		title = "Add collection (Enter submit, Esc cancel)"
	} else if a.inputMode == inputAddItem {
		title = "Add file or directory (Enter submit, Esc cancel)"
	}
	a.drawText(x+2, y+1, dialogWidth-4, title)
	a.drawText(x+2, y+2, dialogWidth-4, string(a.input))
}

func (a *App) drawDeleteConfirmation(width, height int) {
	dialogWidth := min(70, max(34, width-4))
	dialogHeight := 5
	x := (width - dialogWidth) / 2
	y := (height - dialogHeight) / 2

	for row := 0; row < dialogHeight; row++ {
		for column := 0; column < dialogWidth; column++ {
			char := ' '
			if row == 0 || row == dialogHeight-1 {
				char = '─'
			} else if column == 0 || column == dialogWidth-1 {
				char = '│'
			}

			if (row == 0 || row == dialogHeight-1) && (column == 0 || column == dialogWidth-1) {
				char = '┼'
			}
			a.screen.SetContent(x+column, y+row, char, nil, tcell.StyleDefault)
		}
	}
	name := a.entries[a.selected].Name
	a.drawText(x+2, y+1, dialogWidth-4, "Delete "+name+"? (Enter/y confirm, Esc/n cancel)")
}

func (a *App) drawItemDeleteConfirmation(width, height int) {
	dialogWidth := min(70, max(34, width-4))
	dialogHeight := 5
	x := (width - dialogWidth) / 2
	y := (height - dialogHeight) / 2
	for row := 0; row < dialogHeight; row++ {
		for column := 0; column < dialogWidth; column++ {
			char := ' '
			if row == 0 || row == dialogHeight-1 {
				char = '─'
			} else if column == 0 || column == dialogWidth-1 {
				char = '│'
			}
			if (row == 0 || row == dialogHeight-1) && (column == 0 || column == dialogWidth-1) {
				char = '┼'
			}
			a.screen.SetContent(x+column, y+row, char, nil, tcell.StyleDefault)
		}
	}
	name := a.current.Items[a.itemSelected].Name()
	a.drawText(x+2, y+1, dialogWidth-4, "Delete "+name+"? (Enter/y confirm, Esc/n cancel)")
}

func (a *App) drawText(x, y, width int, value string) {
	a.drawStyledText(x, y, width, value, tcell.StyleDefault)
}

func (a *App) drawStyledText(x, y, width int, value string, style tcell.Style) {
	if width <= 0 || y < 0 {
		return
	}
	value = strings.TrimSpace(value)
	cursor := x
	end := x + width
	for _, character := range []rune(value) {
		if cursor >= end {
			break
		}
		_, cellWidth := a.screen.Put(cursor, y, string(character), style)
		if cellWidth <= 0 {
			continue
		}
		if cursor+cellWidth > end {
			a.screen.SetContent(cursor, y, ' ', nil, style)
			break
		}
		cursor += cellWidth
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func currentDirectory() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	return filepath.Abs(directory)
}
