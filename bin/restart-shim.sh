#!/bin/sh
# restart-shim.sh — safe shim restart (used by oc + the web dashboard).
# pkill pattern matches ONLY the real binary paths — never the invoking command
# (a broad `pkill -f anthropic-shim` matches the caller's own argv and kills the
# restart before it runs).
CONF=/etc/conf.d/anthropic-shim
for P in $(pgrep -f "anthropic-shim-arm64" 2>/dev/null; pgrep -f "claude-official/anthropic-shim" 2>/dev/null); do
  [ "$P" != "$$" ] && kill "$P" 2>/dev/null
done
sleep 1
[ -f "$CONF" ] && { set -a; . "$CONF" 2>/dev/null; set +a; }
B=/root/claude-code-hunting-rig-portable/claude-official/anthropic-shim-arm64
[ -x "$B" ] || B=/root/workspaces/rig-portable/claude-official/anthropic-shim
if [ -x "$B" ]; then
  setsid nohup "$B" >> /var/log/shim.log 2>&1 < /dev/null &
  sleep 2
  pgrep -f "anthropic-shim-arm64" 2>/dev/null | tail -1 > /run/anthropic-shim.pid
  echo "restarted (pid $(cat /run/anthropic-shim.pid 2>/dev/null))"
else
  echo "shim binary not found: $B"
  exit 1
fi
