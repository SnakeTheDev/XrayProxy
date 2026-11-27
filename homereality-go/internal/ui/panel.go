package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/SnakeTheDev/XrayProxy/homereality-go/internal/core"
	"github.com/SnakeTheDev/XrayProxy/homereality-go/internal/download"
	"github.com/SnakeTheDev/XrayProxy/homereality-go/internal/proxy"
	"github.com/SnakeTheDev/XrayProxy/homereality-go/internal/update"
)

const (
	AppName    = "Home-Reality"
	ServerHost = "play.rowstonsoftware.com"
	ServerPort = 443
	LocalProxy = "127.0.0.1:2080"
	ShareURI   = "vless://a750d63d-3348-4f8b-be6a-be17606aff60@play.rowstonsoftware.com:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=dl.google.com&fp=chrome&pbk=npglY0nkJZ16gJ-AmWkt30hf1zgikjNRN9BhPUOKVnQ&sid=0123456789abcdef&type=tcp#Home-Reality"
)

// AppVersion is set at build time via -ldflags "-X ...AppVersion=..."
var AppVersion = "dev"

// Panel is tray-only: a hidden Fyne window keeps the process alive; the
// system-tray context menu is the entire UI on every platform.
type Panel struct {
	win   fyne.Window
	app   fyne.App
	core  *core.Manager
	proxy proxy.Controller
	dir   string

	busy     bool
	miStatus *fyne.MenuItem
	miToggle *fyne.MenuItem
	miAuto   *fyne.MenuItem
	trayMenu *fyne.Menu
}

func NewPanel(a fyne.App, dir string) *Panel {
	// Tiny hidden owner window — never shown. Required so Fyne stays running
	// and clipboard / dialogs have a window parent if needed later.
	w := a.NewWindow(AppName)
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(1, 1))
	w.SetCloseIntercept(func() { w.Hide() })

	p := &Panel{
		win:   w,
		app:   a,
		dir:   dir,
		core:  core.New(dir),
		proxy: proxy.New(dir),
	}
	p.setupTray()
	p.refresh()
	go p.poll()
	go p.ensureCore()
	// Tray app can register for login; proxy stays OFF until the user turns it on.
	if runtime.GOOS == "windows" && !isAutostartEnabled() {
		_ = setAutostart(true)
		log.Println("registered for Windows startup (proxy stays off until you Turn On)")
	}
	return p
}

func (p *Panel) setupTray() {
	desk, ok := p.app.(desktop.App)
	if !ok {
		log.Println("not a desktop app — no system tray")
		return
	}

	p.miStatus = fyne.NewMenuItem("Status: Disconnected", nil)
	p.miStatus.Disabled = true
	p.miToggle = fyne.NewMenuItem("Turn On", func() { p.onToggle() })

	copyItem := fyne.NewMenuItem("Copy VLESS URI", func() { p.copyURI() })
	logItem := fyne.NewMenuItem("Open log", func() { p.openLog() })
	folderItem := fyne.NewMenuItem("Open folder", func() { p.openFolder() })

	p.miAuto = fyne.NewMenuItem("Start with Windows", func() { p.toggleAutostart() })
	if runtime.GOOS != "windows" {
		p.miAuto = nil
	} else if isAutostartEnabled() {
		p.miAuto.Label = "✓ Start with Windows"
	}

	checkStable := fyne.NewMenuItem("Check updates (stable)", func() {
		go p.checkUpdate(update.ChannelStable)
	})
	checkNightly := fyne.NewMenuItem("Check updates (nightly)", func() {
		go p.checkUpdate(update.ChannelNightly)
	})

	quitItem := fyne.NewMenuItem("Quit", func() { p.quit() })

	items := []*fyne.MenuItem{
		p.miStatus,
		fyne.NewMenuItemSeparator(),
		p.miToggle,
		fyne.NewMenuItemSeparator(),
		copyItem,
		logItem,
		folderItem,
	}
	if p.miAuto != nil {
		items = append(items, p.miAuto)
	}
	items = append(items,
		fyne.NewMenuItemSeparator(),
		checkStable,
		checkNightly,
		fyne.NewMenuItemSeparator(),
		quitItem,
	)

	p.trayMenu = fyne.NewMenu(AppName, items...)

	icon := fyne.NewStaticResource("home-reality-tray.png", generateTrayIconPNG())
	p.app.SetIcon(icon)
	desk.SetSystemTrayIcon(icon)
	desk.SetSystemTrayMenu(p.trayMenu)
	desk.SetSystemTrayIcon(icon)
}

func (p *Panel) onToggle() {
	if p.busy {
		return
	}
	if p.core.Running() {
		p.setBusy(true)
		go func() {
			p.core.Stop()
			_ = p.proxy.Disable()
			p.setBusy(false)
			p.refresh()
		}()
		return
	}
	p.setBusy(true)
	go func() {
		if !p.core.HasBinary() {
			p.setStatus("Downloading sing-box…")
			if err := download.Ensure(p.dir, func(m string) { p.setStatus(m) }); err != nil {
				p.setStatus("Error: " + err.Error())
				p.setBusy(false)
				p.refresh()
				return
			}
		}
		if err := p.core.Start(); err != nil {
			p.setStatus("Error: " + err.Error())
			p.setBusy(false)
			p.refresh()
			return
		}
		if err := p.proxy.Enable(); err != nil {
			p.core.Stop()
			p.setStatus("Proxy error: " + err.Error())
			p.setBusy(false)
			p.refresh()
			return
		}
		p.setBusy(false)
		p.refresh()
	}()
}

