#!/bin/bash
# Revert openclaude dist/cli.mjs to the stock base snapshot.
# Usage: ./revert.sh [path-to-cli.mjs]
set -e
CLI="${1:-/usr/local/lib/node_modules/@gitlawb/openclaude/dist/cli.mjs}"
BASE="$CLI.bak-compact"
[ -f "$BASE" ] || { echo "no $BASE found"; exit 1; }
cp "$BASE" "$CLI"
node --check "$CLI" && echo "reverted to stock base OK"
