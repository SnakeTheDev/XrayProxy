#!/bin/bash
# Home-Reality installer for macOS - double-click this file.
cd "$(dirname "$0")" || exit 1
SRC="$(pwd)"
APP="$HOME/.homereality"
VER="1.14.0"
AGENT="com.homereality.menubar"
PLIST="$HOME/Library/LaunchAgents/$AGENT.plist"

# make sure macOS doesn't block the rest of the folder
xattr -dr com.apple.quarantine "$SRC" >/dev/null 2>&1
chmod +x "$SRC"/*.command "$SRC"/*.sh 2>/dev/null

pause() { echo; read -n 1 -s -r -p "Press any key to close..."; echo; }
fail()  { echo; echo "✗ $1"; pause; exit 1; }

clear
echo "=============================="
echo "   Home-Reality  -  Installer"
echo "=============================="
echo

case "$(uname -m)" in
  arm64)  PLAT="arm64" ;;
  x86_64) PLAT="amd64" ;;
  *) fail "Unsupported Mac processor: $(uname -m)" ;;
esac

# stop any older copy (and put the Mac's proxy settings back)
launchctl bootout "gui/$(id -u)/$AGENT" >/dev/null 2>&1
pkill -f "$APP/menubar.js" >/dev/null 2>&1
[ -x "$APP/proxyctl.sh" ] && "$APP/proxyctl.sh" stop >/dev/null 2>&1

mkdir -p "$APP" || fail "Couldn't create $APP"
cp "$SRC/sing-box-config.json" "$SRC/proxyctl.sh" "$SRC/menubar.js" "$SRC/selftest.sh" "$SRC/Test.command" "$APP/" || fail "Couldn't copy files."
chmod +x "$APP/proxyctl.sh" "$APP/selftest.sh" "$APP/Test.command"

# --- get the sing-box engine ---
if [ -f "$SRC/sing-box" ]; then
  echo "Using the sing-box file found next to the installer."
  cp "$SRC/sing-box" "$APP/sing-box"
elif [ -x "$APP/sing-box" ]; then
  echo "sing-box is already installed."
else
  TMP="$(mktemp -d)"
  URL="https://github.com/SagerNet/sing-box/releases/download/v$VER/sing-box-$VER-darwin-$PLAT.tar.gz"
  echo "Downloading sing-box $VER for your Mac ($PLAT)..."
  if ! curl -fL --retry 3 --connect-timeout 20 -o "$TMP/sb.tgz" "$URL"; then
    echo "That version didn't download - trying the latest release instead..."
    URL="$(curl -fsSL https://api.github.com/repos/SagerNet/sing-box/releases/latest | grep -Eo "https://[^\"]+darwin-$PLAT\.tar\.gz" | head -n 1)"
    [ -n "$URL" ] || fail "Couldn't download sing-box. Check your internet, or see README.txt to add it by hand."
    curl -fL --retry 3 --connect-timeout 20 -o "$TMP/sb.tgz" "$URL" || fail "Download failed. See README.txt to add sing-box by hand."
  fi
  tar -xzf "$TMP/sb.tgz" -C "$TMP" || fail "Couldn't unpack the download."
  BIN="$(find "$TMP" -name sing-box -type f | head -n 1)"
  [ -n "$BIN" ] || fail "sing-box wasn't found inside the download."
  cp "$BIN" "$APP/sing-box"
  find "$TMP" -name '*.dylib' -exec cp {} "$APP/" \; 2>/dev/null
  rm -rf "$TMP"
fi
chmod +x "$APP/sing-box"
xattr -cr "$APP" >/dev/null 2>&1

echo
"$APP/sing-box" version | head -n 1 || fail "sing-box won't run on this Mac."
if "$APP/sing-box" check -c "$APP/sing-box-config.json"; then echo "Config OK."; else echo "(Config check reported a problem - the self-test below will show details.)"; fi

# --- open at login + start now ---
"$APP/proxyctl.sh" login on
launchctl bootstrap "gui/$(id -u)" "$PLIST" >/dev/null 2>&1 || launchctl load "$PLIST" >/dev/null 2>&1

echo
echo "✓ Installed!  Look for the shield icon in the menu bar (top right)."
echo "  Click it and choose 'Turn On'. It never turns the proxy on by itself."
echo
if [ -z "$HR_NO_TEST" ]; then
  echo "Now running a quick self-test (you may be asked for your Mac password)..."
  sleep 2
  echo
  bash "$APP/selftest.sh"
fi
pause
