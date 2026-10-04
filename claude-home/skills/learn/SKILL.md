---
name: learn
description: Auto-memory curation. Reads pending captures in ~/.claude/memory/inbox/, extracts durable learnings (user preferences, feedback, project context, references), dedupes against existing memory, and writes them to the project memory store (MEMORY.md + typed files). Run whenever a session start mentions pending captures, after compaction, or when the user says "remember", "learn", or "store that".
category: memory
---

# Learn — Inbox Curation

Purpose: turn raw session captures (transcripts snapshotted by the PreCompact/Stop
hooks) into durable, retrievable memories. This is the "store" half of auto-memory.

## Input

- `~/.claude/memory/inbox/` — zero or more files:
  - `precompact-<ts>-*.jsonl` — transcript snapshot taken at context compaction
  - `stop-<ts>-*.jsonl` — transcript snapshot taken at session end
- Existing memory: read the project memory store before writing anything to dedupe.

## Locate the memory store

- Project memory root: `~/.claude/projects/<sanitized-cwd>/memory/`
  where `<sanitized-cwd>` = `$PWD` with every `/` replaced by `-` (e.g. `/root` → `-root`).
- Create the store dir (and `team/` subdir) with `mkdir -p` if it does not exist.
- Structure per store: `MEMORY.md` (index, one short line per entry, ≤200 lines / ≤24KB)
  plus one typed markdown file per memory (`user/*`, `feedback/*`, `project/*`,
  `reference/*` are semantic folders; flat files are fine too). Team-shared memories
  live in `memory/team/`.

## Procedure

1. **List** the inbox; process files oldest-first. Read each capture fully.
2. **Extract** durable facts only. Ask for each candidate: *"is this still true in a
   month, and is it useful to Me/other sessions?"* Reject ephemeral task details
   (in-progress work, resolved bugs, command transcripts) — those live in the transcript,
   not memory.
3. **Dedupe** against the existing store: read `MEMORY.md` and the memory files. Skip
   anything already captured, or update the existing file instead of writing a new one.
4. **Classify** each accepted item using the harness memory types:
   - `user` — who the user is, role, goals, knowledge, collaboration style (always private)
   - `feedback` — guidance on how to work: corrections AND validated approaches, with Why/How-to-apply
   - `project` — ongoing work, goals, decisions, who/why/buckets (default team)
   - `reference` — pointers to external systems/dashboards/places to look (usually team)
   Special rules: never save secrets/PII/credentials in team memory; convert relative
   dates ("Thursday") to absolute dates before saving.
5. **Write**: one file per memory in the store directory + append a single concise line
   to that store's `MEMORY.md` (keep existing entries; never write memory content inline
   in MEMORY.md). Frontmatter format:
   ```
   ---
   name: <slug>
   description: <one-line, specific — drives future relevance matching>
   type: <user|feedback|project|reference>
   ---
   <body>
   ```
   For feedback/project bodies: lead with the rule/fact, then **Why:** and
   **How to apply:** lines.
6. **Archive**: after successful write, move the processed capture to
   `~/.claude/memory/archive/` (keep the filename; never delete human-visible
   history). Leave untouched anything you could not curate.
7. **Report**: list what you saved (type + file), what you skipped and why, and how many
   captures remain in the inbox.

## Guardrails

- Do not bloat: a session that produced one durable fact saves one memory, not ten.
- Do not duplicate MEMORY.md entries; update in place when a memory evolves.
- If a capture contains contradictory or ambiguous material, skip it and note why.
- Never write potential credentials, API keys, or real user PII into memory files.