//go:build linux

package filesystem

func revealPath(folder, _ string) error {
	return openPath(folder)
}
