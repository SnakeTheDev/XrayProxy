#!/bin/bash
# Removes Home-Reality completely - double-click this file.
APP="$HOME/.homereality"
AGENT="com.homereality.menubar"
launchctl bootout "gui/$(id -u)/$AGENT" >/dev/null 2>&1
[ -x "$APP/proxyctl.sh" ] && "$APP/proxyctl.sh" stop >/dev/null 2>&1
pkill -f "$APP/menubar.js" >/dev/null 2>&1
rm -f "$HOME/Library/LaunchAgents/$AGENT.plist"
rm -rf "$APP"
echo "Home-Reality has been removed and your proxy settings were put back as they were."
read -n 1 -s -r -p "Press any key to close..."; echo
