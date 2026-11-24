package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Manager struct {
	Dir        string
	ConfigPath string
	LogPath    string
	mu         sync.Mutex
	cmd        *exec.Cmd
	pid        int
	started    time.Time
}

func New(dir string) *Manager {
	return &Manager{
		Dir:        dir,
		ConfigPath: filepath.Join(dir, "sing-box-config.json"),
		LogPath:    filepath.Join(dir, "proxy_core.log"),
	}
}

func (m *Manager) BinaryPath() string {
	name := "sing-box"
	if runtime.GOOS == "windows" {
		name = "sing-box.exe"
	}
	return filepath.Join(m.Dir, name)
}

func (m *Manager) HasBinary() bool {
	st, err := os.Stat(m.BinaryPath())
	return err == nil && st.Size() > 1000
}

func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil {
		// poll: on Unix Signal(0); on Windows check via FindProcess is weak — track pid file
		if processAlive(m.pid) {
			return true
		}
		m.cmd = nil
		m.pid = 0
	}
	// recover from pid file if app restarted
	if pid := readPID(m.pidFile()); pid > 0 && processAlive(pid) {
		m.pid = pid
		return true
	}
	return false
}

func (m *Manager) StartedAt() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pid > 0 && processAlive(m.pid) {
		return nil
	}
	bin := m.BinaryPath()
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("sing-box not found — wait for download or place it in the app folder")
	}
	if _, err := os.Stat(m.ConfigPath); err != nil {
		return fmt.Errorf("config missing: %s", m.ConfigPath)
	}
	logF, err := os.OpenFile(m.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "run", "-c", m.ConfigPath)
	cmd.Stdout = logF
	cmd.Stderr = logF
	hideConsole(cmd)
	if err := cmd.Start(); err != nil {
		logF.Close()
		return err
	}
	m.cmd = cmd
	m.pid = cmd.Process.Pid
	m.started = time.Now()
	_ = os.WriteFile(m.pidFile(), []byte(strconv.Itoa(m.pid)), 0o644)
	go func() {
		_ = cmd.Wait()
		logF.Close()
		m.mu.Lock()
		if m.cmd == cmd {
			m.cmd = nil
			m.pid = 0
			_ = os.Remove(m.pidFile())
		}
		m.mu.Unlock()
	}()
	// brief settle
	time.Sleep(400 * time.Millisecond)
	if !processAlive(m.pid) {
		return fmt.Errorf("sing-box exited immediately — check proxy_core.log")
	}
	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	pid := m.pid
	cmd := m.cmd
	m.cmd = nil
	m.pid = 0
	m.mu.Unlock()
	_ = os.Remove(m.pidFile())
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		return
	}
	if pid > 0 {
		killPID(pid)
	}
}

func (m *Manager) pidFile() string {
	return filepath.Join(m.Dir, "homereality.pid")
}

func readPID(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return n
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		// FindProcess always succeeds on Windows; use a zero-signal style check via tasklist is heavy.
		// Try Wait with timeout 0 via open handle — simpler: kill with signal 0 doesn't exist.
		// Use syscall duplicate: if we started it we track cmd; for pid file recovery use tasklist-free approach.
		return windowsProcessExists(pid)
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil
}

func killPID(pid int) {
	p, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = p.Kill()
}
