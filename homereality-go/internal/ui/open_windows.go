//go:build windows

package ui

import (
	"os/exec"
	"syscall"
)

func windowsHideAttr() *syscall.SysProcAttr {
	const CREATE_NO_WINDOW = 0x08000000
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: CREATE_NO_WINDOW,
	}
}

// silence unused import if openPath uses SysProcAttr type only via this helper
var _ = exec.Command
