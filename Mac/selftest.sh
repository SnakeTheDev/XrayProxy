#!/bin/bash
# Home-Reality self-test: checks every part, writes a report, copies it to the clipboard.
APP="$HOME/.homereality"
CTL="$APP/proxyctl.sh"
REPORT="$HOME/Desktop/HomeReality-selftest.txt"
OUT="$(mktemp /tmp/hr_report.XXXXXX)"
P=0; F=0
say()  { echo "$*" | tee -a "$OUT"; }
ok()   { say "  PASS  $*"; P=$((P+1)); }
bad()  { say "  FAIL  $*"; F=$((F+1)); }
warn() { say "  WARN  $*"; }
info() { say "        $*"; }
mask() { echo "$1" | awk -F. 'NF==4{print $1"."$2".x.x"; next}{print "(hidden)"}'; }
snap() { scutil --proxy 2>/dev/null | awk '$1 ~ /^(HTTP|HTTPS|SOCKS)(Enable|Proxy|Port)$/ {v[$1]=$3} END {n=split("HTTP HTTPS SOCKS",k," "); for(i=1;i<=n;i++){e=v[k[i]"Enable"]+0; printf "%s=%d", k[i], e; if(e==1) printf " %s:%s", v[k[i]"Proxy"], v[k[i]"Port"]; printf "\n"}}'; }
SOCKS="-x socks5h://127.0.0.1:2080"; HTTPX="-x http://127.0.0.1:2080"

clear 2>/dev/null
say "=============================="
say "  Home-Reality  -  Self-test"
say "=============================="
say "$(date '+%F %T')   macOS $(sw_vers -productVersion 2>/dev/null) ($(sw_vers -buildVersion 2>/dev/null))   $(uname -m)   bash $BASH_VERSION"
say

if [ ! -x "$CTL" ]; then bad "Not installed - run Install.command first."; exit 1; fi
BEFORE="$(snap)"
trap '"$CTL" stop >/dev/null 2>&1' EXIT

say "1) Files and engine"
for f in proxyctl.sh sing-box sing-box-config.json menubar.js; do [ -e "$APP/$f" ] && ok "$f present" || bad "$f missing"; done
V="$("$APP/sing-box" version 2>&1 | head -n 1)"
[ -n "$V" ] && ok "engine runs: $V" || bad "sing-box won't run (Gatekeeper? try: xattr -cr ~/.homereality)"
C="$("$APP/sing-box" check -c "$APP/sing-box-config.json" 2>&1)" && ok "config valid" || { bad "config check failed"; info "$C"; }
J="$(osascript -l JavaScript -e 'ObjC.import("Cocoa"); "ok"' 2>&1)"
[ "$J" = "ok" ] && ok "osascript/JavaScript bridge works" || { bad "osascript JXA problem"; info "$J"; }

say; say "2) Core + tunnel (system settings untouched)"
"$CTL" start --core-only; RC=$?
if [ "$RC" = "0" ] && [ "$("$CTL" status)" = "on" ]; then ok "core running on 127.0.0.1:2080"; else bad "core did not start (rc=$RC): $(tail -n 3 "$APP/last_error" 2>/dev/null)"; fi
CODE="$(curl -s -o /dev/null -m 15 -w '%{http_code}' $SOCKS https://www.gstatic.com/generate_204 2>/dev/null)"
[ "$CODE" = "204" ] && ok "traffic flows through server (SOCKS)" || bad "no traffic via SOCKS (http=$CODE) - server unreachable / blocked?"
CODE="$(curl -s -o /dev/null -m 15 -w '%{http_code}' $HTTPX https://www.gstatic.com/generate_204 2>/dev/null)"
[ "$CODE" = "204" ] && ok "traffic flows through server (HTTP proxy, like browsers use)" || bad "no traffic via HTTP proxy (http=$CODE)"
PIP="$(curl -s -m 15 $SOCKS https://api.ipify.org 2>/dev/null)"; DIP="$(curl -s -m 15 https://api.ipify.org 2>/dev/null)"
if [ -n "$PIP" ] && [ "$PIP" != "$DIP" ]; then ok "exit IP differs from your normal IP ($(mask "$PIP") vs $(mask "$DIP"))"
elif [ -n "$PIP" ]; then warn "exit IP same as normal IP ($(mask "$PIP"))"; else bad "couldn't read exit IP through proxy"; fi

