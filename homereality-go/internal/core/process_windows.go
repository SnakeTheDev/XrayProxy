//go:build windows

package core

import (
	"os/exec"
	"strconv"
	"syscall"
)

func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func windowsProcessExists(pid int) bool {
	// tasklist is slow; OpenProcess via taskkill / query
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").CombinedOutput()
	if err != nil {
		return false
	}
	return len(out) > 0 && !containsNoTasks(string(out))
}

func containsNoTasks(s string) bool {
	return len(s) < 10 || // empty-ish
		(len(s) > 0 && (containsFold(s, "No tasks") || containsFold(s, "INFO:")))
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && indexFold(s, sub) >= 0))
}

func indexFold(s, sub string) int {
	// simple ASCII fold
	ls, lsub := toLower(s), toLower(sub)
	for i := 0; i+len(lsub) <= len(ls); i++ {
		if ls[i:i+len(lsub)] == lsub {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}
