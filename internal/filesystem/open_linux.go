//go:build linux

package filesystem

func openPath(path string) error {
	return startDesktopOpener("xdg-open", path)
}
