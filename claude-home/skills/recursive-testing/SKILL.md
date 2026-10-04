---
name: recursive-testing
description: Recursive testing methodology - never stop, loop until full coverage, propagate findings
---

# Recursive Testing Methodology

## CORE RULE: NEVER STOP

Testing is never "run one payload, read one response, move on."
For every hypothesis, work it like a loop until genuinely exhausted.

## THE RECURSIVE LOOP

```
┌─────────────────────────────────────────────────────────┐
│  1. GENERATE variations BEFORE testing                  │
│  2. EXECUTE most promising variant FIRST                │
│  3. OBSERVE all response differences                    │
│  4. CHAIN what you learn into next variant              │
│  5. BRANCH if direct approach exhausted                 │
│  6. RECURSE into new leads immediately                  │
│  7. Only mark dead after FULL loop exhaustion           │
└─────────────────────────────────────────────────────────┘
```

### Step 1: Generate Variations
Before testing ANY endpoint, generate:
- Different payloads (SQLi, XSS, SSRF, etc.)
- Different encodings (URL, double-URL, HTML, Unicode)
- Different parameters (GET, POST, JSON, XML)
- Different HTTP methods (GET, POST, PUT, DELETE, PATCH)
- Different content types (form, JSON, XML, multipart)

### Step 2: Execute Most Promising
Start with the highest-probability variant.
Don't test one variant and conclude.

### Step 3: Observe Response Differences
- Status code changes
- Response length changes
- Timing differences
- Error message differences
- Header differences

### Step 4: Chain Learnings
Feed what you learned into the next variant.
If SQLi blocked on `'`, try `"`, `)`, `))`, etc.

### Step 5: Branch When Exhausted
If the direct approach is fully exhausted, step back:
- What's a DIFFERENT vulnerability class?
- What's a DIFFERENT entry point?
- What's a DIFFERENT parameter?

### Step 6: Recurse Into New Leads
Any new endpoint, parameter, token, or role discovered mid-test
goes back through this same loop BEFORE returning to original.

### Step 7: Mark Dead Only After Full Loop
Only mark a hypothesis "dead" after:
- Multiple payload variations tested
- Multiple encoding attempts
- Multiple HTTP methods
- Multiple parameters
- Different entry points tried
- New angles from @intel explored

## PROPAGATION RULE

When something is confirmed on endpoint A:
1. Does this technique apply to OTHER endpoints?
2. Does this finding CHAIN with other findings?
3. Does this finding ENABLE access to new endpoints?
4. Does this reveal a NEW vulnerability class?

## AGENT CALLING DURING LOOP

- Finding confirmed → @exploit (verify)
- Need fresh angle → @intel (research)
- Need payloads → @deepseek (generate)
- Need tool fix → @fixer (repair)
- Dead end exhausted → @intel (new angles)
- New endpoint found → test it (recursive)

## THE RULE

**If you stopped after one test, you didn't finish.**
The loop runs until ALL variations are exhausted or ALL angles tried.
