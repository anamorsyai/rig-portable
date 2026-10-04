---
description: "Inspect or rotate the hunt autopilot ledger JSONL files (findings.jsonl, negatives.jsonl). Caps file size and keeps N rotated backups so memory does not grow unbounded. Self-contained — uses standard shell tools, no external engine. Usage: /memory-gc [--max-mb 10] [--rotate] [--purge-backups]"
---

# MEMORY-GC — $ARGUMENTS

Garbage-collect the hunt autopilot ledger. Reports current sizes, rotates
oversized files past a configurable cap, or purges old backups. This is a
housekeeping command for the hunt-state JSONL ledgers — independent of the
native autoreply memory store.

## Ledger location

`~/workspaces/hunts/<target>/state/` — the `findings.jsonl` and `negatives.jsonl`
written by the hunt state tracker. When run without a target, default to
`~/.claude/hunts/` (best-effort scan of all hunt ledgers).

## Usage

```
/memory-gc                          # report sizes only
/memory-gc --rotate                 # rotate files above 10 MB (default cap)
/memory-gc --rotate --max-mb 5      # custom cap
/memory-gc --purge-backups          # remove old *-backup-*.jsonl rotates
/memory-gc --dir <path>             # scan a non-default ledger dir
```

## Behavior

1. `find` every `findings.jsonl` / `negatives.jsonl` under the ledger dir.
2. Report each file's size in MB.
3. With `--rotate`: any file over the cap is moved to
   `<file>.backup-<ts>.jsonl` (dump line count first, then rotate), keeping at
   most `N=5` backups per ledger.
4. With `--purge-backups`: delete `*.backup-*` older than 30 days.

Rotation stays bounded on append inside the hunt state tracker itself; this
command is the operator-facing manual pass.

---

*Sourced from the claude-bughunter bundle; adapted to rig conventions. Does not
touch the native autoreply memory hooks or capture.sh.*
