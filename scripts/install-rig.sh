#!/bin/sh
# install-rig.sh — universal installer for the Claude hunting rig.
#
# Pipeable one-liner (the way the AlpineTerm button runs it):
#   wget -qO- --header="Accept: application/vnd.github.raw" \
#     https://api.github.com/repos/<owner>/<repo>/contents/scripts/install-rig.sh | sh
#
# Also works from inside a checkout:  sh scripts/install-rig.sh
#
# What it does: fetches the latest repo tree (tarball of the default branch, so every
# pushed update installs automatically), restores ~/.claude (skills/commands/agents/
# settings), deploys the shim config, builds/uses the shim binary, installs oc/rig-models,
# starts the proxy on 127.0.0.1:9086 and verifies it answers.
#
# Env overrides:  RIG_REPO=owner/name   (default below)
set -eu
RIG_REPO="${RIG_REPO:-anamorsyai/rig-portable}"
say(){ printf '\n\033[36m== %s\033[0m\n' "$1"; }

# ---- locate (or fetch) the repo tree -------------------------------------
R=""
if [ -n "${0:-}" ] && [ -f "$0" ]; then
  R="$(cd "$(dirname "$0")" && pwd)"
  while [ "$R" != "/" ] && [ ! -d "$R/claude-home" ]; do R="$(dirname "$R")"; done
  [ -d "$R/claude-home" ] || R=""
fi
if [ -z "$R" ]; then
  say "0. fetching latest rig tree ($RIG_REPO)"
  TMP="$(mktemp -d /tmp/rig-install.XXXXXX)"
  trap 'rm -rf "$TMP"' EXIT INT TERM
  wget -qO "$TMP/rig.tar.gz" "https://codeload.github.com/$RIG_REPO/tar.gz/refs/heads/$(wget -qO- "https://api.github.com/repos/$RIG_REPO" | sed -n 's/.*"default_branch": *"\([^"]*\)".*/\1/p' | head -1)" 2>/dev/null \
    || wget -qO "$TMP/rig.tar.gz" "https://codeload.github.com/$RIG_REPO/tar.gz/refs/heads/main"
  tar -xzf "$TMP/rig.tar.gz" -C "$TMP"
  R="$(find "$TMP" -mindepth 1 -maxdepth 1 -type d | head -1)"
  [ -d "$R/claude-home" ] || { echo "ERROR: fetched tree has no claude-home/"; exit 1; }
  echo "got $(find "$R" -type f | wc -l) files (latest)"
fi

# ---- dependency check ----------------------------------------------------
say "1. dependency check"
MISSING=""
for c in node curl jq python3 git; do command -v $c >/dev/null || MISSING="$MISSING $c"; done
[ -n "$MISSING" ] && { echo "installing:$MISSING"; apk add --no-cache $MISSING >/dev/null 2>&1 || true; }
for c in node curl jq python3 git; do
  command -v $c >/dev/null || { echo "MISSING: $c (install it, re-run)"; exit 1; }
done
echo "ok"

# ---- claude CLI (musl build) ---------------------------------------------
say "2. claude CLI"
if command -v claude >/dev/null 2>&1; then
  echo "claude $(claude --version 2>/dev/null | head -1)"
else
  command -v npm >/dev/null 2>&1 || apk add --no-cache nodejs npm >/dev/null 2>&1
  npm install -g @anthropic-ai/claude-code @anthropic-ai/claude-code-linux-arm64-musl >/dev/null 2>&1 \
    || npm install -g @anthropic-ai/claude-code >/dev/null 2>&1
  MUSL=/usr/local/lib/node_modules/@anthropic-ai/claude-code-linux-arm64-musl/claude
  if [ -f "$MUSL" ]; then
    ln -sf "$MUSL" /usr/local/bin/claude
  elif [ -f /usr/local/lib/node_modules/@anthropic-ai/claude-code/cli-wrapper.cjs ]; then
    printf '#!/bin/sh\nexec node /usr/local/lib/node_modules/@anthropic-ai/claude-code/cli-wrapper.cjs "$@"\n' > /usr/local/bin/claude
    chmod +x /usr/local/bin/claude
  fi
  command -v claude >/dev/null 2>&1 && echo "installed: $(claude --version 2>/dev/null | head -1)" || echo "WARN: claude missing"
