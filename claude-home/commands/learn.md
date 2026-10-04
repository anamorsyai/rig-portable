---
description: "Curate pending auto-memory captures into the memory store. Usage: /learn"
---

# LEARN — Curate Memory Inbox

Load the `learn` skill and execute it end-to-end:

1. Scan `~/.claude/memory/inbox/` for pending captures.
2. Read each capture, extract durable learnings (user, feedback, project, reference).
3. Dedupe against the existing memory store and write new/updated memory files + `MEMORY.md` lines.
4. Move processed captures to `~/.claude/memory/archive/`.
5. Report: saved (type + file), skipped (why), remaining inbox count.

If the inbox is empty, say so and stop — do not manufacture memories.