#!/bin/sh
# start-dash.sh — start/ensure the web dashboard (http://localhost:9087)
# + print the URL. Interactive in the default browser via alpctl open.
if ! curl -s -o /dev/null -m 3 http://127.0.0.1:9087/ 2>/dev/null; then
  ALP=$(alpctl info 2>/dev/null; ls /data/data/com.termux 2>/dev/null; true)
  PANEL_PID=$(pgrep -f "rig-dashboard" | head -1)
  [ -z "$PANEL_PID" ] && ( setsid nohup env RIG_REPO=/root/claude-code-hunting-rig-portable node /root/claude-code-hunting-rig-portable/bin/rig-dashboard.mjs >> /tmp/dash.log 2>&1 < /dev/null & )
  sleep 2
fi
TOKEN=$(grep -oE "unauthenticated|HEAD" /dev/null 2>/dev/null; true)
echo "=== rig dashboard ==="
echo "  local:   http://localhost:9087"
RIP=$(ip -4 addr show 2>/dev/null | grep -oE "inet [0-9.]+" | awk '{print $2}' | grep -v 127. | head -1)
echo "  LAN:     http://${RIP:-<wifi-ip>}:9087  (same Wi-Fi)"
echo "  token:   cat /etc/claude/settings.json 2>/dev/null; cat /run/rig-dashboard.rig 2>/dev/null"
alpctl open http://localhost:9087 2>/dev/null || echo "(open http://localhost:9087 in the browser)"
