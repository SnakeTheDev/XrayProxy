//go:build windows

package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type winProxy struct {
	dataDir string
	savePath string
}

type savedWin struct {
	ProxyEnable uint32 `json:"proxy_enable"`
	ProxyServer string `json:"proxy_server"`
	ProxyOverride string `json:"proxy_override"`
}

func newPlatform(dataDir string) Controller {
	return &winProxy{
		dataDir:  dataDir,
		savePath: filepath.Join(dataDir, "proxy_restore.json"),
	}
}

func internetSettings() (registry.Key, error) {
	return registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
}

func (w *winProxy) Enable() error {
	k, err := internetSettings()
	if err != nil {
		return err
	}
	defer k.Close()

	// save current
	var s savedWin
	s.ProxyEnable, _, _ = k.GetIntegerValue("ProxyEnable")
	s.ProxyServer, _, _ = k.GetStringValue("ProxyServer")
	s.ProxyOverride, _, _ = k.GetStringValue("ProxyOverride")
	_ = writeJSON(w.savePath, s)

	server := fmt.Sprintf("%s:%d", LocalHost, LocalPort)
	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return err
	}
	if err := k.SetStringValue("ProxyServer", server); err != nil {
		return err
	}
	// keep local bypass
	ov := s.ProxyOverride
	if ov == "" {
		ov = "localhost;127.*;10.*;192.168.*;<local>"
	}
	_ = k.SetStringValue("ProxyOverride", ov)
	notifyProxyChanged()
	return nil
}

func (w *winProxy) Disable() error {
	k, err := internetSettings()
	if err != nil {
		return err
	}
	defer k.Close()

	var s savedWin
	if err := readJSON(w.savePath, &s); err == nil {
		_ = k.SetDWordValue("ProxyEnable", s.ProxyEnable)
		_ = k.SetStringValue("ProxyServer", s.ProxyServer)
		_ = k.SetStringValue("ProxyOverride", s.ProxyOverride)
		_ = os.Remove(w.savePath)
	} else {
		_ = k.SetDWordValue("ProxyEnable", 0)
	}
	notifyProxyChanged()
	return nil
}

func (w *winProxy) IsOurs() bool {
	k, err := internetSettings()
	if err != nil {
		return false
	}
	defer k.Close()
	en, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil || en == 0 {
		return false
	}
	srv, _, err := k.GetStringValue("ProxyServer")
	if err != nil {
		return false
	}
	want := fmt.Sprintf("%s:%d", LocalHost, LocalPort)
	return strings.Contains(srv, want) || strings.Contains(srv, fmt.Sprintf("http=%s", want))
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

func notifyProxyChanged() {
	// InternetSetOption via syscall would be ideal; registry change is usually enough for most apps after refresh.
}
