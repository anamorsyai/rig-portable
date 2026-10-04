#!/bin/sh
# setup-claude.sh — reproduce the official Claude Code + anthropic-shim stack from this mirror.
# Run on a fresh box:  sh claude-official/setup-claude.sh
set -e

RIG="$HOME/.openclaude"
SRC="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$SRC/.." && pwd)"

echo "[1/7] official Claude Code (npm global)"
command -v claude >/dev/null 2>&1 || npm install -g @anthropic-ai/claude-code
# musl/Alpine needs the optional native dep:
PKGDIR="$(npm root -g)/@anthropic-ai/claude-code"
[ "$(ldd --version 2>&1 | grep -ci musl)" != "0" ] && (cd "$PKGDIR" && npm install @anthropic-ai/claude-code-linux-x64-musl --no-save) || true
ln -sf "$PKGDIR/cli.js" /usr/local/bin/claude 2>/dev/null || true
hash -r; claude --version

echo "[2/7] settings.json -> $HOME/.claude/"
mkdir -p "$HOME/.claude"
cp "$SRC/settings.json" "$HOME/.claude/settings.json"

echo "[3/7] mcpServers -> $HOME/.claude.json"
python3 - "$SRC/claude-json.mcp.json" <<'PY'
import json, sys, os
p = os.path.expanduser("~/.claude.json")
cur = json.load(open(p)) if os.path.exists(p) else {}
add = json.load(open(sys.argv[1]))
# Merge per-server-name rather than replacing the whole "mcpServers" object —
# a wholesale replace is fine on a truly fresh box (cur starts empty) but
# silently wipes out any server added later via `claude mcp add` if this
# "fresh box" script is ever re-run on an already-configured one.
cur.setdefault("mcpServers", {}).update(add.get("mcpServers", {}))
cur.setdefault("hasCompletedOnboarding", True)
json.dump(cur, open(p, "w"), indent=2)
PY

echo "[4/7] rig symlinks (commands/skills/agents/plugins/CLAUDE.md)"
mkdir -p "$RIG"
for l in commands skills agents plugins CLAUDE.md; do
  [ -e "$RIG/$l" ] || { echo "  !! $RIG/$l missing — install openclaude rig first"; continue; }
  ln -sfn "$RIG/$l" "$HOME/.claude/$l"
done

echo "[5/7] anthropic-shim service"
install -m755 "$SRC/services/init.d-anthropic-shim" /etc/init.d/anthropic-shim
rc-update add anthropic-shim default 2>/dev/null || true
rc-service anthropic-shim restart 2>/dev/null || rc-service anthropic-shim start

echo "[6/7] bifrost (optional legacy gateway)"
if command -v bifrost-http-0 >/dev/null 2>&1 || [ -x "$HOME/.cache/bifrost/v1.6.11/bin/bifrost-http-0" ]; then
  install -m755 "$SRC/services/init.d-bifrost" /etc/init.d/bifrost
  rc-update add bifrost default 2>/dev/null || true
  sh "$SRC/bifrost-setup.sh" || echo "  !! bifrost config script failed (non-fatal)"
else
  echo "  skipped (binary not present)"
fi

echo "[7/7] verify"
for i in $(seq 1 10); do curl -s -m2 http://127.0.0.1:9086/ >/dev/null && break; sleep 1; done
curl -s -m3 http://127.0.0.1:9086/ && echo
timeout 90 claude -p "Reply with exactly: SETUP-CLAUDE-OK" && echo
echo "done."

# --- rig-models dashboard + runtime conf ---
if [ ! -f /etc/conf.d/anthropic-shim ]; then
  install -m 644 "$(dirname "$0")/anthropic-shim.conf.example" /etc/conf.d/anthropic-shim
  echo "installed /etc/conf.d/anthropic-shim (edit keys there)"
fi
ln -sf "$REPO/bin/rig-models" /usr/local/bin/rig-models
echo "rig-models installed — run: rig-models (dashboard) | rig-models list|chain|default|provider|test"
