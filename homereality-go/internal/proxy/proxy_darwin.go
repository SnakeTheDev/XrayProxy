//go:build darwin

package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type macProxy struct {
	dataDir  string
	savePath string
}

type savedMac struct {
	Services map[string]serviceState `json:"services"`
}

type serviceState struct {
	WebEnabled   bool   `json:"web_enabled"`
	WebServer    string `json:"web_server"`
	WebPort      string `json:"web_port"`
	HTTPSEnabled bool   `json:"https_enabled"`
	HTTPSServer  string `json:"https_server"`
	HTTPSPort    string `json:"https_port"`
	SOCKSEnabled bool   `json:"socks_enabled"`
	SOCKSServer  string `json:"socks_server"`
	SOCKSPort    string `json:"socks_port"`
}

func newPlatform(dataDir string) Controller {
	return &macProxy{
		dataDir:  dataDir,
		savePath: filepath.Join(dataDir, "proxy_restore.json"),
	}
}

func networkServices() []string {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil
	}
	var list []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "An asterisk") || strings.HasPrefix(line, "*") {
			continue
		}
		// disabled services start with *
		if strings.HasPrefix(line, "*") {
			continue
		}
		list = append(list, line)
	}
	return list
}

func (m *macProxy) Enable() error {
	services := networkServices()
	saved := savedMac{Services: map[string]serviceState{}}
	for _, svc := range services {
		st := readService(svc)
		saved.Services[svc] = st
	}
	_ = writeJSON(m.savePath, saved)

	host, port := LocalHost, fmt.Sprintf("%d", LocalPort)
	for _, svc := range services {
		_ = exec.Command("networksetup", "-setwebproxy", svc, host, port).Run()
		_ = exec.Command("networksetup", "-setsecurewebproxy", svc, host, port).Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxy", svc, host, port).Run()
		_ = exec.Command("networksetup", "-setwebproxystate", svc, "on").Run()
		_ = exec.Command("networksetup", "-setsecurewebproxystate", svc, "on").Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxystate", svc, "on").Run()
	}
	return nil
}

func (m *macProxy) Disable() error {
	var saved savedMac
	services := networkServices()
	if err := readJSON(m.savePath, &saved); err == nil && len(saved.Services) > 0 {
		for svc, st := range saved.Services {
			restoreService(svc, st)
		}
		_ = os.Remove(m.savePath)
		return nil
	}
	for _, svc := range services {
		_ = exec.Command("networksetup", "-setwebproxystate", svc, "off").Run()
		_ = exec.Command("networksetup", "-setsecurewebproxystate", svc, "off").Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxystate", svc, "off").Run()
	}
	return nil
}

func (m *macProxy) IsOurs() bool {
	for _, svc := range networkServices() {
		out, err := exec.Command("networksetup", "-getwebproxy", svc).Output()
		if err != nil {
			continue
		}
		s := string(out)
		if strings.Contains(s, "Enabled: Yes") && strings.Contains(s, LocalHost) && strings.Contains(s, fmt.Sprintf("%d", LocalPort)) {
			return true
		}
	}
	return false
}

func readService(svc string) serviceState {
	var st serviceState
	parse := func(args []string, en *bool, server, port *string) {
		out, err := exec.Command("networksetup", args...).Output()
		if err != nil {
			return
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Enabled:") {
				*en = strings.Contains(line, "Yes")
			}
			if strings.HasPrefix(line, "Server:") {
				*server = strings.TrimSpace(strings.TrimPrefix(line, "Server:"))
			}
			if strings.HasPrefix(line, "Port:") {
				*port = strings.TrimSpace(strings.TrimPrefix(line, "Port:"))
			}
		}
	}
	parse([]string{"-getwebproxy", svc}, &st.WebEnabled, &st.WebServer, &st.WebPort)
	parse([]string{"-getsecurewebproxy", svc}, &st.HTTPSEnabled, &st.HTTPSServer, &st.HTTPSPort)
	parse([]string{"-getsocksfirewallproxy", svc}, &st.SOCKSEnabled, &st.SOCKSServer, &st.SOCKSPort)
	return st
}

func restoreService(svc string, st serviceState) {
	set := func(kind, server, port string, on bool) {
		state := "off"
		if on && server != "" {
			state = "on"
			switch kind {
			case "web":
				_ = exec.Command("networksetup", "-setwebproxy", svc, server, port).Run()
			case "secure":
				_ = exec.Command("networksetup", "-setsecurewebproxy", svc, server, port).Run()
			case "socks":
				_ = exec.Command("networksetup", "-setsocksfirewallproxy", svc, server, port).Run()
			}
		}
		switch kind {
		case "web":
			_ = exec.Command("networksetup", "-setwebproxystate", svc, state).Run()
		case "secure":
			_ = exec.Command("networksetup", "-setsecurewebproxystate", svc, state).Run()
		case "socks":
			_ = exec.Command("networksetup", "-setsocksfirewallproxystate", svc, state).Run()
		}
	}
	set("web", st.WebServer, st.WebPort, st.WebEnabled)
	set("secure", st.HTTPSServer, st.HTTPSPort, st.HTTPSEnabled)
	set("socks", st.SOCKSServer, st.SOCKSPort, st.SOCKSEnabled)
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
