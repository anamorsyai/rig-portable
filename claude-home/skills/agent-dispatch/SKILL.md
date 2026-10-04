---
name: agent-dispatch
description: Parallel multi-agent dispatch rules — when to call which specialist, ownership boundaries, and return contracts for maximum bug bounty throughput.
---

# Agent Dispatch (Maximum Throughput)

## Parallel start patterns
- New engagement: `@intake` + `@recon` together
- Partial live hosts: `@scan` + `@map` + `@vuln` on P0 (auth/payment/API)
- Testing: `@vuln` + `@bizlogic` + `@bac` in parallel on different surfaces
- Broken tool + active lead: `@fixer` + owner of the lead together

## Who owns what (call, don't absorb)
| Need | Call |
|------|------|
| Subdomains / OSINT | @recon (+ @intel for web OSINT) |
| Ports / stack / WAF | @scan |
| Endpoints / auth flows / JS | @map |
| Injection / IDOR / SSRF / tech vulns | @vuln |
| Workflow / race / price / state | @bizlogic |
| Verify + impact gate | @exploit |
| Combine findings | @chain |
| Scope / dupe check | @scope |
| Write report | @report |
| Tool broken | @fixer |
| Technique research | @intel (+ @osint live web) |
| Bulk payloads | @deepseek |
| Quick script <50 lines | @hy3 |
| Complex PoC / multi-file code | @hy3 |
| Fresh account / role | @accounts |
| Progress consolidate | @tracker |
| Policy / scope extract | @intake |

## Return contract (every subagent)
1. Status: done | partial | blocked | failed
2. Artifacts written (paths)
3. Handoffs made (agent + why)
4. Blockers (if any)
5. Next recommended action (one line)

## Anti-patterns
- Waiting for full recon before testing known hosts
- @build doing ffuf/sqlmap itself
- Reporting theoretical impact
- Deep T4 checks before T1-T3 on same endpoint
- One agent serializing three independent tasks
