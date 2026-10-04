---
name: continuous-loop
description: Recursive testing loop â€” ensures every endpoint/vuln-class is covered before stopping. Use when dispatching agents, checking progress, or deciding next action. Prevents premature stopping.
---

# Continuous-Loop Engagement Controller

This skill ensures the hunt never stops until full target coverage is confirmed.

## The Loop (every agent must follow)

### Phase transitions (automatic)
```
RECON â†’ SCAN â†’ MAP â†’ VULN/BIZLOGIC â†’ EXPLOIT â†’ CHAIN â†’ SCOPE â†’ REPORT
  â†‘                                                              |
  â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€ never stop, always loop back â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜
```

### Decision tree after every agent return

```
Agent returned result
  â”œâ”€ New asset discovered? â†’ dispatch @recon/@scan for that asset
  â”œâ”€ New endpoint found? â†’ dispatch @vuln on it (T1-T3)
  â”œâ”€ Finding confirmed? â†’ @exploit â†’ @chain â†’ @scope â†’ @report
  â”œâ”€ Finding needs chaining? â†’ @chain with other findings
  â”œâ”€ Phase complete? â†’ check PROGRESS.md for untested areas
  â”œâ”€ All P0 tested? â†’ move to P1
  â”œâ”€ All P1 tested? â†’ move to P2
  â”œâ”€ All P2 tested? â†’ move to P3
  â”œâ”€ All endpoints tested? â†’ run @chain on all findings
  â”œâ”€ All chains checked? â†’ run @exploit on all candidates
  â”œâ”€ All candidates verified? â†’ @report writes up
  â””â”€ @tracker confirms FULL COVERAGE â†’ THEN stop
```

### What "full coverage" means (all must be true)
1. Every mapped endpoint tested for T1-T3 vuln classes
2. Every confirmed finding checked for chain potential
3. Every chain verified by @exploit
4. Every verified finding has report or ruled-out status
5. @tracker's PROGRESS.md shows ALL phases done

### NEVER stop when
- "Most" endpoints tested â€” all must be tested
- First finding found â€” keep looking for chains and more
- One phase seems done â€” check if new assets appeared mid-testing
- Bored â€” boredom is not a coverage signal

### Auto-dispatch triggers (fire immediately, don't batch)
| Signal | Dispatch |
|--------|----------|
| New subdomain found | @scan on it |
| New live host | @map on it |
| New endpoint | @vuln on it (T1-T3 first) |
| New parameter | @vuln on that param |
| New auth flow | @auth-bypass on it |
| New JS file | @map for endpoint extraction |
| New finding | @exploit for verification |
| Finding verified | @chain for combination check |
| Phase complete | @tracker to update PROGRESS.md |
| All phases done | @tracker for final coverage check |

### Progress tracking
After every 3 agent dispatches, @build calls @tracker to:
1. Read PROGRESS.md
2. Check which endpoints/flows are still untested
3. Identify gaps
4. Dispatch responsible agents for gaps

### The kill condition
Only @build can stop the loop, and ONLY when @tracker confirms:
- "All P0/P1/P2 endpoints tested T1-T3"
- "All findings chained or ruled out"
- "All chains verified"
- "Full coverage confirmed"

Until then: KEEP DISPATCHING.

## Error handling in the loop

Every agent return must check for errors before continuing:

```
Agent returned
  â”œâ”€ Error in result?
  â”‚   â”œâ”€ Transient (timeout/reset/stream)? â†’ retry once â†’ if fails â†’ @fixer
  â”‚   â”œâ”€ Deterministic (missing tool/permission)? â†’ @fixer immediately
  â”‚   â”œâ”€ Upstream (proxy/502/503)? â†’ oc-proxy status â†’ oc-watchdog â†’ wait 5s â†’ retry
  â”‚   â”œâ”€ Rate limit (429/403)? â†’ wait 30s â†’ retry â†’ if persists â†’ switch provider
  â”‚   â”œâ”€ Stream truncated? â†’ re-run that specific checkpoint (not whole phase)
  â”‚   â””â”€ Unknown? â†’ log in retry-queue.md â†’ try alternate approach
  â”œâ”€ No error? â†’ continue normal loop
  â””â”€ Agent blocked? â†’ dispatch different agent for same target
```

### Auto-fix triggers (fire @fixer immediately)
| Signal | @fixer action |
|--------|---------------|
| `command not found` | Install tool |
| `ECONNRESET` 2x | Check proxy health, restart |
| `Connect Timeout` 3x | Switch provider or check network |
| `circuit breaker open` | Wait 60s or restart proxy |
| `permission denied` | Fix file perms |
| `JSON truncated` | Re-run checkpoint |
| Tool crashes 2x | Reinstall tool |
