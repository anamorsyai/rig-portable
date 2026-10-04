# OpenClaude Rig — Portable Setup (v2, post-tmpfs-wipe rebuild)

Lives at `/root/workspaces/rig-portable` (persistent disk — NEVER keep in /tmp;
a reboot on 2026-08-21 19:27 wiped the previous copy from tmpfs).

## Two runtimes coexist

### 1. `openclaude` (fork, patched core) — entry: `/usr/local/bin/oc`
- Binary: `/usr/local/lib/node_modules/@gitlawb/openclaude/dist/cli.mjs`
- Stock base snapshot: same dir, `cli.mjs.bak-compact`
- Cumulative patch (9 feature groups) + apply/revert: `core-patch/`
  - compaction caps 64k/600k, IMPORTANT-BOUND, persistent retry
    (429/408/409/5xx/529), stream-integrity gate, reasoning→thinking adapter,
    violet thinking tag, Zen UA override, **empty-reply guard at 3 sites**
    (stream tail / JSON branch / non-streaming converter) + whitespace-proofing
- `bin/oc`: wrapper exporting Zen env (`OPENAI_BASE_URL/KEY/MODEL`),
  `-lm` model probe (nc-based), `--fallback-model DeepSeek-V4-Flash`
- `bin/oc-sess`: session manager (list/view/resume with cross-dir cd/delete)
- `commands-sessions.md`: imperative `/sessions` command
- Restore: `cp bin/* /usr/local/bin/ && chmod +x`, then `core-patch/apply.sh`

### 2. Official Claude Code + Bifrost gateway — entry: `claude` (no env needed)
- `claude` CLI: npm global `@anthropic-ai/claude-code@2.1.238`
  (musl box → needs optional dep `claude-code-linux-x64-musl`; if missing:
  `cd <pkgdir> && npm install @anthropic-ai/claude-code-linux-x64-musl --no-save`)
- Bifrost v1.6.11: binary `/root/.cache/bifrost/v1.6.11/bin/bifrost-http-0`,
  service `/etc/init.d/bifrost` (port 9084), config SQLite `~/.config/bifrost/`
- Upstream: **Zen speaks NATIVE Anthropic** at `https://opencode.ai/zen/v1/messages`
  (array-form content required; UA-gated)
- Bifrost openai-provider config (via API):
  - `network_config.base_url = https://opencode.ai/zen` (NO `/v1` — bifrost appends it)
  - `network_config.extra_headers.User-Agent = opencode/1.18.18 ai-sdk/provider-utils/4.0.23 runtime/bun/1.3.14` (Zen gates hard on this)
  - key `zen` value **`public`** (Zen free-tier literal — never a real key) with models: big-pickle, moonshotai/Kimi-K2.6,
    deepseek-ai/DeepSeek-V4-Flash-0731, mimo-v2.5-free, nemotron-3-ultra-free,
    nemotron-3.5-lightning-free, hy3-free; `use_anthropic_endpoints=true`
- Claude Code config: `claude-official/settings.json` → `~/.claude/settings.json`
  (env ANTHROPIC_BASE_URL=http://127.0.0.1:9084/anthropic, model big-pickle,
  context 262144, permissions + memory hooks replicated)
- MCP servers (`claude-official/mcp-servers.json`) → `~/.claude.json` top-level
- Rig dirs symlinked (single source of truth in ~/.openclaude):
  `~/.claude/{commands,skills,agents,plugins,CLAUDE.md} -> ~/.openclaude/*`

## Known limitations (official CC path)
- Bifrost's `/anthropic` ingress emits minimal SSE (no message_start /
  content_block_start); official CC tolerates it. Thinking blocks are NOT
  forwarded by bifrost translation — visible reasoning is absent under `claude`.
- burp/caido MCP show failed when those apps aren't running — environmental.

## Verify
```
rc-service bifrost status
curl -s http://127.0.0.1:9084/api/providers/openai | grep base_url
echo 'Say OK' | claude -p 'Reply exactly: E2E-OK'
oc -lm        # fork model probe
```

## Thinking + Streaming for official Claude Code (anthropic-shim)

Zen sends NO thinking on its native Anthropic endpoint, and Bifrost strips `reasoning_content`
on both ingress modes. Fix: local translation shim (`claude-official/anthropic-shim.mjs`).

