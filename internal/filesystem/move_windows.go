//go:build windows

package filesystem

import (
	"errors"
	"syscall"
)

const errorNotSameDevice syscall.Errno = 17

func isCrossDeviceError(err error) bool {
	return errors.Is(err, errorNotSameDevice)
}
