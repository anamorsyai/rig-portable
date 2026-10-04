#!/bin/sh
# rig-portable — new-device bootstrap
# Restores the full hunting rig: ~/.claude home, shim proxy + service, oc/rig-models tools.
# Usage: sh install-new-device.sh
set -e
R="$(cd "$(dirname "$0")" && pwd)"
say(){ printf '\n\033[36m== %s\033[0m\n' "$1"; }

say "0. dependency check"
for c in node curl jq python3 git; do command -v $c >/dev/null || { echo "MISSING: $c (install it, re-run)"; exit 1; }; done
command -v claude >/dev/null || echo "NOTE: claude CLI not found — install it before first launch"
echo "ok"

say "1. restore ~/.claude home (skills/commands/agents/settings — fresh, no session data)"
mkdir -p "$HOME/.claude"
cp -a "$R/claude-home/." "$HOME/.claude/"
# seed clean runtime dirs so the CLI starts with a pristine slate
for d in projects memory/inbox todos jobs file-history shell-snapshots session-env paste-cache daemon backups; do
  mkdir -p "$HOME/.claude/$d"
done
echo "restored $(find "$HOME/.claude" -type f | wc -l) files (fresh state)"

say "2. deploy shim config -> /etc/conf.d/anthropic-shim"
if [ -f /etc/conf.d/anthropic-shim ] && ! [ -w /etc/conf.d ]; then
  echo "need root to write /etc/conf.d — re-run with sudo"; exit 1
fi
mkdir -p /etc/conf.d
cp "$R/secrets/anthropic-shim.conf" /etc/conf.d/anthropic-shim
chmod 600 /etc/conf.d/anthropic-shim
echo "deployed (real keys included — keep this repo PRIVATE)"

say "3. install shim service (Go binary)"
mkdir -p "$R/services" 2>/dev/null || true
if command -v go >/dev/null 2>&1; then
  # prefer prebuilt binary; build fresh if missing so a device needs no Go toolchain to run
  if [ ! -x "$R/claude-official/anthropic-shim" ]; then
    ( cd "$R/claude-official" && go build -o anthropic-shim anthropic-shim.go ) || echo "WARN: go build failed — copy a prebuilt anthropic-shim binary"
  fi
fi
[ -x "$R/claude-official/anthropic-shim" ] || echo "WARN: no anthropic-shim binary — shim will fail to start"
if [ -d /etc/init.d ]; then
  cp "$R/claude-official/services/init.d-anthropic-shim" /etc/init.d/anthropic-shim
  chmod +x /etc/init.d/anthropic-shim
  rc-update add anthropic-shim default 2>/dev/null || true
  rc-service anthropic-shim restart && echo "service started"
else
  echo "no OpenRC detected — start manually:"
  echo "  set -a; . /etc/conf.d/anthropic-shim; set +a; $R/claude-official/anthropic-shim &"
fi

# log rotation for the shim (size-based, via periodic)
mkdir -p /etc/periodic/daily
cp "$R/claude-official/services/shim-logrotate" /etc/periodic/daily/shim-logrotate
chmod +x /etc/periodic/daily/shim-logrotate
echo "shim log rotation installed"

say "4. install control tools -> /usr/local/bin"
for b in oc rig-models; do
  [ -f "$R/bin/$b" ] && { cp "$R/bin/$b" /usr/local/bin/$b 2>/dev/null || sudo cp "$R/bin/$b" /usr/local/bin/$b; chmod +x /usr/local/bin/$b; echo "  $b ✓"; }
done

say "5. deploy .claude.json MCP config + settings.local.json"
if [ -f "$R/claude-official/claude-json.mcp.json" ]; then
  # merge MCP servers into existing .claude.json (or create fresh)
  if [ -f "$HOME/.claude.json" ]; then
    jq -s '.[0] * .[1]' "$HOME/.claude.json" "$R/claude-official/claude-json.mcp.json" > "$HOME/.claude.json.tmp" && mv "$HOME/.claude.json.tmp" "$HOME/.claude.json"
    echo "merged MCP servers into existing .claude.json"
  else
    cp "$R/claude-official/claude-json.mcp.json" "$HOME/.claude.json"
    echo "created new .claude.json with MCP servers"
  fi
fi
if [ -f "$R/claude-official/settings.local.json" ]; then
  cp "$R/claude-official/settings.local.json" "$HOME/.claude/settings.local.json"
  echo "deployed settings.local.json (MCP enabled servers)"
fi

say "5. verify proxy"
sleep 1
code=$(curl -s -o /dev/null -m 10 -w '%{http_code}' http://127.0.0.1:9086/v1/models)
[ "$code" = "200" ] && echo "shim :9086 alive ✓" || echo "WARNING: shim not responding (HTTP $code) — check: rc-service anthropic-shim status; tail /var/log/shim.log"

say "6. done — expected behavior after clone+run"
echo "  claude launches with big-pickle primary via local shim :9086"
echo "  thinking visible, failover: zen -> bai -> google -> cline -> groq"
echo "  UA: opencode/1.18.27 + X-Opencode-Session header per-request"
echo "  sessions from projects/ resumable; MCP servers burp/caido/playwright/browser-control"
