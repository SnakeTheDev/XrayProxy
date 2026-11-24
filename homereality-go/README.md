# Home-Reality (Go + Fyne)

Cross-platform system-tray client for the Home-Reality / RSNet proxy.

The app starts hidden and is controlled from its native notification-area / menu-bar icon. It does not open a separate panel window on startup.

**Supported:** Windows, macOS, Linux

---

## Features

- Native system-tray icon with an embedded PNG resource
- Tray menu with connection status and **Turn On / Turn Off**
- **Copy VLESS URI**, **Open log**, **Open application folder**, **Quit**
- No visible Fyne window on startup
- Auto-downloads **sing-box 1.14.0** on first run
- Sets system HTTP/HTTPS proxy (and SOCKS on macOS)
- Restores previous proxy settings when you turn off
- Clears stale proxy if the core process dies

---

## Run (dev)

```bash
cd homereality-go
go mod tidy
go run ./cmd/homereality
```

On Linux you need GUI build deps:

```bash
sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev
```

---

## Build locally

```bash
go build -ldflags "-s -w" -o HomeReality ./cmd/homereality
```

Put `sing-box-config.json` next to the binary (a copy is in this folder).

---

## GitHub Actions

Push this tree on branch **`go-rewrite`** (or merge to `main`).  
Workflow: `.github/workflows/build.yml`

Builds and uploads zips:

| Artifact | Runner |
|----------|--------|
| `HomeReality-windows-amd64.zip` | windows-latest |
| `HomeReality-darwin-arm64.zip` | macos-latest |
| `HomeReality-darwin-amd64.zip` | macos-latest |
| `HomeReality-linux-amd64.zip` | ubuntu-latest |

You do **not** need a Mac — Actions builds the macOS binaries for you.

---

## How to use

1. Unzip the build for your OS  
2. Run `HomeReality` (or `.exe` on Windows)  
3. Wait for first-run sing-box download if needed  
4. Flip the toggle **On**

---

## Layout of this folder

```
homereality-go/
├── cmd/homereality/main.go
├── internal/
│   ├── ui/          # Fyne panel, theme, toggle
│   ├── core/        # sing-box process
│   ├── proxy/       # system proxy (win/mac/linux)
│   └── download/    # first-run sing-box fetch
├── sing-box-config.json
├── go.mod
└── README.md
```

When placing on GitHub under `XrayProxy`, either:

- put this folder at repo root on branch `go-rewrite`, or  
- keep it as `homereality-go/` and keep the workflow `working-directory: homereality-go`

---

## Notes

- Fyne cannot clone WinForms pixel-perfect; colours, toggle, card, and actions match the Windows app closely.
- macOS may prompt for admin when changing the system proxy (`networksetup`).
- Edit `sing-box-config.json` for your own server before distributing widely.
