# Windows – Home-Reality

Simple system-proxy client. One double-click to start.

---

## Install & run

1. Put this whole **Win** folder anywhere you like (Desktop, Downloads, etc.).
2. Double-click **`ProxyTray.vbs`**.
3. On first run it downloads sing-box automatically (~35 MB).  
   The tray panel shows **Downloading sing-box (first run)…**
4. When it says **Ready**, click the tray icon → **Turn On**.


---

## Daily use

| Action | How |
|--------|-----|
| Turn on | Tray icon → **Turn On** |
| Turn off | Tray icon → **Turn Off** |
| Status | Hover the tray icon or open the panel |
| Quit | Tray → Quit (proxy is turned off first) |

- It does **not** turn the proxy on by itself after a reboot.
- Turning off restores your previous Windows proxy settings.

### Optional: classic batch menu

Double-click **`proxy_toggle.bat`** for a simple Enable / Disable menu.  
It will also download sing-box on first use if the tray has never been run.

---

## First-run download

`get-singbox.ps1` fetches the official sing-box 1.14.0 Windows build (amd64 or arm64), checks the SHA-256 hash, and places:

- `sing-box.exe`
- `libcronet.dll`

next to the scripts.

You can also run it by hand:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\get-singbox.ps1
```

If a `sing-box.exe` is already present, nothing is downloaded.

---

## Troubleshooting

**Tray doesn’t appear**
- Right-click `ProxyTray.vbs` → Open with → Windows PowerShell, or
- Check Windows Defender / antivirus didn’t quarantine the scripts.

**Stuck on “Downloading…”**
- Check internet and firewall.
- Or run `get-singbox.ps1` yourself (see above).

**No internet after Turn Off**
- Settings → Network & Internet → Proxy → make sure everything is off.

**Want quieter logs**
- Open `sing-box-config.json` and set `"level": "warn"` (already the default on Mac; Windows may still be `"info"`).

---

## Remove

1. Turn the proxy **Off**.
2. Quit the tray app.
3. Delete this folder.

No leftover system services are installed.

---

## Files

| File | Purpose |
|------|---------|
| `ProxyTray.vbs` | Start the tray app (double-click this) |
| `proxy_tray.ps1` | The actual tray UI |
| `get-singbox.ps1` | Downloads sing-box on first run |
| `proxy_toggle.bat` | Optional Enable/Disable menu |
| `sing-box-config.json` | Client config (edit for your own server) |