- Chain: `claude` -> shim :9086 -> Zen `/v1/chat/completions` (direct; Bifrost NOT in claude path)
- Shim converts Anthropic /v1/messages <-> OpenAI chat/completions:
  - upstream `reasoning_content` deltas -> `thinking` block + synthetic `signature_delta` before stop
  - full event grammar: message_start, content_block_start/delta/stop (thinking/text/tool_use), message_delta(stop_reason), message_stop
  - tool_use roundtrip: anthropic tools/tool_choice -> openai functions; tool_result -> role:"tool"
  - non-stream clients: upstream always stream=true, aggregated to a single message JSON
- Service: `/etc/init.d/anthropic-shim` (openrc default runlevel), port 9086, log /var/log/shim.log
- Claude Code env: ANTHROPIC_BASE_URL=http://127.0.0.1:9086 (settings.json); key value ignored by shim
- Verified E2E: thinking blocks recorded inside ~/.claude/projects transcripts; bash tool exec OK;
  adaptive reasoning = short questions may legitimately skip thinking.
- Gotcha: never `pkill -f anthropic-shim` from a shell whose own cmdline contains that string;
  kill by PID from `ss -tlnp`.

## Capability audit (2026-08-21)

- Boot autostart: `anthropic-shim` + `bifrost` both in openrc default runlevel; cold-start cycle verified.
  Gotcha: after killing the shim, wait for :9086 to free before `rc-service start` (stale-pid race).
- `claude doctor`: no installation issues.
- Verified working through shim: streaming (full SSE grammar), thinking blocks (in CC transcripts),
  tool_use roundtrip (Bash exec), Read/Glob/Grep (added to permissions.allow), WebSearch (client-side, works),
  session resume (`claude -p --resume <id>`), MCP stdio servers (chrome-devtools, sequential-thinking,
  playwright connected; burp/caido need their apps running), hooks config + auto-memory.sh executable,
  rig symlinks all resolve.
- Subagents: CC spawns them with model `claude-opus-5`; shim now whitelists KNOWN_MODELS and falls back
  to big-pickle for anything else (env: KNOWN_MODELS, FALLBACK_MODEL, MODEL_MAP).
- Images: anthropic image blocks -> openai image_url data URLs; Zen/big-pickle accepts them ("Red" test).
  Bugs fixed during audit: msgStart never called; content_block_stop missing for tool_use blocks;
  parts-array push-by-reference emptied before send; textAgg null-concat "nullRed" prefix.
- burp/caido MCP fail until Burp Suite / Caido apps are started (environmental, not config).

## Port verification & failover (2026-08-22)

- Full port check: 23 commands, 150 skills, 10 agents, plugins, CLAUDE.md + RECON-METHODOLOGY/
  TOOLS/WORDLISTS all reachable via ~/.claude symlinks; frontmatter is Claude-native; live test
  `/porttest` executed through claude -p OK. Settings parity env/permissions/hooks confirmed.
- Shim v2: real model FAILOVER chain — on connect error, upstream !ok, or EMPTY stream it retries
  down FALLBACK_CHAIN (default big-pickle -> Kimi-K2.6 -> DeepSeek-V4-Flash -> mimo-v2.5-free).
  Client emission is gated until first usable chunk, so hard failures still return clean HTTP 502.
  Proven: MODEL_MAP invalid-first test recovered on attempt 2 with full thinking+answer.
- Flags reference: claude-official/FLAGS-REFERENCE.md (curated full flag set, rig notes).

## Autocompact / automemory verification + portability pass (2026-08-22)

- Autocompact: FUNCTIONAL — live test with a 700KB (175k-token) file and --autocompact 100000
  triggered repeated compactions; CC thrash-guard aborted the deliberately oversized read.
  No env/config disables it; hooks fire on every PreCompact.
- Automemory: FUNCTIONAL under claude — natural Stop captures + SessionStart digest observed;
  manual hook invocation with CC-style stdin JSON works; transcript_path from CC consumed directly,
  compact_tail parser matches CC transcript format. Inbox had 31 pending captures for /learn.
- Portability: all rig docs de-hardcoded (/root/... -> ~/...), canonical-paths contract appended to
  CLAUDE.md, hooks in both settings.json files now use $HOME. Zero opencode-app references or
  secrets in rig content. Plugin state files left upstream-managed.

