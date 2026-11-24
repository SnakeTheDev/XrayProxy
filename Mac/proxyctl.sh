#!/bin/bash
# Home-Reality proxy controller for macOS.
# Usage: proxyctl.sh start [--core-only] | stop | status | cleanup | stale | copyuri | login on|off|status
DIR="$(cd "$(dirname "$0")" && pwd)"
BIN="$DIR/sing-box"
CFG="$DIR/sing-box-config.json"
LOG="$DIR/proxy_core.log"
ERR="$DIR/last_error"
LOCK="$DIR/.busy"
PIDF="$DIR/sing-box.pid"
PREV="$DIR/prev_proxy.tsv"
STALEF="$DIR/.stale_tried"
PLIST="$HOME/Library/LaunchAgents/com.homereality.menubar.plist"
ADDR="127.0.0.1"
PORT="2080"
TYPES="web secure socks"
URI='vless://a750d63d-3348-4f8b-be6a-be17606aff60@play.rowstonsoftware.com:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=dl.google.com&fp=chrome&pbk=npglY0nkJZ16gJ-AmWkt30hf1zgikjNRN9BhPUOKVnQ&sid=0123456789abcdef&type=tcp#Home-Reality'

# ---------- core process (tracked by PID, so other sing-box copies are never touched) ----------
core_pid() {
  local p=""
  [ -f "$PIDF" ] && p="$(cat "$PIDF" 2>/dev/null)"
  if [ -n "$p" ] && kill -0 "$p" 2>/dev/null && ps -p "$p" -o command= 2>/dev/null | grep -q "sing-box" && ! ps -p "$p" -o stat= 2>/dev/null | grep -q '^Z'; then echo "$p"; return 0; fi
  p="$(pgrep -f "$BIN run -c $CFG" 2>/dev/null | head -n 1)"
  [ -n "$p" ] && { echo "$p"; return 0; }
  return 1
}
running() { core_pid >/dev/null 2>&1; }
port_open() { (exec 3<>"/dev/tcp/$ADDR/$PORT") 2>/dev/null; }
stop_core() {
  local p i
  p="$(core_pid)" && kill "$p" 2>/dev/null
  pkill -f "$BIN run -c $CFG" 2>/dev/null
  for i in 1 2 3 4 5 6 7 8 9 10; do running || break; sleep 0.2; done
  running && pkill -9 -f "$BIN run -c $CFG" 2>/dev/null
  rm -f "$PIDF"
}

# ---------- system proxy (restores whatever the Mac had before) ----------
services() { networksetup -listallnetworkservices 2>/dev/null | tail -n +2 | sed '/^\*/d'; }
getname() { case "$1" in web) echo getwebproxy;; secure) echo getsecurewebproxy;; socks) echo getsocksfirewallproxy;; esac; }
setname() { case "$1" in web) echo setwebproxy;; secure) echo setsecurewebproxy;; socks) echo setsocksfirewallproxy;; esac; }
field() { awk -F': ' -v k="$1" '$1==k{print $2; exit}'; }

ours_enabled() {  # service type
  local out; out="$(networksetup -"$(getname "$2")" "$1" 2>/dev/null)"
  [ "$(echo "$out" | field Enabled)" = "Yes" ] && [ "$(echo "$out" | field Server)" = "$ADDR" ] && [ "$(echo "$out" | field Port)" = "$PORT" ]
}
our_proxy_set() {
  local s t
  while IFS= read -r s; do
    for t in $TYPES; do ours_enabled "$s" "$t" && return 0; done
  done < <(services)
  return 1
}
save_prev() {  # remember the user's own proxy settings once, before we change them
  # keep an existing backup only if we're still the active proxy (crash leftover); otherwise the live settings are the truth
  [ -f "$PREV" ] && our_proxy_set && return 0
  local s t out en sv pt tmp="$PREV.tmp"
  : > "$tmp"
  while IFS= read -r s; do
    for t in $TYPES; do
      out="$(networksetup -"$(getname "$t")" "$s" 2>/dev/null)"
      en="$(echo "$out" | field Enabled)"; sv="$(echo "$out" | field Server)"; pt="$(echo "$out" | field Port)"
      [ "$sv" = "$ADDR" ] && [ "$pt" = "$PORT" ] && en="No"
      [ -n "$en" ] || en="No"; [ -n "$sv" ] || sv="-"; [ -n "$pt" ] || pt="0"
      printf '%s\t%s\t%s\t%s\t%s\n' "$s" "$t" "$en" "$sv" "$pt" >> "$tmp"
    done
  done < <(services)
  mv "$tmp" "$PREV"
}
build_cmds() {
  local state="$1" s t line en sv pt
  while IFS= read -r s; do
    for t in $TYPES; do
      if [ "$state" = "on" ]; then
        printf 'networksetup -%s %q %s %s\n' "$(setname "$t")" "$s" "$ADDR" "$PORT"
        printf 'networksetup -%sstate %q on\n' "$(setname "$t")" "$s"
      else
        en="No"; sv="-"; pt="0"
        line="$(awk -F'\t' -v s="$s" -v t="$t" '$1==s && $2==t {print $3 "\t" $4 "\t" $5; exit}' "$PREV" 2>/dev/null)"
        [ -n "$line" ] && IFS=$'\t' read -r en sv pt <<< "$line"
        if [ "$en" = "Yes" ] && [ "$sv" != "-" ] && ! { [ "$sv" = "$ADDR" ] && [ "$pt" = "$PORT" ]; }; then
          printf 'networksetup -%s %q %s %s\n' "$(setname "$t")" "$s" "$sv" "$pt"
          printf 'networksetup -%sstate %q on\n' "$(setname "$t")" "$s"
        else
          printf 'networksetup -%sstate %q off\n' "$(setname "$t")" "$s"
        fi
      fi
    done
  done < <(services)
}
proxy_ok() {
  local s t
  if [ "$1" = "on" ]; then
    s="$(services | head -n 1)"; [ -n "$s" ] || return 0
    for t in $TYPES; do ours_enabled "$s" "$t" || return 1; done
    return 0
  fi
  ! our_proxy_set
}
# Try normally first; if macOS refuses, ask for the Mac password once.
apply_proxy() {
  local state="$1" tmp
  [ "$state" = "on" ] && save_prev
  tmp="$(mktemp /tmp/hr_ns.XXXXXX)"
  build_cmds "$state" > "$tmp"
  bash "$tmp" >/dev/null 2>&1
  if ! proxy_ok "$state"; then
    osascript -e "do shell script \"/bin/bash $tmp\" with administrator privileges" >/dev/null 2>&1
  fi
  rm -f "$tmp"
  if proxy_ok "$state"; then
    [ "$state" = "off" ] && rm -f "$PREV"
    return 0
  fi
  return 1
}