func (p *Panel) setBusy(b bool) {
	p.busy = b
	if b {
		p.setStatus("Working…")
	}
}

func (p *Panel) setStatus(s string) {
	if p.miStatus == nil {
		return
	}
	if len(s) > 60 {
		s = s[:57] + "…"
	}
	p.miStatus.Label = s
	p.applyMenu()
}

func (p *Panel) applyMenu() {
	if desk, ok := p.app.(desktop.App); ok && p.trayMenu != nil {
		desk.SetSystemTrayMenu(p.trayMenu)
	}
}

func (p *Panel) refresh() {
	running := p.core.Running()
	if p.busy {
		p.setStatus("Status: Working…")
		if p.miToggle != nil {
			p.miToggle.Label = "…"
		}
	} else if running {
		p.setStatus("Status: Connected · " + LocalProxy)
		if p.miToggle != nil {
			p.miToggle.Label = "Turn Off"
		}
	} else {
		p.setStatus("Status: Disconnected")
		if p.miToggle != nil {
			p.miToggle.Label = "Turn On"
		}
	}
	p.applyMenu()
}

func (p *Panel) poll() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for range t.C {
		if !p.busy && !p.core.Running() && p.proxy.IsOurs() {
			_ = p.proxy.Disable()
		}
		p.refresh()
	}
}

func (p *Panel) ensureCore() {
	if p.core.HasBinary() {
		return
	}
	p.setStatus("Downloading sing-box (first run)…")
	_ = download.Ensure(p.dir, func(m string) { p.setStatus(m) })
	p.refresh()
}

func (p *Panel) copyURI() {
	p.win.Clipboard().SetContent(ShareURI)
	p.setStatus("URI copied")
	go func() {
		time.Sleep(1500 * time.Millisecond)
		p.refresh()
	}()
}

func (p *Panel) openLog() {
	appLog := filepath.Join(p.dir, "app.log")
	path := appLog
	if _, err := os.Stat(appLog); err != nil {
		path = p.core.LogPath
	}
	if _, err := os.Stat(path); err != nil {
		p.setStatus("No log yet")
		return
	}
	openPath(path)
}

func (p *Panel) openFolder() {
	openPath(p.dir)
}

func (p *Panel) toggleAutostart() {
	if runtime.GOOS != "windows" || p.miAuto == nil {
		return
	}
	on := !isAutostartEnabled()
	if err := setAutostart(on); err != nil {
		p.setStatus("Autostart: " + err.Error())
		return
	}
	if on {
		p.miAuto.Label = "✓ Start with Windows"
		p.setStatus("Will start with Windows (proxy stays off)")
	} else {
		p.miAuto.Label = "Start with Windows"
		p.setStatus("Removed from Windows startup")
	}
	p.applyMenu()
	go func() {
		time.Sleep(2 * time.Second)
		p.refresh()
	}()
}

func (p *Panel) checkUpdate(ch update.Channel) {
	p.setStatus("Checking " + string(ch) + "…")
	info, err := update.Check(ch, AppVersion)
	if err != nil {
		p.setStatus("Update check failed: " + err.Error())
		log.Println("update check:", err)
		return
	}
	if !info.Available {
		p.setStatus("Up to date (" + string(ch) + ")")
		go func() {
			time.Sleep(3 * time.Second)
			p.refresh()
		}()
		return
	}
	// Open the download / release page in the browser.
	p.setStatus("Update: " + info.Label)
	if info.URL != "" {
		openPath(info.URL)
	}
}

func (p *Panel) quit() {
	p.core.Stop()
	_ = p.proxy.Disable()
	p.app.Quit()
}

func openPath(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// cmd /c start works for both files and https:// URLs; hide the console.
		cmd = exec.Command("cmd", "/c", "start", "", path)
		cmd.SysProcAttr = windowsHideAttr()
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	_ = cmd.Start()
}

func EnsureConfig(dir, example string) error {
	dst := filepath.Join(dir, "sing-box-config.json")
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	b, err := os.ReadFile(example)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func DataDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "sing-box-config.json")); err != nil {
		if cwd, e := os.Getwd(); e == nil {
			if _, e2 := os.Stat(filepath.Join(cwd, "sing-box-config.json")); e2 == nil {
				return cwd, nil
			}
		}
	}
	return dir, nil
}

func generateTrayIconPNG() []byte {
	const s = 64
	img := image.NewRGBA(image.Rect(0, 0, s, s))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)
	bg := color.RGBA{R: 0x16, G: 0x18, B: 0x1D, A: 0xFF}
	green := color.RGBA{R: 0x34, G: 0xC7, B: 0x59, A: 0xFF}
	white := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	cx, cy := s/2, s/2
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			dx, dy := x-cx, y-cy
			r2 := dx*dx + dy*dy
			if r2 <= 28*28 {
				img.Set(x, y, bg)
			}
			if r2 <= 18*18 {
				img.Set(x, y, green)
			}
			if r2 <= 7*7 {
				img.Set(x, y, white)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