## Multi-provider model management (2026-08-22)
- anthropic-shim v3: UPSTREAMS provider map in /etc/conf.d/anthropic-shim
  (per-provider url/key($ENV)/ua/models); routing = MODEL_PROVIDER override >
  first provider listing model > first provider; chain entries support
  "model@provider"; a chain pin on the requested model becomes the PRIMARY
  candidate (proven: dead-provider attempt fails in ms -> falls to zen).
- /etc/conf.d/anthropic-shim: single-line JSON only (shell-sourced by openrc).
- rig-models dashboard (bin/, symlinked to /usr/local/bin): menu + subcommands
  list/add/provider/chain/default/test/status/log/restart; `default` updates
  proxy FALLBACK_MODEL AND ~/.claude/settings.json env.ANTHROPIC_MODEL.
- Verified: model test OK; cross-provider failover dahlx->zen; default switch
  Kimi<->big-pickle reflected in settings.json; claude E2E DASHBOARD-E2E-OK.

## Providers added (2026-08-23)
- tokenrouter (api.tokenrouter.com, 127 models incl. claude/gpt/gemini/qwen/kimi/grok/glm)
- bynara (router.bynara.id, ~40 models incl. ox-alpha, gpt-5.6-*, kimi-k3)
- Keys stored ONLY in local /etc/conf.d/anthropic-shim as $TOKENROUTER_KEY/$BYNARA_KEY;
  UPSTREAMS references them via "$ENV" refs. KNOWN_MODELS merged (184 total).
- Verified: direct curl OK on both; shim routing @tokenrouter/@bynara OK;
  streaming + thinking blocks via tokenrouter claude-haiku-4.5 OK.

## Free-tier model set (2026-08-23)
- Providers trimmed to free models only: zen (big-pickle, x-preview-f-free
  [= ox-alpha free], muse-spark-1.2-contributor-free, hy3-free,
  nemotron-3-ultra-free, nemotron-3.5-lightning-free, mimo-v2.5-free),
  dahl (Kimi-K2.6, DeepSeek-V4-Flash-0731; key rotated), tokenrouter
  (qwen/qwen3.8-max-free), bynara (11 free ids incl. ox-alpha),
  google (8 probed responders: gemini-3.x flash/lite + gemma-4).
- All models in shim KNOWN_MODELS => selectable in Claude Code via
  rig-models default <m> or ANTHROPIC_MODEL env override (verified:
  x-preview-f-free E2E OXFREE-E2E-OK, default path big-pickle STILL-OK).

## Groq provider added (2026-08-23)
- Probed groq catalog (13 ids; audio/tts filtered): responders = groq/compound-mini,
  openai/gpt-oss-120b, openai/gpt-oss-20b, qwen/qwen3.6-27b. allam-2-7b and
  groq/compound are blocked at key level (403). Key from openclaude profile,
  stored locally only ($GROQ_KEY). All 4 verified through shim; gpt-oss-20b
  E2E via claude: GROQ-E2E-OK. Proxy now 6 providers / 33 routes.
- Groq retry (same day): blocks lifted — added allam-2-7b, groq/compound,
  openai/gpt-oss-safeguard-20b. Groq now 7 models; proxy 6 providers / 36 routes.

## Full verification + secret mirror (2026-08-23)
- SMALL_FAST_MODEL switched big-pickle -> openai/gpt-oss-20b (fast free).
- Fallback re-proven: big-pickle@deadx fails -> attempt 2 big-pickle@zen.
- 35-model sweep: 34 direct PASS; groq/compound-mini transient network fail
  (fallback covered it; passes on retry).
- Claude E2E: MASTER-OK (big-pickle), SMALL-OK (gpt-oss-20b).
- secrets/ added WITH keys per owner authorization (private repo):
  anthropic-shim.conf, provider-profiles.json, opencode-auth.json, KEYS.md.

## Anti-hang retry patch (2026-08-23)
- Symptom: on provider failures CC showed "waiting for API" with 5-min
  escalating backoff instead of quick fallback.
- Fix: shim now wraps every candidate fetch in AbortController with
  ATTEMPT_TIMEOUT_MS (default 45s) + TOTAL_BUDGET_MS (150s) deadline, so a
  hanging provider can never stall the chain; fallback stays invisible to CC.
