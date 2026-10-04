# Claude Code CLI — Flag Reference (v2.1.238)

Curated from `claude --help`. ★ = especially useful in this rig.

## Session & resume
| Flag | Info |
|---|---|
| ★ `-p, --print` | Non-interactive: print response, exit. For pipes/scripts. |
| ★ `-r, --resume [id]` | Resume session by ID (or interactive picker). |
| ★ `-c, --continue` | Continue most recent conversation in cwd. |
| `--session-id <uuid>` | Force a specific session ID. |
| `--fork-session` | With resume/continue: new session ID instead of reusing. |
| `-n, --name <name>` | Display name for session (prompt box, /resume picker). |
| `--no-session-persistence` | Don't save session (print mode only). |
| `--from-pr [value]` | Resume session linked to a PR number/URL. |
| `--teleport [session]` | Resume a teleport session. |
| `-w, --worktree [name]` | Run in a fresh git worktree. |
| `--tmux` | Worktree inside tmux session. |
| ★ `--bg, --background` | Start as background agent, return immediately (`claude agents` manages). |

## Model & behavior
| Flag | Info |
|---|---|
| ★ `--model <m>` | Session model alias or full name (we default big-pickle via env). |
| `--fallback-model <list>` | CC-native model failover, comma list (**print mode only**; our shim chains for all modes too). |
| `--effort <level>` | low/medium/high/xhigh/max reasoning effort. |
| `--autocompact <auto\|tokens>` | Auto-compact window (100k–1M). |
| `--system-prompt <p>` | Replace default system prompt. |
| ★ `--append-system-prompt <p>` | Append to default system prompt (keeps CC built-ins). |
| `--betas <b...>` | API beta headers (API-key auth only). |
| `--json-schema <s>` | Structured output validation schema (print mode). |
| `--max-budget-usd <n>` | Spend cap per print run. |

## Tools & permissions
| Flag | Info |
|---|---|
| ★ `--allowedTools <t...>` | Allow-list tools for this run (e.g. `"Bash(git *)"`). |
| ★ `--disallowedTools <t...>` | Deny-list tools. |
| `--tools <t...>` | Restrict built-in toolset ("", "default", or names). |
| `--permission-mode <m>` | acceptEdits/auto/bypassPermissions/manual/dontAsk/plan. |
| ★ `--dangerously-skip-permissions` | Bypass ALL checks — sandboxed boxes only (this box qualifies but keep off by default). |
| `--allow-dangerously-skip-permissions` | Merely *enables* the bypass option without activating. |

## MCP / plugins / settings
| Flag | Info |
|---|---|
| ★ `--mcp-config <files...>` | Extra MCP servers from JSON files/strings. |
| `--strict-mcp-config` | Ignore all other MCP sources, use only --mcp-config. |
| ★ `--settings <file\|json>` | Extra settings layer for this run. |
| `--setting-sources <srcs>` | Which layers load: user,project,local. |
| ★ `--plugin-dir <path>` | Load plugin dir/zip this session (repeatable). |
| `--plugin-url <url>` | Fetch plugin zip from URL. |
| `--agents <json>` | Inline agent definitions (name→{description,prompt}). |

## I/O & streaming (SDK-style automation)
| Flag | Info |
|---|---|
| ★ `--output-format text\|json\|stream-json` | json = single result incl. session_id/cost; stream-json = realtime events. |
| `--input-format text\|stream-json` | stream-json enables realtime streaming input. |
| `--include-partial-messages` | Emit partial deltas (stream-json). |
| `--include-hook-events` | Hook lifecycle events in stream. |
| `--replay-user-messages` | Echo stdin user messages back (ack). |
| `--forward-subagent-text` | Subagent text/thinking into parent stream (stream-json). |
| `--prompt-suggestions` | Emits predicted next prompt after each turn. |
| ★ `-d, --debug [filter]` | Debug logs w/ category filter ("api,hooks"). |
| `--debug-file <path>` | Debug log to file. |
| `--verbose` | Force verbose. |

## Safety / isolation / misc
| Flag | Info |
|---|---|
| ★ `--safe-mode` | Disable ALL customizations (skills/hooks/MCP/CLAUDE.md…) — first aid for broken config. |
| `--bare` | Minimal mode: no hooks/LSP/plugins/memory; explicit context only. |
| `--disable-slash-commands` | Turn off skills/slash commands. |
| `--exclude-dynamic-system-prompt-sections` | Move machine-specific sysprompt bits into first user msg (cache reuse). |
| `--append-system-prompt-file` / `--system-prompt-file` | Same as above but from files (implied by docs of bare mode). |
| `--file <id:path...>` | Download file resources at startup. |
| `--ide` | Auto-connect IDE if exactly one available. |
| `--chrome` / `--no-chrome` | Claude-in-Chrome integration. |
| `--cloud [desc\|id\|url]`, `--environment <id>` | Cloud sessions / self-hosted env. |
| `--remote-control [name]` | Interactive Remote Control session. |
| `--ax-screen-reader` | Accessibility-friendly flat output. |
| `--brief` | Enable SendUserMessage agent→user channel. |
| `-h/--help`, `-v/--version` | Meta. |

## Subcommands
`agents` (background agents) · `auth` · `doctor` (health check) · `mcp` (server config) ·
`plugin[s]` · `project` · `import` (migrate from other agents) · `install [ver]` ·
`update` · `setup-token` · `gateway` (enterprise) · `ultrareview` (cloud code review) ·
`auto-mode` (classifier inspect/reset)

## Rig notes
- Env defaults already set in ~/.claude/settings.json: ANTHROPIC_BASE_URL=:9086 shim,
  ANTHROPIC_MODEL=big-pickle, CLAUDE_CODE_MAX_CONTEXT_TOKENS=262144.
- Model failover lives in the shim (FALLBACK_CHAIN, all modes) — CC's --fallback-model
  is redundant here but harmless if passed.
