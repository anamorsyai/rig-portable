---
name: aggressive-hunting
description: Aggressive bug bounty hunting - never stop, never settle for weak findings, always chain to real impact
---

# Aggressive Hunting Mode

## CORE RULES (NON-NEGOTIABLE)

### Rule 1: NEVER STOP
- If one path is blocked, try another immediately
- If a tool fails, fix it or use alternative
- If an agent is slow, continue with other work
- NEVER say "done" until ALL endpoints are tested

### Rule 2: REAL IMPACT OR NOTHING
- "Interesting" is NOT "exploitable"
- "Works" is NOT "impactful"
- Every finding MUST answer: "What did the attacker ACTUALLY get?"
- If answer is "nothing concrete" â†’ NOT A FINDING

### Rule 3: CHAIN EVERYTHING
- Single low-severity finding = CHAIN COMPONENT, not report
- IDOR + email change = ATO
- XSS + cookie theft = session hijack
- SSRF + metadata = cloud compromise
- SQLi + data exfil = breach

### Rule 4: ATTACKER MINDSET
- What would a real attacker do with this?
- How would they monetize this access?
- What's the worst case scenario?
- Can they escalate to admin?

## FINDING BAR (5 GATES)

Every finding MUST pass ALL 5 gates:

1. **Reproducible from FRESH account** - new identity, no prior state
2. **Copy-paste repro steps** - hostile triager can follow exactly
3. **REAL damage demonstrated** - data, money, access, NOT theory
4. **High or Critical severity** - Medium = chain component only
5. **Survives hostile triage** - skeptical triager still accepts

## AGENT CALLING RULES

### When to call another agent:
- Finding needs verification â†’ @exploit
- Finding needs chaining â†’ @chain
- Need fresh account â†’ @accounts
- Need tool fixed â†’ @fixer
- Need research â†’ @intel
- Need Burp analysis â†’ oc-burp-* tooling (oc-burp-setup/queue/params/tokens)

### How to call:
```
task(subagent_type="exploit", prompt="Verify finding F001: [details]")
task(subagent_type="chain", prompt="Check if F001 chains with F002")
task(subagent_type="accounts", prompt="Need fresh admin account for testing")
```

### NEVER:
- Stop testing because one path failed
- Report theoretical impact
- Skip verification steps
- Assume something is secure without testing

## RECURSIVE TESTING LOOP

For every endpoint:
1. Test T1 (RCE, auth bypass, SQLi)
2. Test T2 (IDOR, privilege escalation, stored XSS)
3. Test T3 (race conditions, business logic)
4. Test T4 (info disclosure, only if chains to higher)
5. If blocked â†’ try different angle immediately
6. If finding â†’ chain with others â†’ verify impact â†’ report

## REPORT QUALITY CHECK

Before writing report, ask:
1. Would a hostile triager pay $500+ for this?
2. Can they reproduce it in under 5 minutes?
3. Is the impact obvious from evidence alone?
4. Does it survive "Informational" closure?
5. Is this a real attack path?

If ANY answer is NO â†’ keep testing, don't report.
