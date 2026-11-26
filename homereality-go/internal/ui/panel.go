package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/SnakeTheDev/XrayProxy/homereality-go/internal/core"
	"github.com/SnakeTheDev/XrayProxy/homereality-go/internal/download"
	"github.com/SnakeTheDev/XrayProxy/homereality-go/internal/proxy"
)

const (
	AppName    = "Home-Reality"
	ServerHost = "play.rowstonsoftware.com"
	ServerPort = 443
	LocalProxy = "127.0.0.1:2080"
	ShareURI   = "vless://a750d63d-3348-4f8b-be6a-be17606aff60@play.rowstonsoftware.com:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=dl.google.com&fp=chrome&pbk=npglY0nkJZ16gJ-AmWkt30hf1zgikjNRN9BhPUOKVnQ&sid=0123456789abcdef&type=tcp#Home-Reality"
)

type Panel struct {
	win   fyne.Window
	app   fyne.App
	core  *core.Manager
	proxy proxy.Controller
	dir   string

	toggle   *Toggle
	status   *canvas.Text
	sub      *canvas.Text
	errLabel *canvas.Text
	vHost    *canvas.Text
	vLocal   *canvas.Text
	vCore    *canvas.Text
	vType    *canvas.Text
	busy     bool
	miStatus *fyne.MenuItem
	miToggle *fyne.MenuItem
	trayMenu *fyne.Menu
}

func NewPanel(a fyne.App, dir string) *Panel {
	a.Settings().SetTheme(NewTheme())
	w := a.NewWindow(AppName)
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(320, 460))
	w.CenterOnScreen()

	p := &Panel{
		win:   w,
		app:   a,
		dir:   dir,
		core:  core.New(dir),
		proxy: proxy.New(dir),
	}
	p.build()
	p.setupTray()
	p.refresh()
	go p.poll()
	go p.ensureCore()
	return p
}

func (p *Panel) Show() {
	p.refresh()
	p.win.Show()
	p.win.RequestFocus()
}

func (p *Panel) Hide() { p.win.Hide() }

func (p *Panel) Window() fyne.Window { return p.win }

func (p *Panel) setupTray() {
	desk, ok := p.app.(desktop.App)
	if !ok {
		return
	}

	// On Windows the systray is only initialised when SetSystemTrayMenu is
	// called. Setting the icon before that produces:
	//   "systray error: unable to set icon: tray not ready yet"
	// so create the menu first, then the icon.
	p.miStatus = fyne.NewMenuItem("Status: Disconnected", nil)
	p.miStatus.Disabled = true
	p.miToggle = fyne.NewMenuItem("Turn On", func() { p.onToggle() })
	showItem := fyne.NewMenuItem("Show panel", func() { p.Show() })
	copyItem := fyne.NewMenuItem("Copy VLESS URI", func() { p.copyURI() })
	logItem := fyne.NewMenuItem("Open log", func() { p.openLog() })
	folderItem := fyne.NewMenuItem("Open application folder", func() { p.openFolder() })

	var autoItem *fyne.MenuItem
	if runtime.GOOS == "windows" {
		label := "Start with Windows"
		if isAutostartEnabled() {
			label = "✓ Start with Windows"
		}
		autoItem = fyne.NewMenuItem(label, func() {
			on := !isAutostartEnabled()
			if err := setAutostart(on); err != nil {
				p.setErr("Autostart: " + err.Error())
				return
			}
			if on {
				autoItem.Label = "✓ Start with Windows"
				p.setErr("Will start with Windows")
			} else {
				autoItem.Label = "Start with Windows"
				p.setErr("Removed from startup")
			}
			if desk, ok := p.app.(desktop.App); ok && p.trayMenu != nil {
				desk.SetSystemTrayMenu(p.trayMenu)
			}
			go func() {
				time.Sleep(1500 * time.Millisecond)
				if p.errLabel != nil && (p.errLabel.Text == "Will start with Windows" || p.errLabel.Text == "Removed from startup") {
					p.setErr("")
				}
			}()
		})
	}

	quitItem := fyne.NewMenuItem("Quit", func() { p.quit() })

	items := []*fyne.MenuItem{
		p.miStatus,
		fyne.NewMenuItemSeparator(),
		p.miToggle,
		showItem,
		fyne.NewMenuItemSeparator(),
		copyItem,
		logItem,
		folderItem,
	}
	if autoItem != nil {
		items = append(items, autoItem)
	}
	items = append(items, fyne.NewMenuItemSeparator(), quitItem)

	p.trayMenu = fyne.NewMenu(AppName, items...)

	// Always use a pure-Go generated PNG. Embedded assets and Fyne's default
	// BrokenImageIcon (SVG) both caused "image: unknown format" on Windows.
	icon := fyne.NewStaticResource("home-reality-tray.png", generateTrayIconPNG())
	p.app.SetIcon(icon)
	// Store icon before the tray starts so the onReady callback does not fall
	// back to the SVG default. May log "tray not ready yet" once — harmless.
	desk.SetSystemTrayIcon(icon)
	desk.SetSystemTrayMenu(p.trayMenu)
	// Set again after the tray is live (definitive).
	desk.SetSystemTrayIcon(icon)

	// Tray-only: never show the window on start. Close hides it.
	p.win.SetCloseIntercept(func() {
		p.win.Hide()
	})
}

