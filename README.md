## Download

- [Windows](https://github.com/SnakeTheDev/XrayProxy/releases/latest/download/HomeReality-Windows.zip)
- [macOS](https://github.com/SnakeTheDev/XrayProxy/releases/latest/download/HomeReality-macOS.zip)

# XrayProxy (Home-Reality)

Simple system-proxy client for **Windows** and **macOS**.  
Uses [sing-box](https://github.com/SagerNet/sing-box) + VLESS/REALITY.

One click to turn on. One click to turn off.  
Your previous proxy settings are restored when you turn it off.  
It never turns the proxy on by itself after a reboot.

---

## Quick start

### Windows
1. Open the **[Win](Win/)** folder
2. Double-click **`ProxyTray.vbs`**
3. Wait for the first-run download of sing-box (~35 MB)
4. Click the tray icon → **Turn On**

Full details → [Win/README.md](Win/README.md)

### macOS
1. Open the **[Mac](Mac/)** folder
2. Double-click **`Install.command`**  
   (if blocked: right-click → Open → Open)
3. Enter your Mac password if asked
4. Click the shield in the menu bar → **Turn On**

Full details → [Mac/README.md](Mac/README.md)

---

## Folder layout

```
XrayProxy/
├── README.md          ← you are here
├── Win/               ← Windows client
│   ├── README.md
│   ├── ProxyTray.vbs  ← start here
│   ├── proxy_tray.ps1
│   ├── get-singbox.ps1
│   ├── proxy_toggle.bat
│   └── sing-box-config.json
└── Mac/               ← macOS client
    ├── README.md
    ├── Install.command
    ├── Uninstall.command
    ├── Test.command
    ├── proxyctl.sh
    ├── menubar.js
    └── sing-box-config.json
```

---

## Requirements

| Platform | Needs |
|----------|--------|
| Windows 10/11 | Internet on first run (downloads sing-box) |
| macOS (Apple Silicon or Intel) | Internet on first run (downloads sing-box) |

sing-box is downloaded from the official GitHub releases.  
On Windows the download is checksum-verified.

---

## How it works

- Runs a local mixed proxy on `127.0.0.1:2080` (HTTP + SOCKS).
- Sets the system proxy so browsers and most apps use it.
- Traffic goes through your VLESS + REALITY server.
- Turning off restores whatever proxy settings you had before.
- Apps that ignore the system proxy will not go through it (same limitation on both platforms).

---

## Notes

- The included `sing-box-config.json` is pre-configured for the author’s server.  
  If you use your **own** server, edit that file (UUID, Reality keys, address, etc.) before first run.
- Logs are quiet by default (`"level": "warn"`). Change to `"info"` only when debugging.
- This is a client only. It does not set up a server.

---

## Credits

- macOS port: [@palmernate1307-ux](https://github.com/palmernate1307-ux)
- Engine: [sing-box](https://github.com/SagerNet/sing-box)
