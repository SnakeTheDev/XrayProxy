//go:build !windows

package ui

import "syscall"

func windowsHideAttr() *syscall.SysProcAttr {
	return nil
}