func (p *Panel) build() {
	title := canvas.NewText(AppName, ColText)
	title.TextSize = 18
	title.TextStyle = fyne.TextStyle{Bold: true}

	subtitle := canvas.NewText("VLESS Reality", ColMuted)
	subtitle.TextSize = 12

	headerText := container.NewVBox(title, subtitle)
	header := container.NewBorder(nil, nil, nil, nil, container.NewPadded(headerText))

	p.toggle = NewToggle(func() { p.onToggle() })
	p.toggle.Resize(fyne.NewSize(132, 56))
	toggleBox := container.NewCenter(container.NewGridWrap(fyne.NewSize(132, 56), p.toggle))

	p.status = canvas.NewText("Disconnected", ColRed)
	p.status.TextSize = 16
	p.status.TextStyle = fyne.TextStyle{Bold: true}
	p.status.Alignment = fyne.TextAlignCenter

	p.sub = canvas.NewText("Tap the switch to connect", ColMuted)
	p.sub.TextSize = 12
	p.sub.Alignment = fyne.TextAlignCenter

	p.vHost = canvas.NewText(fmt.Sprintf("%s:%d", ServerHost, ServerPort), ColText)
	p.vHost.TextSize = 12
	p.vLocal = canvas.NewText(LocalProxy, ColText)
	p.vLocal.TextSize = 12
	p.vCore = canvas.NewText("—", ColText)
	p.vCore.TextSize = 12
	p.vType = canvas.NewText("sing-box", ColText)
	p.vType.TextSize = 12

	row := func(label string, val *canvas.Text) fyne.CanvasObject {
		l := canvas.NewText(label, ColMuted)
		l.TextSize = 12
		return container.NewBorder(nil, nil, l, nil, val)
	}

	cardBG := canvas.NewRectangle(ColCard)
	cardBG.CornerRadius = 10
	cardInner := container.NewVBox(
		row("Host", p.vHost),
		row("Local", p.vLocal),
		row("Core", p.vCore),
		row("Type", p.vType),
	)
	card := container.NewStack(cardBG, container.NewPadded(cardInner))

	p.errLabel = canvas.NewText("", ColRed)
	p.errLabel.TextSize = 11

	mkBtn := func(label string, fn func()) *widget.Button {
		b := widget.NewButton(label, fn)
		b.Importance = widget.MediumImportance
		return b
	}
	btns := container.NewGridWithColumns(4,
		mkBtn("Copy URI", p.copyURI),
		mkBtn("Log", p.openLog),
		mkBtn("Folder", p.openFolder),
		mkBtn("Quit", p.quit),
	)

	content := container.NewVBox(
		header,
		layout.NewSpacer(),
		toggleBox,
		container.NewPadded(p.status),
		container.NewPadded(p.sub),
		layout.NewSpacer(),
		container.NewPadded(card),
		container.NewPadded(p.errLabel),
		layout.NewSpacer(),
		container.NewPadded(btns),
	)

	bg := canvas.NewRectangle(ColBg)
	p.win.SetContent(container.NewStack(bg, container.NewPadded(content)))
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
	p.errLabel.Text = ""
	p.errLabel.Refresh()
	go func() {
		if !p.core.HasBinary() {
			p.setErr("Downloading sing-box…")
			if err := download.Ensure(p.dir, func(m string) { p.setErr(m) }); err != nil {
				p.setErr(err.Error())
				p.setBusy(false)
				p.refresh()
				return
			}
		}
		if err := p.core.Start(); err != nil {
			p.setErr(err.Error())
			p.setBusy(false)
			p.refresh()
			return
		}
		if err := p.proxy.Enable(); err != nil {
			p.core.Stop()
			p.setErr("Proxy: " + err.Error())
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
		p.toggle.SetState(2)
	}
}

func (p *Panel) setErr(s string) {
	if len(s) > 220 {
		s = s[:220] + "…"
	}
	p.errLabel.Text = s
	p.errLabel.Refresh()
}

func (p *Panel) refresh() {
	running := p.core.Running()
	if p.busy {
		p.toggle.SetState(2)
		if p.miStatus != nil {
			p.miStatus.Label = "Status: Working…"
		}
		p.status.Text = "Working…"
		p.status.Color = ColAmber
		p.sub.Text = ""
		if p.miToggle != nil {
			p.miToggle.Label = "…"
		}
	} else if running {
		p.toggle.SetState(1)
		if p.miStatus != nil {
			p.miStatus.Label = "Status: Connected"
		}
		p.status.Text = "Connected"
		p.status.Color = ColGreen
		up := ""
		if t := p.core.StartedAt(); !t.IsZero() {
			up = "Up " + formatUp(time.Since(t)) + "  ·  "
		}
		p.sub.Text = up + LocalProxy
		if p.miToggle != nil {
			p.miToggle.Label = "Turn Off"
		}
	} else {
		p.toggle.SetState(0)
		if p.miStatus != nil {
			p.miStatus.Label = "Status: Disconnected"
		}
		p.status.Text = "Disconnected"
		p.status.Color = ColRed
		p.sub.Text = "Tap the switch to connect"
		if p.miToggle != nil {
			p.miToggle.Label = "Turn On"
		}
	}
	p.status.Refresh()
	p.sub.Refresh()

	if desk, ok := p.app.(desktop.App); ok && p.trayMenu != nil {
		desk.SetSystemTrayMenu(p.trayMenu)
	}

	if p.core.HasBinary() {
		p.vCore.Text = "sing-box " + download.Version
	} else {
		p.vCore.Text = "not installed"
	}
	p.vCore.Refresh()
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
	p.setErr("Downloading sing-box (first run)…")
	_ = download.Ensure(p.dir, func(m string) { p.setErr(m) })
	p.setErr("")
	p.refresh()
}

func (p *Panel) copyURI() {
	p.win.Clipboard().SetContent(ShareURI)
	p.setErr("URI copied")
	go func() {
		time.Sleep(1500 * time.Millisecond)
		if p.errLabel.Text == "URI copied" {
			p.setErr("")
		}
	}()
}

func (p *Panel) openLog() {
	// Prefer app.log (startup / Fyne messages); fall back to sing-box log.
	appLog := filepath.Join(p.dir, "app.log")
	path := appLog
	if _, err := os.Stat(appLog); err != nil {
		path = p.core.LogPath
	}
	if _, err := os.Stat(path); err != nil {
		p.setErr("No log yet.")
		return
	}
	openPath(path)
}

// generateTrayIconPNG builds a simple 64x64 PNG in memory (no embed dependency).
// Fyne on Windows rejects SVG defaults and some embed paths with
// "image: unknown format"; a hand-encoded PNG always works.
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

func (p *Panel) openFolder() {
	openPath(p.dir)
}

func (p *Panel) quit() {
	p.core.Stop()
	_ = p.proxy.Disable()
	p.app.Quit()
}

func formatUp(d time.Duration) string {
	if d.Hours() >= 1 {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return "just now"
}

func openPath(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
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
