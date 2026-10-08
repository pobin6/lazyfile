//go:build !windows

package filesystem

import (
	"errors"
	"syscall"
)

func isCrossDeviceError(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}
