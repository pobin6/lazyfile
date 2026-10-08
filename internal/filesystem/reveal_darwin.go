//go:build darwin

package filesystem

func revealPath(folder, selectedPath string) error {
	if selectedPath == "" {
		return openPath(folder)
	}
	return startDesktopOpener("open", "-R", selectedPath)
}
