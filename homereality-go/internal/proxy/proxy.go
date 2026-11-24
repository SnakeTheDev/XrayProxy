package proxy

const LocalHost = "127.0.0.1"
const LocalPort = 2080

// Controller sets/clears the OS system HTTP+HTTPS(+SOCKS where supported) proxy.
type Controller interface {
	// Enable points system proxy at 127.0.0.1:2080 and saves prior settings.
	Enable() error
	// Disable restores prior settings (or clears if none saved).
	Disable() error
	// IsOurs reports whether the system proxy currently points at our local port.
	IsOurs() bool
}

func New(dataDir string) Controller {
	return newPlatform(dataDir)
}
