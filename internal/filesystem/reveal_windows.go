//go:build windows

package filesystem

func revealPath(folder, selectedPath string) error {
	if selectedPath == "" {
		return openPath(folder)
	}
	return startDesktopOpener("explorer.exe", "/select,"+selectedPath)
}
