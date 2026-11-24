//go:build !windows

package core

import "os/exec"

func hideConsole(cmd *exec.Cmd) {}

func windowsProcessExists(pid int) bool { return false }
