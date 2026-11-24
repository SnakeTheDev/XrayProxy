HOME-REALITY FOR MAC
====================

FASTEST WAY TO TEST (about 2 minutes)
1. Unzip, open the Terminal app, and paste this one line:

     cd ~/Downloads/HomeReality-Mac && xattr -dr com.apple.quarantine . ; bash Install.command

   (adjust the path if you unzipped somewhere else)
2. It installs, then runs a self-test by itself. If your Mac asks for its
   password, enter it - that is macOS letting the app change the proxy setting.
3. At the end the test report is already on your clipboard (and saved to
   Desktop/HomeReality-selftest.txt). Paste it to Claude.
4. Look at the top right of the screen for a shield icon. Click it > Turn On
   (turns green when connected) and Turn Off. Tell Claude how it looked.

Normal install: double-click Install.command (if blocked: right-click > Open).

USING IT
- Grey shield = off. Green/blue shield = connected. If the icon can't be drawn
  you'll see "HR" text in the menu bar instead.
- It opens at login but NEVER turns the proxy on by itself.
- Turning it off puts your Mac's proxy settings back exactly as they were.
- It turns the proxy off automatically on shutdown/restart.
- Menu: Copy URI, Open Log, Open Folder, Run Self-Test.

IF SOMETHING GOES WRONG
- Menu shows a warning with the reason; click it to open the log.
- Run Test.command (or menu > Run Self-Test) any time for a fresh report.
- Logs are in ~/.homereality : proxy_core.log, menubar.log, menubar.err.log.
- Only warnings are logged. For full detail change "level": "warn" to "info"
  in ~/.homereality/sing-box-config.json.
- Couldn't download sing-box? Get the darwin-arm64 (M-series) or darwin-amd64
  (Intel) file from https://github.com/SagerNet/sing-box/releases, unpack it, put
  the file called "sing-box" next to Install.command and run it again.
- No internet after turning it off? System Settings > Network > your
  connection > Details > Proxies, and switch everything off.

REMOVE IT
Double-click Uninstall.command.
