---
name: mandatory-triage-gate
description: Mandatory triage check before ANY finding is written up or delivered. Think like a hostile triager who pays $500+ bounties. If you can't pass all gates, don't write the report.
---

# Mandatory Triage Gate — Think Like a Paying Triager

## CRITICAL RULE
**Before writing ANY report or declaring ANY finding "reportable," you MUST pass this gate. No exceptions.**

## The Gate Questions (ALL must be YES)

### Gate 1: Real Impact Demonstrated
- **Did I actually cause harm?** Not "could cause" — DID cause
- **What data did I steal?** Name the exact data (emails, passwords, tokens, files)
- **What account did I take over?** Show the before/after state
- **What money did I move?** Show the balance change
- **If I can't point to concrete damage → NOT A FINDING**

### Gate 2: Reproducible by Hostile Triager
- **Can a triager with zero context paste my steps and see the same result?**
- **No "login as admin"** — show HOW to get admin credentials
- **No "navigate to settings"** — show the exact URL
- **No "send this request"** — provide the full raw HTTP request
- **If I can't write clean repro steps → NOT A FINDING**

### Gate 3: The "So What?" Test
- **If I submitted this to HackerOne right now, would a hostile triager pay $500+?**
- **Or would they close it as Informational in 2 minutes?**
- **The triager's first response is always "so what?" — my evidence must answer it**

### Gate 4: Fresh Account PoC
- **Did I reproduce from a fresh, disposable, never-before-used account?**
- **No reuse of existing sessions or cookies**
- **If I can't reproduce from scratch → NOT A FINDING**

### Gate 5: Impact Over Cleverness
- **Is this a real attack a real attacker would use?**
- **Or is it a clever trick with no practical impact?**
- **If it requires unrealistic preconditions → NOT A FINDING**

## The Kill Question

Before EVERY report, answer this:

> "If I submitted this to HackerOne right now with $500+ bounty, would a hostile triager pay — or close it as Informational in 2 minutes?"

**If the answer isn't clearly "$500+", it does NOT get written up.**

## Common False Positives to REJECT

| Finding | Why it's false positive |
|---------|----------------------|
| CAPTCHA bypass without ATO | "Can send spam" = Informational |
| SSRF without data exfil | "Server tried to connect" = Informational |
| IDOR that returns same data | "Accessed my own data" = Not a finding |
| XSS that doesn't execute | "Payload reflected but encoded" = Not XSS |
| Missing security headers | "No bounty, informational" |
| Rate limiting without abuse | "Theoretical, not demonstrated" |
| Version disclosure | "No CVE, no impact" |
| Verbose errors without leak | "Annoying, not harmful" |

## The Standard

**Bug bounty pays for IMPACT, not for WEAKNESSES.**

A pentest report gets paid for coverage — every finding down to informational goes in. Bug bounty pays for NONE of that: only a real, exploitable, high-impact vulnerability gets accepted.

**Every finding must answer:**
1. What did the attacker ACTUALLY get?
2. Can a triager reproduce it in under 5 minutes?
3. Would this survive "so what?" from a skeptical program owner?
4. Is this a finding or a theoretical weakness?

**If ANY answer is "I don't know" or "theoretically" → DON'T WRITE IT UP.**

## Worked example — same lead, failing then passing the gate

**First pass (does NOT pass — this is the mistake to avoid):**
> "Found IDOR on `/api/orders/{id}`. Changing the ID to a different number
> returns a 200 instead of a 403, so it looks like you can access other
> users' orders."

Why this fails: Gate 1 fails — "returns a 200" is not "I saw another named
user's real order data." No proof was actually captured. This is a
hypothesis, not a finding, even though the instinct (sequential ID + no
ownership check) is correct.

**Second pass (passes — this is the standard to hit):**
> Gate 1 (impact): using disposable account A1's token against order ID
> belonging to disposable account B1, the response body contains B1's real
> name, shipping address, and order total — pasted verbatim in evidence,
> B1's email visible in the JSON.
> Gate 2 (repro): raw request/response included, roles labeled A1/B1, no
> "log in as admin" hand-waving.
> Gate 3 ("so what"): *"Any authenticated user can read any other user's
> full order + shipping/PII by incrementing an integer ID — full account
> enumeration of the order history of the entire user base."*
> Gate 4 (fresh account): A1 and B1 were both provisioned fresh for this
> engagement, confirmed via `@accounts` ledger.
> Gate 5 (realism): sequential integer ID, zero special preconditions —
> exactly what a real attacker would do first.

Same underlying bug, same technique — the difference between Informational
and a paid report is entirely in what evidence was actually captured before
writing it up. This is exactly what `@skeptic` (see the
`adversarial-verification` skill) checks independently before anything
reaches `@report`.
