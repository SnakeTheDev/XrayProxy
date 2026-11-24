# macOS – Home-Reality

Simple system-proxy client with a menu-bar shield icon.

---

## Install (easiest)

1. Unzip so you have this **Mac** folder.
2. Double-click **`Install.command`**.  
   - If macOS says it can’t be opened: **right-click → Open → Open**.
3. Enter your Mac password if asked (needed to change the system proxy).
4. Look at the top-right of the screen for a **shield** icon (or “HR” text).
5. Click it → **Turn On**.

### One-liner (if double-click is blocked)

Open Terminal and paste (adjust the path if needed):

```bash
cd ~/Downloads/Mac && xattr -dr com.apple.quarantine . ; bash Install.command
```

The installer:
- Downloads sing-box for your Mac (Apple Silicon or Intel)
- Installs into `~/.homereality`
- Starts the menu-bar app
- Runs a self-test and copies the report to your clipboard + Desktop

---

## Daily use

| Action | How |
|--------|-----|
| Turn on | Click shield → **Turn On** (icon turns green) |
| Turn off | Click shield → **Turn Off** |
| Status | Icon colour + menu text (“● Connected”) |
| Quit | Menu → Quit (stays closed until next login or you start it again) |

- “Start at Login” is on by default, but the **proxy stays off** until you click Turn On.
- Turning off restores your previous macOS proxy settings exactly.
- It turns the proxy off automatically on shutdown/restart.

### Menu extras

- Copy URI  
- Open Log  
- Open Folder  
- Run Self-Test  
- Start at Login (toggle)

---

## Self-test

- Runs automatically after install.
- Report is copied to the clipboard and saved as `Desktop/HomeReality-selftest.txt`.
- Re-run any time: double-click **`Test.command`** or use the menu item.

It checks that the engine starts, traffic goes through the server, the system proxy turns on/off cleanly, and the menu-bar app is alive.

---

## Troubleshooting

**No shield icon**
- Icons can be hidden by the notch or a crowded menu bar — hold **Cmd** and drag icons to make room, or quit other menu-bar apps.
- Force start:
  ```bash
  launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.homereality.menubar.plist
  ```
  If it says already loaded:
  ```bash
  launchctl kickstart -k gui/$(id -u)/com.homereality.menubar
  ```

**Still nothing**
```bash
cat ~/.homereality/menubar.log ~/.homereality/menubar.err.log
```

**No internet after Turn Off**
- System Settings → Network → your connection → Details → Proxies → switch everything off.

**Couldn’t download sing-box**
- Get the matching build from https://github.com/SagerNet/sing-box/releases  
  (`darwin-arm64` for M-series, `darwin-amd64` for Intel), unpack, put the `sing-box` binary next to `Install.command`, and run the installer again.

**Want more detailed logs**
- Edit `~/.homereality/sing-box-config.json` and set `"level": "info"`.

---

## Remove

Double-click **`Uninstall.command`**.

It stops the app, restores your proxy settings, and removes the LaunchAgent and `~/.homereality` files.

---

## Files

| File | Purpose |
|------|---------|
| `Install.command` | Install + start (double-click this) |
| `Uninstall.command` | Clean removal |
| `Test.command` | Re-run the self-test any time |
| `proxyctl.sh` | Core on/off + proxy restore logic |
| `menubar.js` | Menu-bar app (JXA) |
| `selftest.sh` | Automated checks |
| `sing-box-config.json` | Client config (edit for your own server) |
