//go:build !windows

package ui

func isAutostartEnabled() bool { return false }

func setAutostart(enable bool) error {
	_ = enable
	return nil
}
