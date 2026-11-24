//go:build linux

package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type linuxProxy struct {
	dataDir  string
	savePath string
}

type savedLinux struct {
	Mode   string `json:"mode"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	HadGsettings bool `json:"had_gsettings"`
}

func newPlatform(dataDir string) Controller {
	return &linuxProxy{
		dataDir:  dataDir,
		savePath: filepath.Join(dataDir, "proxy_restore.json"),
	}
}

func (l *linuxProxy) Enable() error {
	s := savedLinux{Port: LocalPort, Host: LocalHost}
	if _, err := exec.LookPath("gsettings"); err == nil {
		s.HadGsettings = true
		s.Mode = gsettingsGet("org.gnome.system.proxy", "mode")
		_ = writeJSON(l.savePath, s)
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", LocalHost).Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", fmt.Sprintf("%d", LocalPort)).Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", LocalHost).Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", fmt.Sprintf("%d", LocalPort)).Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", LocalHost).Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", fmt.Sprintf("%d", LocalPort)).Run()
		return nil
	}
	// fallback: write env file hint
	_ = writeJSON(l.savePath, s)
	return fmt.Errorf("gsettings not found — set HTTP_PROXY manually to http://%s:%d", LocalHost, LocalPort)
}

func (l *linuxProxy) Disable() error {
	var s savedLinux
	_ = readJSON(l.savePath, &s)
	if s.HadGsettings {
		mode := s.Mode
		if mode == "" {
			mode = "none"
		}
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", mode).Run()
	}
	_ = os.Remove(l.savePath)
	return nil
}

func (l *linuxProxy) IsOurs() bool {
	if _, err := exec.LookPath("gsettings"); err != nil {
		return false
	}
	mode := gsettingsGet("org.gnome.system.proxy", "mode")
	if mode != "manual" {
		return false
	}
	host := gsettingsGet("org.gnome.system.proxy.http", "host")
	return host == LocalHost
}

func gsettingsGet(schema, key string) string {
	out, err := exec.Command("gsettings", "get", schema, key).Output()
	if err != nil {
		return ""
	}
	s := string(out)
	// strip quotes
	s = stringsTrim(s)
	return s
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\'' || s[0] == '"') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n' || s[len(s)-1] == '\'' || s[len(s)-1] == '"') {
		s = s[:len(s)-1]
	}
	return s
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
