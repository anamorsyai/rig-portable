# rig-portable

Complete, self-restoring mirror of the Claude Code hunting rig. Clone on any device, run the installer, and you're fully operational.

## Quick start (new device)

```sh
git clone https://github.com/anamorsyai/rig-portable.git
cd rig-portable
sh install-new-device.sh   # needs root for /etc/conf.d + init.d
claude
```

The installer restores `~/.claude` (fresh state — no session/memory data), deploys the proxy config with keys to `/etc/conf.d/anthropic-shim`, registers and starts the OpenRC service, and installs the `oc` / `rig-models` control tools. Requires: `node`, `curl`, `jq`, `python3`.

## Layout

| Path | Purpose |
|---|---|
| `install-new-device.sh` | one-shot bootstrap: home + service + tools + health check |
| `claude-home/` | full `~/.claude` mirror: 150 skills, 23 commands, 10 agents, plugins, settings (fresh-image policy: no sessions/memory/history) |
| `claude-official/anthropic-shim.mjs` | Anthropic->OpenAI translation proxy (:9086) with failover chain, per-provider thinking flags |
| `claude-official/services/` | OpenRC units (`init.d-anthropic-shim`, `init.d-bifrost`) |
| `claude-official/settings.json` | deployed Claude settings snapshot |
| `bin/oc`, `bin/rig-models` | interactive rig control panel + model/provider manager |
| `secrets/anthropic-shim.conf` | REAL provider keys (repo must stay PRIVATE). Source of truth for `/etc/conf.d/anthropic-shim` |

## Runtime architecture

- Claude Code -> local shim `127.0.0.1:9086` -> provider chain
- Primary: `deepseek-ai/DeepSeek-V4-Flash-0731@dahl`; fallbacks: `x-preview-f-free@zen` -> `moonshotai/Kimi-K2.6@dahl` -> `agnes-2.5-flash@bynara`
- Thinking is forced visible on dahl via `chat_template_kwargs.thinking` (`thinking_param` flag)
- `ox-alpha` auto-aliases to zen's `x-preview-f-free` (bynara's variant stays reachable as `ox-alpha-bynara`)
- Timeouts tuned against burst load: 45s per attempt, 120s total budget

## Operations

- Panel: `oc` (models/liveness/chain/service control), logs at `/var/log/shim.log`
- Change default model: `rig-models default <model-id>` or `oc -d <model>`
- Restart shim: `rc-service anthropic-shim restart`

## History notes

- 08-22: initial sanitized mirror; 08-23: multi-provider routing, oc panel; 08-24/25: adaptive thinking, burst-resilient chain, fresh-image claude-home, bootstrap installer.
- Earlier commits contained real API keys; assume rotated. Current tree keeps keys ONLY under `secrets/` by owner's explicit choice.
