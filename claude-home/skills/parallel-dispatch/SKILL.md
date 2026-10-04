---
name: parallel-dispatch
description: Parallel agent dispatch - multiple agents work simultaneously, never wait, maximize throughput
---

# Parallel Dispatch Methodology

## CORE RULE: NEVER WAIT

If you're waiting for one agent, you should be dispatching another.
Independent work runs TOGETHER.

## PARALLEL PATTERNS

### Pattern 1: New Target (Full Engagement)
```
DISPATCH IN PARALLEL:
  @intake → fetch program policy, extract scope
  @recon → subdomain enumeration
  @scan → port scan on known live hosts (if any)
  @map → endpoint discovery from recon data

DO NOT WAIT for @intake before starting @recon.
DO NOT WAIT for @recon before starting @scan.
```

### Pattern 2: Recon Done (Testing Begins)
```
DISPATCH IN PARALLEL:
  @scan → port scan on all live hosts
  @map → start aggregating endpoints
  @vuln → start testing P0 endpoints (auth, payment, API)

DO NOT WAIT for @scan to finish before @map starts.
DO NOT WAIT for @map to finish before @vuln starts.
```

### Pattern 3: Testing Underway
```
DISPATCH IN PARALLEL:
  @vuln → testing endpoint A
  @bizlogic → testing workflow B
  @bac → manual testing on endpoint C

DO NOT WAIT for one to finish before starting the other.
```

### Pattern 4: Multiple Findings
```
DISPATCH IN PARALLEL:
  @exploit → verify finding 1
  @exploit → verify finding 2
  @exploit → verify finding 3

DO NOT WAIT for one verification before starting the next.
```

### Pattern 5: Chaining
```
DISPATCH IN PARALLEL:
  @chain → check if finding 1 chains with finding 2
  @chain → check if finding 1 chains with finding 3
  @chain → check if finding 2 chains with finding 3

DO NOT WAIT for one chain analysis before starting the next.
```

### Pattern 6: Multi-Asset
```
DISPATCH IN PARALLEL:
  @vuln → test asset A
  @vuln → test asset B
  @vuln → test asset C

DO NOT WAIT for one asset before starting the next.
```

## WHAT BLOCKS, WHAT DOESN'T

| Task Type | Blocks Pipeline? | Why |
|-----------|-----------------|-----|
| nmap port scan | NO | Results needed for @map, not @vuln immediate |
| ffuf fuzzing | NO | Results feed into @map |
| subdomain enum | NO | @scan can start on discovered hosts |
| JS file analysis | NO | @map can aggregate partial results |
| httpx probing | Depends | Small batch: wait. Large: dispatch and continue |
| Tool installation | NO | @fixer handles, others continue |
| Report writing | NO | No downstream depends on report |

## THE RULE

**If you're waiting, you're doing it wrong.**
There's always something else to work on while a slow task runs.
Dispatch, continue, check back.

## AGENT CALLING

When dispatching:
```
task(subagent_type="recon", prompt="Start subdomain enumeration for target.com")
task(subagent_type="scan", prompt="Port scan on known live hosts: [list]")
task(subagent_type="vuln", prompt="Test P0 endpoints: [list]")
```

All three run in parallel. Do not wait for one to finish before starting the next.
