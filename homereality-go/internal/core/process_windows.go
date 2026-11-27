//go:build windows

package core

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func hideConsole(cmd *exec.Cmd) {
	// CREATE_NO_WINDOW stops console-subsystem binaries (sing-box) from
	// flashing a black window. HideWindow alone is not enough.
	const CREATE_NO_WINDOW = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: CREATE_NO_WINDOW,
	}
}

// windowsProcessExists checks if a PID is alive without spawning tasklist
// (tasklist was opening a console window every poll tick).
func windowsProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	err = windows.GetExitCodeProcess(h, &code)
	if err != nil {
		return false
	}
	// STILL_ACTIVE = 259
	return code == 259
}