say; say "3) System proxy on (may ask for your Mac password)"
"$CTL" start; RC=$?
S="$(scutil --proxy 2>/dev/null)"
if echo "$S" | grep -q 'HTTPSProxy : 127.0.0.1' && echo "$S" | grep -q 'HTTPSPort : 2080' && echo "$S" | grep -q 'HTTPSEnable : 1'; then ok "macOS web proxy points at 127.0.0.1:2080"; else bad "system proxy not set (rc=$RC): $(tail -n 2 "$APP/last_error" 2>/dev/null)"; fi
echo "$S" | grep -q 'SOCKSEnable : 1' && ok "SOCKS proxy set" || bad "SOCKS proxy not set"

say; say "4) Turn off + restore"
"$CTL" stop; RC=$?
[ "$("$CTL" status)" = "off" ] && ok "core stopped" || bad "core still running / error (rc=$RC)"
S="$(scutil --proxy 2>/dev/null)"
if echo "$S" | grep -q 'HTTPSProxy : 127.0.0.1' && echo "$S" | grep -q 'HTTPSEnable : 1'; then bad "macOS still points at our proxy"; else ok "system proxy no longer ours"; fi
[ "$(snap)" = "$BEFORE" ] && ok "your previous proxy settings are back exactly as they were" || { bad "proxy settings differ from before the test"; info "before: $(echo $BEFORE | tr '\n' ' ')"; info "after:  $(snap | tr '\n' ' ')"; }

say; say "5) Menu bar app"
AG="$(launchctl print "gui/$(id -u)/com.homereality.menubar" 2>&1)"
echo "$AG" | grep -q 'state = running' && ok "login agent is running" || { bad "login agent not running"; info "$(echo "$AG" | head -n 3)"; }
i=0; while [ $i -lt 12 ] && [ -z "$(find "$APP/menubar.alive" -mmin -1 2>/dev/null)" ]; do sleep 1; i=$((i+1)); done
[ -n "$(find "$APP/menubar.alive" -mmin -1 2>/dev/null)" ] && ok "menu bar app is alive (heartbeat)" || bad "menu bar app not running its loop"
grep -q 'started; statusItem=ok' "$APP/menubar.log" 2>/dev/null && ok "menu bar item created" || bad "menu bar item NOT created: $(tail -n 2 "$APP/menubar.log" 2>/dev/null)"
[ -s "$APP/menubar.err.log" ] && { warn "menubar.err.log has output:"; info "$(tail -n 6 "$APP/menubar.err.log")"; }

say; say "------------------------------"
if [ "$F" = "0" ]; then say "RESULT: all $P checks passed."; else say "RESULT: $P passed, $F FAILED."; fi
say "Manual check: you should now see a grey shield in the menu bar (top right)."
say "Click it > Turn On, wait a few seconds, it should turn green. Then Turn Off."
if [ "$F" != "0" ]; then
  { echo; echo "--- logs ---"; echo "[proxy_core.log]"; tail -n 12 "$APP/proxy_core.log" 2>/dev/null; echo "[last_error]"; cat "$APP/last_error" 2>/dev/null; echo "[menubar.log]"; tail -n 8 "$APP/menubar.log" 2>/dev/null; } >> "$OUT"
fi
cp "$OUT" "$REPORT" 2>/dev/null; pbcopy < "$OUT" 2>/dev/null
say; say "Report copied to your clipboard (and saved to Desktop/HomeReality-selftest.txt) - paste it to Claude."
rm -f "$OUT"
