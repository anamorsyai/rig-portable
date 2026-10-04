#!/bin/bash
# Apply the cumulative patch to openclaude's dist/cli.mjs. Idempotent.
# Usage: ./apply.sh [path-to-cli.mjs]
set -e
CLI="${1:-/usr/local/lib/node_modules/@gitlawb/openclaude/dist/cli.mjs}"
DIR="$(cd "$(dirname "$0")" && pwd)"

if grep -q 'COMPACT_MAX_OUTPUT_TOKENS=64000' "$CLI" \
   && grep -q 'error41 instanceof APIConnectionError' "$CLI" \
   && grep -q 'opencodeZenUserAgent' "$CLI" \
   && grep -q 'reasoning-only turn' "$CLI"; then
  echo "already patched — nothing to do"
  exit 0
fi

cp "$CLI" "$CLI.bak-pre-apply-$(date +%s)"
cd "$(dirname "$CLI")"
patch -p0 --forward < "$DIR/cli.mjs.patch"

echo "--- verify ---"
grep -q 'COMPACT_MAX_OUTPUT_TOKENS=64000' "$CLI" && echo "  OK  compaction caps (64k/600k)"
grep -q 'IMPORTANT-BOUND' "$CLI" && echo "  OK  IMPORTANT-BOUND marker"
grep -q 'error41 instanceof APIConnectionError' "$CLI" && echo "  OK  persistent retry (429/408/409/5xx/529)"
grep -q 'stream-integrity\|eof_without_terminal' "$CLI" && echo "  OK  stream-integrity gate"
grep -q 'reasoning_content' "$CLI" && echo "  OK  reasoning->thinking adapter"
grep -q 'opencodeZenUserAgent' "$CLI" && echo "  OK  opencode UA override (Zen free tier gate)"
grep -q 'reasoning-only turn' "$CLI" && echo "  OK  empty-reply guard x3 sites (stream tail, JSON branch, non-streaming converter)"
echo "patched OK"
