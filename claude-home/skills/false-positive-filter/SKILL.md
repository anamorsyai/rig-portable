---
name: false-positive-filter
description: Three-gate false positive detection and filtering — technical validity, impact validity, triage survival. Kill non-findings before they waste report-writing time.
---

# False Positive Filter

## The reproduce-or-drop rule
**Can't reproduce twice in a row? It's not real. Drop it.**

## Three-gate filter

### Gate 1: Technical validity
Did you ACTUALLY trigger the vulnerability?

| Vuln | Real | Fake |
|------|------|------|
| SQLi | DB error, different data, timing delay | Normal response, no DB error, same timing |
| XSS | Browser executed JS | Payload reflected but encoded/escaped |
| IDOR | Accessed another user's data | Same data regardless of ID, or your own data |
| SSRF | Got callback from server | No callback, or server blocked it |
| Auth bypass | Accessed protected resource unauthenticated | Got 401/403, redirected to login, or resource is public |
| RCE | Command actually executed | Payload reflected but no execution |
| Race | State changed unintentionally | State consistent, request serialized |
| Biz logic | Business rule bypassed | Server enforced rule, client-side only |

### Gate 2: Impact validity
Did it produce real damage?

- Did you see actual sensitive data? (not error page, not login form, not your own data)
- Did the application state actually change?
- Can you repeat the impact consistently?
- Is this a real attack path? (not impossible preconditions)

### Gate 3: Triage survival
Would a hostile triager accept this?

- Is this a known informational finding? (missing headers, verbose errors, version disclosure)
- Does the program explicitly exclude this? (best practices, defense-in-depth)
- Is this a well-known duplicate?
- Is the impact theoretical only?
- Does it require unrealistic user interaction?

## Per-vuln-class false positive traps

| Vuln | Common fake | Verify it's real |
|------|-------------|-----------------|
| SQLi | WAF/custom error page returns DB-like errors | Confirm with sqlmap or manual payload that extracts data |
| XSS | Payload reflected but HTML-encoded | Check source — inside JS string, attribute, or raw HTML? |
| IDOR | Returns same data for all IDs | Confirm response body differs between users |
| SSRF | Callback from CDN/proxy, not server | Verify source IP of callback |
| Auth bypass | Endpoint is actually public | Check unauthenticated vs authenticated response |
| Open redirect | Redirect blocked by browser | Confirm redirect actually completes |
| Race condition | Request retried automatically | Disable auto-retry, send once, verify state |
| Business logic | Client-side validation only | Confirm server enforces rule (raw request without UI) |

## The "so what?" checklist
Before promoting any finding:
1. Can a triager reproduce in under 5 min?
2. Does the triager need my explanation to get the impact?
3. Would this survive a skeptical program owner's "so what?" challenge?
4. Is this a finding or a feature?

## When to reject
- Not reproducible → reject
- Only works once → reject
- Theoretical impact only → reject
- Known informational → reject
- Program excludes it → reject
- Requires impossible preconditions → reject

## Logging false positives
Write to `vuln/false-positives.md`:
```
## [Vuln class] on [endpoint] — FALSE POSITIVE
- What I tried: [payload/method]
- What happened: [result]
- Why it's fake: [reason]
- Lesson: [detection pattern to avoid next time]
```
