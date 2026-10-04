#!/usr/bin/env bash
#
# capture.sh — auto-memory capture half.
#
# Snapshots the currently-active Claude Code session transcript whenever a hook
# fires (Stop / PreCompact / SessionStart) and stages the byte-delta into
# ~/.claude/memory/inbox/ for the `learn` skill to curate into the native
# memory store (~/.claude/projects/<sanitized-cwd>/memory/).
#
# Design:
#   - The transcript is the JSONL at $CLAUDE_PROJECT_DIR/<session_id>.jsonl
#   - We track per-session byte offsets in ~/.claude/memory/.capture-state
#   - Each run appends ONLY the bytes after the last seen offset, so we never
#     duplicate and never lose a tail that a crash might have skipped.
#   - Hook input is on stdin as JSON: {"session_id": "...", "transcript_path": "...", ...}
#   - Never fails the hook (best-effort: any error is logged, exit 0).

MEM=~/.claude/memory
INBOX="$MEM/inbox"
STATE="$MEM/.capture-state"
LOG="$MEM/logs/capture.log"
mkdir -p "$INBOX" "$MEM/logs"

echo "capture.sh: hook=$CLAUDE_HOOK_TYPE stdin=$(cat 2>/dev/null | jq -c . 2>/dev/null | head -c 400)" >> "$LOG" 2>/dev/null

# enable aliases/expansions used below; this is non-interactive so just be explicit
# lock to avoid concurrent hook runs clobbering .capture-state
LOCK="$MEM/.capture.lock"
exec 9>"$LOCK"
if ! flock -n 9; then
  echo "capture.sh: another run holds the lock; skipping" >> "$LOG"
  exit 0
fi

# --- resolve transcript path ---
session_id=""
transcript=""
if [ -n "$CLAUDE_PROJECT_DIR" ]; then
  transcript="$CLAUDE_PROJECT_DIR"
fi
# prefer the hook-provided transcript path / session id
if [ -n "$CLAUDE_HOOK" ]; then :; fi
# parse stdin (optional) for session_id / transcript_path
if [ -t 0 ]; then
  :
else
  stdin="$(cat 2>/dev/null)"
  if [ -n "$stdin" ]; then
    sid=$(printf '%s' "$stdin" | jq -r '.session_id // empty' 2>/dev/null)
    tpath=$(printf '%s' "$stdin" | jq -r '.transcript_path // empty' 2>/dev/null)
    [ -n "$sid" ] && session_id="$sid"
    [ -n "$tpath" ] && transcript="$tpath"
  fi
fi

# fallback: find the most recently modified session transcript under ~/.claude/projects
if [ -z "$transcript" ] || [ ! -f "$transcript" ]; then
  transcript="$(find ~/.claude/projects -name '*.jsonl' -mmin -5 2>/dev/null | head -1)"
fi
# if CLAUDE_PROJECT_DIR is a file use it; if it's a dir, nothing more to do
[ -f "$transcript" ] || transcript=""
[ -z "$transcript" ] && { echo "capture.sh: no transcript resolved ($CLAUDE_PROJECT_DIR)" >> "$LOG"; exit 0; }

# --- byte-delta capture ---
size=$(stat -c %s "$transcript" 2>/dev/null || echo 0)
key="$(readlink -f "$transcript")"
seen=0
if [ -f "$STATE" ]; then
  seen=$(awk -v k="$key" '$1==k{print $2; exit}' "$STATE" 2>/dev/null)
fi
seen=${seen:-0}

# This hook fires on Stop too; capture the whole file if we've never seen it.
if [ "$size" -gt "$seen" ]; then
  stamp=$(date +%Y%m%d-%H%M%S)
  hook=${CLAUDE_HOOK_TYPE:-manual}
  out="$INBOX/${hook}-${stamp}-$$.jsonl"
  # tail from the last-seen byte (or whole file the first time)
  if [ "$seen" -gt 0 ] && [ "$seen" -lt "$size" ]; then
    dd if="$transcript" bs=1 skip="$seen" status=none 2>/dev/null > "$out"
  else
    cp "$transcript" "$out"
  fi
  # update state
  awk -v k="$key" -v s="$size" '$1!=k{print}' "$STATE" > "$STATE.tmp" 2>/dev/null
  printf '%s\t%s\n' "$key" "$size" >> "$STATE.tmp"
  mv "$STATE.tmp" "$STATE" 2>/dev/null
  echo "capture.sh: $hook captured $((size-seen))B ($key) -> $out" >> "$LOG"
else
  echo "capture.sh: $(basename "$key") unchanged ($size<=$seen), no capture" >> "$LOG"
fi

flock -u 9 2>/dev/null
exit 0