- Proven: black-hole provider aborted by timer -> next candidate served;
  connect-fail path skips instantly; claude E2E TIMEOUT-PATCH-OK.
- Note: node fetch has ~2.7s/request baseline latency to zen on this box
  (curl is instant) - keep ATTEMPT_TIMEOUT_MS well above it (45s default).

## Migration complete: opencode rig removed, bin remastered (2026-08-23)
- ~/.claude assets materialized from symlinks into real dirs; auto-memory.sh
  moved to ~/.claude/hooks/ (settings repointed); all .openclaude path refs
  rewritten to ~/.claude across CLAUDE.md/docs/skills/commands/agents.
- Purged (~380MB): ~/.openclaude + ~/.openclaude.json, opencode config+state
  (fresh-install reset), legacy bifrost gateway (service+cache), forked
  @gitlawb/openclaude CLI. Keys preserved in secrets/.
- bin remastered for Claude Code: oc = claude launcher (-m model override,
  -n new session, default --continue); oc-sess = CC session list/resume/show
  via ~/.claude/projects jsonl. /usr/local/bin/{oc,oc-sess} now symlinks
  (stale copies caused an interactive hang - fixed).
- Post-purge E2E: POSTPURGE-OK, OC-POST-OK, proxy test OK.

## oc flag set unified (2026-08-23)
- rig-models merged into oc flags: -lm list, -t test, -d default (auto-restart,
  updates claude settings), -fc chain, -p provider, -s status, -r restart,
  -L log, -g dashboard; launch flags -n/-m unchanged. rig-models remains as
  standalone backend. Verified: -lm/-s/-t/-d switch (D-SWITCH-OK)/-m override.

## Default model: ox-alpha alias (2026-08-23)
- MODEL_MAP aliases "ox-alpha"/"oxalpha" -> zen's x-preview-f-free (shadows
  bynara's ox-alpha; bynara variant still reachable as ox-alpha-bynara).
- Defaults now: FALLBACK_MODEL=x-preview-f-free, claude ANTHROPIC_MODEL=ox-alpha
  (small stays openai/gpt-oss-20b). E2E: OX-DEFAULT-OK served @zen.

## 2026-08-23 — oc v2 control panel + hooks cleanup
- `bin/oc` rewritten: bare `oc` = styled interactive dashboard (models/liveness,
  set default, fallback chain, providers add/edit/delete, masked keys, claude
  settings editor, proxy restart/status/log/test); any other args pass straight
  through to `claude` (pure alias; session mgmt stays native via CC picker).
  Flags kept: -lm -t -d -fc -pa -pd -k -cs -s -r -L -h.
- `-lm` fixed (field-order bug, thinking-only replies counted UP), sorted output,
  honors MODEL_MAP, tags <default>/<claude master>/<claude small>.
- Removed openclaude-era auto-memory hooks (PreCompact/Stop/SessionStart) from
  settings.json; script retired to claude-official/auto-memory.sh.retired.
  Memory is native CC only (CLAUDE.md + ~/.claude/projects transcripts).
- CLAUDE.md memory section rewritten to "native only"; AGENTS.md path refs
  .openclaude -> .claude, stale scratch note replaced.
- Conf annotated: ox-alpha is zen's model (id x-preview-f-free); bynara's
  separate ox-alpha reachable as its own id / ox-alpha-bynara.
- Verified: oc -h/-s/-k/-lm/dashboard menus/passthrough (--version)/-t PASS.

## Current state snapshot (2026-09-10)

### Shim proxy (anthropic-shim)
- **Go binary**: `claude-official/anthropic-shim` (9.4 MB, 2839 lines source)
- **Config**: `/etc/conf.d/anthropic-shim` (69 lines, shell-sourced)
- **Service**: `/etc/init.d/anthropic-shim` (openrc, default runlevel)
- **Port**: 9086, log `/var/cache/anthropic-shim/cache.json` (in-memory + disk)
- **Upstream UA**: `opencode/1.18.27 ai-sdk/provider-utils/4.0.23 runtime/bun/1.3.14`
  (Zen gates hard on this exact string)