fi

# ---- ~/.claude home -------------------------------------------------------
say "3. restore ~/.claude home (skills/commands/agents/settings — fresh)"
mkdir -p "$HOME/.claude"
cp -a "$R/claude-home/." "$HOME/.claude/"
for d in projects memory/inbox todos jobs file-history shell-snapshots session-env paste-cache daemon backups; do
  mkdir -p "$HOME/.claude/$d"
done
echo "restored $(find "$HOME/.claude" -type f | wc -l) files"

# ---- zen free-tier ids (auto-provisioned) ---------------------------------
# zen requires (session_id, project_id) created by opencode itself + opencode's
# system-prompt fingerprint. The sysprompt file ships in this repo; the ids are
# grabbed once by running a real opencode generation ( chin) if missing:
say "0b. zen free-tier ids"
if [ -f /etc/conf.d/zen-ids.conf ]; then
  echo "kept existing zen ids"
else
  if command -v opencode >/dev/null 2>&1; then
    echo "provisioning zen ids via a real opencode generation..."
    TMPDIR_OC="$(mktemp -d)"
    ( cd "$TMPDIR_OC" && opencode run -m opencode/big-pickle "hi" >/dev/null 2>&1 )
    SID=$(grep -aoE "x-opencode-session: +ses_[A-Za-z0-9]+" ~/.local/share/opencode/log/opencode.log 2>/dev/null | tail -1 | awk '{print $3}')
    PROJ=$(grep -aoE "x-opencode-project: +[a-f0-9]{40}" ~/.local/share/opencode/log/opencode.log 2>/dev/null | tail -1 | awk '{print $3}')
    if [ -n "$SID" ] && [ -n "$PROJ" ]; then
      printf "# auto-provisioned %s\nexport ZEN_SESSION_ID='%s'\nexport ZEN_PROJECT_ID=\"%s\"\nexport ZEN_SYSPROMPT_FILE=\"/etc/conf.d/zen-sysprompt.txt\"\n" "$(date -u +%FT%TZ)" "$SID" "$PROJ" > /etc/conf.d/zen-ids.conf
      chmod 600 /etc/conf.d/zen-ids.conf
      echo "zen ids provisioned ✓ ($SID)"
    else
      echo "WARN: could not auto-provision zen ids — free zen models won't work until added to /etc/conf.d/zen-ids.conf"
    fi
    rm -rf "$TMPDIR_OC"
  else
    echo "WARN: opencode not installed yet — re-run after installing opencode to provision zen ids"
  fi
fi

# ---- shim config ----------------------------------------------------------
say "4. shim config -> /etc/conf.d/anthropic-shim"
mkdir -p /etc/conf.d
if [ -f "$R/secrets/anthropic-shim.conf" ]; then
  cp "$R/secrets/anthropic-shim.conf" /etc/conf.d/anthropic-shim
  chmod 600 /etc/conf.d/anthropic-shim
  echo "deployed from repo (keys included — repo must stay PRIVATE)"
elif [ -f /etc/conf.d/anthropic-shim ]; then
  echo "kept existing /etc/conf.d/anthropic-shim (repo copy has no keys — public build)"
elif [ -f "$R/claude-official/anthropic-shim.conf.example" ]; then
  cp "$R/claude-official/anthropic-shim.conf.example" /etc/conf.d/anthropic-shim
  chmod 600 /etc/conf.d/anthropic-shim
  echo "deployed EXAMPLE config — add your provider keys via: oc -k  (or edit /etc/conf.d/anthropic-shim)"
fi
# zen free-tier ids + opencode system-prompt fingerprint (multi-line file)
if [ -f "$R/secrets/zen-ids.conf" ]; then
  [ -f /etc/conf.d/zen-ids.conf ] || cp "$R/secrets/zen-ids.conf" /etc/conf.d/zen-ids.conf
  echo "zen ids: kept existing or deployed"