# ---------- commands ----------
strip_ansi() { LC_ALL=C sed $'s/\033\\[[0-9;]*m//g'; }

do_start() {
  local coreonly=0 i=0
  [ "$1" = "--core-only" ] && coreonly=1
  touch "$LOCK"; trap 'rm -f "$LOCK"' EXIT
  rm -f "$ERR"
  if [ ! -x "$BIN" ]; then echo "sing-box not found in $DIR" > "$ERR"; exit 2; fi
  stop_core
  if ! "$BIN" check -c "$CFG" > "$LOG" 2>&1; then tail -n 4 "$LOG" | strip_ansi > "$ERR"; exit 1; fi
  nohup "$BIN" run -c "$CFG" > "$LOG" 2>&1 &
  echo $! > "$PIDF"
  disown 2>/dev/null
  while [ $i -lt 30 ]; do
    sleep 0.2; i=$((i+1))
    running || break
    port_open && break
  done
  if running && port_open; then
    if [ "$coreonly" = "0" ] && ! apply_proxy on; then
      stop_core
      echo "Couldn't change the system proxy (was the password prompt cancelled?)" > "$ERR"
      exit 1
    fi
  else
    tail -n 4 "$LOG" 2>/dev/null | strip_ansi > "$ERR"
    [ -s "$ERR" ] || echo "Core did not start listening on $ADDR:$PORT." > "$ERR"
    stop_core
    exit 1
  fi
}
do_stop() {
  touch "$LOCK"; trap 'rm -f "$LOCK"' EXIT
  stop_core
  if apply_proxy off; then rm -f "$ERR"; else echo "Couldn't turn the system proxy off - see System Settings > Network > Proxies." > "$ERR"; exit 1; fi
}
do_status() {
  if [ -f "$LOCK" ]; then
    if [ -n "$(find "$LOCK" -mmin +1 2>/dev/null)" ]; then rm -f "$LOCK"; else echo busy; return; fi
  fi
  if running; then echo on; elif [ -s "$ERR" ]; then echo err; else echo off; fi
}
do_login() {
  case "$1" in
    on)
      mkdir -p "$HOME/Library/LaunchAgents"
      cat > "$PLIST" <<PL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>com.homereality.menubar</string>
<key>ProgramArguments</key><array>
<string>/usr/bin/osascript</string><string>-l</string><string>JavaScript</string><string>$DIR/menubar.js</string>
</array>
<key>RunAtLoad</key><true/>
<key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
<key>ProcessType</key><string>Interactive</string>
<key>StandardErrorPath</key><string>$DIR/menubar.err.log</string>
</dict></plist>
PL
      ;;
    off) rm -f "$PLIST" ;;
    *) [ -f "$PLIST" ] && echo 1 || echo 0 ;;
  esac
}

case "$1" in
  start)   do_start "$2" ;;
  stop)    do_stop ;;
  status)  do_status ;;
  cleanup) touch "$STALEF"; apply_proxy off && rm -f "$STALEF" ;;
  # 1 = the Mac still points at our dead proxy (retried at most every 10 minutes, so no endless password prompts)
  stale)   if ! running && our_proxy_set && [ -z "$(find "$STALEF" -mmin -10 2>/dev/null)" ]; then echo 1; else echo 0; fi ;;
  copyuri) printf '%s' "$URI" | pbcopy ;;
  login)   do_login "$2" ;;
  *) echo "usage: $0 start [--core-only]|stop|status|cleanup|stale|copyuri|login on|off|status"; exit 1 ;;
esac
