package filesystem

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Copy(source, destination string) error {
	return copyItem(source, destination)
}

func RemoveAll(path string) error {
	return removeAll(path)
}

func Move(source, destination string) error {
	err := os.Rename(source, destination)
	if err == nil {
		return nil
	}
	if !isCrossDeviceError(err) {
		return err
	}
	return moveAcrossDevices(source, destination)
}

func Open(path string) error {
	return openPath(path)
}

func startDesktopOpener(name, path string) error {
	command := exec.Command(name, path)
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

func moveAcrossDevices(source, destination string) error {
	if err := copyItem(source, destination); err != nil {
		if cleanupErr := removeAll(destination); cleanupErr != nil {
			return fmt.Errorf("copy across volumes: %w (remove partial destination: %v)", err, cleanupErr)
		}
		return fmt.Errorf("copy across volumes: %w", err)
	}
	if err := removeAll(source); err != nil {
		return fmt.Errorf("remove source after cross-volume copy: %w", err)
	}
	return nil
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