- **Custom header**: `X-Opencode-Session: ses_{26 random chars}` (fresh per-request)
- **Default model**: `big-pickle` (mapped to `big-pickle@zen`)
- **Small/fast model**: `groq/compound-mini`
- **Context window**: 100,000 tokens (auto-compact fires at ~80k)
- **Cache**: 512 entries, 10 min TTL, deterministic responses only, disk snapshot every 5 min
- **Failover**: 6 providers, 22 models in chain, smart-wait on cooldown, provider-wide blacklist

### Providers (6)
| Provider | URL | Models | Notes |
|----------|-----|--------|-------|
| zen | opencode.ai/zen/v1/chat/completions | big-pickle | Primary; reasoning_param enabled |
| cline | api.cline.bot/api/v1/chat/completions | thinkingmachines/inkling:free, nvidia/nemotron-3-super-120b-a12b:free, cohere/north-mini-code:free | Free tier |
| bynara | router.bynara.id/v1/chat/completions | agnes-2.5-flash, stepfun-3.7-flash | OpenAI gateway |
| google | generativelanguage.googleapis.com/v1beta/openai/chat/completions | gemini-3.x flash/lite, gemma-4-xxb-it | 7 models |
| bai | api.b.ai/v1/chat/completions | glm-5.3-flash, qwen3.8-flash, hy3 | OpenAI compat gateway |
| groq | api.groq.com/openai/v1/chat/completions | groq/compound-mini, qwen/qwen3.6-27b, allam-2-7b, groq/compound, openai/gpt-oss-safeguard-20b | max_output 8192 |

### Failover chain (22 models, capability-ordered)
```
big-pickle@zen → glm-5.3-flash@bai → qwen3.8-flash@bai → gemini-3.6-flash@google →
gemma-4-31b-it@google → nvidia/nemotron-3-super-120b-a12b:free@cline → gemini-3.5-flash@google →
gemma-4-26b-a4b-it@google → groq/compound@groq → hy3@bai → qwen/qwen3.6-27b@groq →
thinkingmachines/inkling:free@cline → gemini-3.1-flash-lite@google → gemini-3.5-flash-lite@google →
gemini-3-flash-preview@google → agnes-2.5-flash@bynara → stepfun-3.7-flash@bynara →
groq/compound-mini@groq → cohere/north-mini-code:free@cline → allam-2-7b@groq →
openai/gpt-oss-safeguard-20b@groq
```

### Claude Code settings (settings.json)
- `ANTHROPIC_BASE_URL`: `http://127.0.0.1:9086` (shim)
- `ANTHROPIC_API_KEY`: `bifrost-local` (shim doesn't validate; anything non-empty works)
- `ANTHROPIC_MODEL`: `glm-5.3-flash`
- `ANTHROPIC_SMALL_FAST_MODEL`: `groq/compound-mini`
- `CLAUDE_CODE_MAX_CONTEXT_TOKENS`: `100000`
- `API_TIMEOUT_MS`: `300000` (5 min client timeout)
- `DISABLE_TELEMETRY`: `1`, `DISABLE_ERROR_REPORTING`: `1`
- Permissions: Bash(*), WebFetch/WebSearch(*), Read/Glob/Grep, MCP servers
- Hooks: Stop + PreCompact → memory/capture.sh, SessionStart → pending captures reminder

### MCP servers (5, in .claude.json)
| Server | Type | Endpoint |
|--------|------|----------|
| burp | http | http://127.0.0.1:9876 |
| caido | http | http://127.0.0.1:3333/mcp |
| sequential-thinking | stdio | mcp-server-sequential-thinking |
| playwright | stdio | playwright-mcp (chromium headless) |
| browser-control | stdio | node /root/browser-control-mcp/mcp-server/dist/server.js |

### Rig contents (as of 2026-09-10)
- **171 skills** (SKILL.md in `claude-home/skills/`)
- **27 commands** (slash commands in `claude-home/commands/`)
- **11 agents** (subagent definitions in `claude-home/agents/`)
- **Dashboard**: `bin/rig-dashboard.mjs` (48 KB, Node.js ESM)
- **Memory capture**: `claude-home/memory/capture.sh` (Stop + PreCompact hooks)

### Install steps (quick)
```sh
# Clone/copy rig-portable to target
cd rig-portable
sh install-new-device.sh
# Installs: ~/.claude home, shim service, oc/rig-models tools
# Requires: node, curl, jq, python3, git; Go optional (prebuilt binary included)
```
