---
description: Manage OpenClaude sessions — list, view, resume, or delete (trashable). Usage: /sessions [list|all|dirs|view <id>|resume <id>|rm <id>|trash|restore <id>|purge]
---

# Session Manager

**Execute the bundled `oc-sess` tool via the Bash tool with the user's arguments and
present its output verbatim. Do not summarize, re-format, or ask follow-up questions.**

Transcripts live at `~/.openclaude/projects/<sanitized-cwd>/<uuid>.jsonl` (+ `.replay.json`
summaries). There is no built-in list/delete command, so `oc-sess` is the manager.

- `/sessions` or `/sessions list` — sessions in the current directory (numbered table)
- `/sessions all` — sessions across every project directory
- `/sessions dirs` — per-directory counts
- `/sessions view <id|#N>` — print the conversation (users, tool calls, retries)
- `/sessions resume <id|#N>` — resume via `oc -r <id>`. If resume says "No conversation
  found", the session was created from a different directory; rerun from there.
- `/sessions rm <id|#N>` — move a session to the trash (reversible; prompts unless `-y`)
- `/sessions trash` — list trashed sessions
- `/sessions restore <id>` — restore a session from the trash
- `/sessions purge` — permanently delete the trash (destructive; confirm)

Map `<id|#N>` from the numbered list `oc-sess` prints (e.g. `/sessions view 2`).

Safety rules:

- **Never target the currently active session** (this conversation) unless the user
  explicitly confirms. `rm` on a live session would trash its transcript mid-write.
- `rm` is reversible (trash); `purge` is not — show what `purge` will delete first.
- If a session id is ambiguous, run `oc-sess list` in the relevant project dir first.