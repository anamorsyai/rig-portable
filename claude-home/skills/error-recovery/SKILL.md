---
name: error-recovery
description: "Auto-detect and recover from errors: timeouts, proxy failures, connection drops, stream truncations, tool crashes. Fire @fixer or self-heal. Never block on errors — track, retry, alternate, escalate."
---

# Error Recovery Protocol

When ANY error occurs, follow this exact sequence:

## Step 1: Classify the error

| Error Pattern | Type | Action |
|---------------|------|--------|
| `Connect Timeout` / `operation timed out` | Transient | Wait 5s, retry once. If fails again â†’ @fixer |
| `ECONNRESET` / `other side closed` | Transient | Wait 3s, retry once. If fails again â†’ @fixer |
| `NGHTTP2_PROTOCOL_ERROR` | Protocol | Already fixed (HTTP/1.1). If recurs â†’ @fixer |
| `stream error` / `stream closed` | Transient | Retry once. If fails â†’ @fixer |
| `circuit breaker open` | Upstream | Wait 60s for cooldown. If persists â†’ switch provider |
| `403` / `429` | Rate limit | Wait 30s, reduce request rate. If persists â†’ switch provider |
| `Internal server error` | Upstream | Retry once. If fails â†’ @fixer |
| `Headers Timeout Error` | Transient | Retry once. If fails â†’ @fixer |
| `depth limit reached` | Structural | Don't retry â€” call agent directly instead of nested delegation |
| `command not found` / `missing` | Deterministic | @fixer install |
| `permission denied` | Config | Check file perms, fix or @fixer |
| `JSON parse error` / `truncated` | Stream failure | Re-run immediately â€” output was cut mid-generation |

## Step 2: Self-heal or delegate

### Self-heal (if simple fix):
```bash
# Proxy down?
oc-proxy status && oc-watchdog; sleep 1; oc-proxy check

# Tool missing?
command -v <tool> || go install <package>@latest

# File permission?
chmod +x <file>

# Connection timeout?
sleep 5  # wait and retry
```

### Delegate to @fixer (if complex):
When to call @fixer:
- Tool crashes 2x with same error
- Proxy restart doesn't fix upstream errors
- Missing dependency that needs package manager
- Any error you can't fix in 1 line

### Switch provider (if upstream dead):
```
oc-rate-limit-handler bypass   # drop/rotate on 429/403, retry via Tor
oc-rotate-ip new               # new Tor circuit (fresh exit IP)
oc-proxy tor                   # route through Tor SOCKS5
```

## Step 3: Track and continue

After recovery:
1. Log the error + fix in `retry-queue.md`
2. Resume from where you left off
3. Don't re-run the whole phase â€” just the failed step

## Step 4: If all else fails

If the error blocks progress and can't be fixed:
1. Log it clearly in STATUS.md as "blocked"
2. Switch to a different endpoint/asset that isn't affected
3. Come back to it later or surface to human

## The rule
**Never let an error stop the hunt.** Every error has a path forward:
- Retry â†’ fix â†’ alternate â†’ switch provider â†’ log and move on
The only thing you never do is STOP.

## Credit/Rate Limit errors

| Error | Cause | Action |
|-------|-------|--------|
| `Insufficient credits` | API key out of money | Switch to free model or provider |
| `rate_limited` | Too many requests | Wait 60s, reduce rate |
| `The model rejected this request` | Credits or unsupported param | Check credits, try free model |

### Auto-switch on credit failure:
```bash
# Rotate exit IP via Tor (fresh circuit)
oc-rotate-ip new

# Or route through Tor SOCKS5 with rate-limit handling
oc-proxy tor
oc-rate-limit-handler bypass
```
