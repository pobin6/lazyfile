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
	itemSelected       int
	itemScroll         int
	previewOffset      int
}

type inputMode int

const (
	inputPath inputMode = iota
	inputAdd
	inputEdit
	inputAddCollection
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
		case "a":
			if a.focusedCol == 0 {
				if a.collectionPage {
					a.openInput(inputAddCollection)
				} else {
					a.openInput(inputAdd)
				}
			} else {
				a.openInput(inputPath)
			}
		case "e":
			if a.focusedCol == 0 && !a.collectionPage && len(a.entries) > 0 {
				a.openInput(inputEdit)
			}
		case "d":
			if a.focusedCol == 0 && !a.collectionPage && len(a.entries) > 0 {
				a.confirmDelete = true
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
			} else {
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
			} else {
				a.movePreview(-1)
			}
		case "h":
			a.moveFocus(-1)
		case "l":
			a.moveFocus(1)
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
	case tcell.KeyLeft:
		a.moveFocus(-1)
	case tcell.KeyRight:
		a.moveFocus(1)
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
	directory, err := filesystem.Load(a.entries[a.selected].Path, a.baseDir)
	if err != nil {
		return err
	}
	a.current = directory
	a.itemSelected = 0
	a.itemScroll = 0
	a.previewOffset = 0
	return nil
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

func (a *App) moveFocus(delta int) {
	a.focusedCol += delta
	if a.focusedCol < 0 {
		a.focusedCol = 0
	}
	if a.focusedCol > 2 {
		a.focusedCol = 2
	}
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
	directory, err := filesystem.Load(string(a.input), a.baseDir)
	if err != nil {
		a.errorMessage = err.Error()
		return
	}
	switch a.inputMode {
	case inputAdd:
		a.entries = append(a.entries, config.Entry{Path: directory.Path, Name: directory.Name})
		a.selected = len(a.entries) - 1
	case inputEdit:
		a.entries[a.selected] = config.Entry{Path: directory.Path, Name: directory.Name}
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

func (a *App) draw() {
	a.screen.Clear()
	width, height := a.screen.Size()
	layout := calculateLayout(width)
	a.drawPathBar(width)

	a.drawColumn(0, layout.LeftWidth, height, "Directories")
	a.drawColumn(layout.LeftWidth, layout.MiddleWidth, height, "")
	a.drawColumn(layout.LeftWidth+layout.MiddleWidth, layout.RightWidth, height, "")
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
	for column := 0; column < width; column++ {
		a.screen.SetContent(column, 0, '─', nil, style)
		a.screen.SetContent(column, 2, '─', nil, style)
	}
	a.screen.SetContent(0, 0, '┌', nil, style)
	a.screen.SetContent(width-1, 0, '┐', nil, style)
	a.screen.SetContent(0, 2, '└', nil, style)
	a.screen.SetContent(width-1, 2, '┘', nil, style)
	a.drawText(2, 1, width-4, a.selectedPath())
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

func (a *App) drawColumn(x, width, height int, title string) {
	if width <= 0 {
		return
	}
	a.drawText(x+1, 4, width-2, title)
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
		if border.focused {
			style = style.Foreground(tcell.ColorPurple)
		}
		right := border.x + border.width - 1
		if border.width < 2 {
			continue
		}
		for column := border.x; column <= right; column++ {
			a.screen.SetContent(column, 3, '─', nil, style)
			a.screen.SetContent(column, height-1, '─', nil, style)
		}
		for row := 4; row < height-1; row++ {
			a.screen.SetContent(border.x, row, '│', nil, style)
			a.screen.SetContent(right, row, '│', nil, style)
		}
		a.screen.SetContent(border.x, 3, '┌', nil, style)
		a.screen.SetContent(right, 3, '┐', nil, style)
		a.screen.SetContent(border.x, height-1, '└', nil, style)
		a.screen.SetContent(right, height-1, '┘', nil, style)
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
