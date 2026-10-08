//go:build darwin

package filesystem

func openPath(path string) error {
	return startDesktopOpener("open", path)
}
