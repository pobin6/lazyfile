//go:build windows

package filesystem

import (
	"errors"
	"syscall"
	"testing"
)

func TestIsCrossDeviceError(t *testing.T) {
	if !isCrossDeviceError(errorNotSameDevice) {
		t.Fatal("Windows ERROR_NOT_SAME_DEVICE was not recognized")
	}
	if isCrossDeviceError(syscall.Errno(5)) {
		t.Fatal("access denied was incorrectly recognized as a cross-device error")
	}
	if isCrossDeviceError(errors.New("unrelated error")) {
		t.Fatal("unrelated error was incorrectly recognized as a cross-device error")
	}
}