fi
if [ -f "$R/claude-official/zen-sysprompt.txt" ]; then
  cp "$R/claude-official/zen-sysprompt.txt" /etc/conf.d/zen-sysprompt.txt
  echo "zen sysprompt fingerprint deployed"
fi

# ---- shim binary ----------------------------------------------------------
say "5. shim binary ($(uname -m))"
SHIM_BIN=""
if [ -x "$R/claude-official/anthropic-shim-$(uname -m | sed 's/x86_64/x86_64/;s/aarch64/arm64/')" ]; then
  SHIM_BIN="$R/claude-official/anthropic-shim-$(uname -m | sed 's/aarch64/arm64/')"
  echo "using prebuilt binary"
elif [ -x "$R/claude-official/anthropic-shim" ] && [ "$(uname -m)" = "x86_64" ]; then
  SHIM_BIN="$R/claude-official/anthropic-shim"
  echo "using prebuilt x86_64 binary"
else
  command -v go >/dev/null 2>&1 || apk add --no-cache go >/dev/null 2>&1
  if command -v go >/dev/null 2>&1; then
    ( cd "$R/claude-official" && go build -o anthropic-shim-built anthropic-shim.go ) && SHIM_BIN="$R/claude-official/anthropic-shim-built" \
      && echo "built from source" || echo "WARN: go build failed"
  else
    echo "WARN: no go toolchain — cannot build shim"
  fi
fi

# ---- control tools --------------------------------------------------------
say "6. control tools -> /usr/local/bin"
for b in oc rig-models; do
  [ -f "$R/bin/$b" ] && { cp "$R/bin/$b" /usr/local/bin/$b; chmod +x /usr/local/bin/$b; echo "  $b ok"; }
done

# ---- MCP config -----------------------------------------------------------
say "7. MCP config"
[ -f "$R/claude-official/claude-json.mcp.json" ] && cp "$R/claude-official/claude-json.mcp.json" "$HOME/.claude.json" && echo "  .claude.json deployed"
[ -f "$R/claude-official/settings.local.json" ] && cp "$R/claude-official/settings.local.json" "$HOME/.claude/settings.local.json" && echo "  settings.local.json deployed"

# ---- start shim -----------------------------------------------------------
say "8. start tls-relay + shim"
mkdir -p /var/log /var/cache/anthropic-shim /run
# tls-relay first: node TLS passes zen's fingerprint gate (go's crypto/tls gets 403)
if command -v node >/dev/null 2>&1 && [ -f "$R/claude-official/tls-relay.mjs" ]; then
  if ! pkill -0 -f tls-relay.mjs 2>/dev/null; then
    ( setsid nohup node "$R/claude-official/tls-relay.mjs" >> /var/log/tls-relay.log 2>&1 < /dev/null & )
    sleep 2
    echo "tls-relay started on :9097 (log: /var/log/tls-relay.log)"
  else
    echo "tls-relay already running"
  fi
fi
if [ -n "$SHIM_BIN" ] && [ -x "$SHIM_BIN" ]; then
  if curl -s -o /dev/null -m 3 http://127.0.0.1:9086/v1/models 2>/dev/null; then
    echo "shim already alive on :9086"
  else
    ( set -a; . /etc/conf.d/anthropic-shim 2>/dev/null; set +a
      nohup "$SHIM_BIN" >> /var/log/shim.log 2>&1 & echo $! > /run/anthropic-shim.pid )
    sleep 2
    code=$(curl -s -o /dev/null -m 10 -w '%{http_code}' http://127.0.0.1:9086/v1/models || echo 000)
    [ "$code" = "200" ] && echo "shim :9086 alive" || echo "WARN: shim not responding (HTTP $code) — tail /var/log/shim.log"
  fi
else
  echo "WARN: no shim binary — proxy won't start"
fi

say "done"
echo "  launch with: claude"
echo "  panel: oc | models: rig-models | logs: /var/log/shim.log"
