---
name: ai-coverage-intelligence
description: AI intelligence layer that reviews script-based coverage gates â€” ensures every vulnerability class, bypass, CVE, endpoint, and domain is genuinely tested. Scripts can false-positive; AI is the second brain that catches gaps scripts miss. Mandatory after every oc-coverage-tracker / oc-decision-gate run.
---

# AI Coverage Intelligence â€” The Second Brain

Scripts measure coverage mechanically. **This skill is the AI intelligence
layer that reviews every script output and makes the final call.** Scripts
can miss things. Scripts can false-positive. This agent ensures nothing
falls through the cracks.

## When to use

- After every `oc-coverage-tracker` run
- After every `oc-decision-gate` run
- Before any engagement is declared "complete"
- When the script says STOP but something feels wrong
- When deciding whether to recurse or finalize

## The Problem with Scripts

Scripts check:
- âœ… Does a file exist?
- âœ… Does a keyword appear in a file?
- âœ… Is the endpoint count > 0?

Scripts MISS:
- âŒ Was the SQLi test ACTUALLY thorough (not just one payload)?
- âŒ Was the XSS test done with DOM variants, not just reflected?
- âŒ Were CloudFlare bypass techniques attempted for this WAF?
- âŒ Were CVEs checked for the specific framework version?
- âŒ Were all HTTP methods tested (not just GET)?
- âŒ Were parameter pollution and content-type switches tested?
- âŒ Was the IDOR tested across tenants (not just same-tenant)?
- âŒ Were race conditions tested with actual parallel requests?
- âŒ Were file upload bypasses tested (not just regular upload)?
- âŒ Were GraphQL endpoints tested if the API has them?

**A file existing does NOT mean the test was complete.**

## AI Review Procedure

### Step 1: Read Script Output

```bash
oc-coverage-tracker "$PROJECT" --json
oc-decision-gate "$PROJECT" --json
oc-methodology-gate "$PROJECT" --status --json
```

Parse the JSON. Understand what the script THINKS is covered.

### Step 2: Deep Quality Review

For EVERY vulnerability class the script claims is "tested":

**SQL Injection:**
- Was error-based tested? â†’ confirm payload tried `'`, `"``, `))`
- Was blind tested? â†’ confirm boolean-based AND time-based attempted
- Was UNION tested? â†’ confirm `UNION SELECT` variants tried
- Was WAF bypass tested? â†’ confirm encoding, chunking, comment injection
- Were ALL parameters tested? â†’ not just one param
- Was out-of-band tested? â†’ confirm dns/http callback attempted
- What DBMS? â†’ MySQL, PostgreSQL, MSSQL need different payloads

**XSS:**
- Was reflected tested? â†’ confirm payload in response body
- Was stored tested? â†’ confirm persistence verified
- Was DOM tested? â†’ confirm source/sink analysis done
- Was CSP tested? â†’ confirm bypass attempted if CSP present
- Were event handlers tested? â†’ `onload`, `onerror`, `onfocus`, etc.
- Were filter bypasses tested? â†’ `<img`, `<svg`, `<details`, case variation

**SSRF:**
- Was cloud metadata tested? â†’ `169.254.169.254` for AWS/GCP/Azure
- Was internal service discovery attempted? â†’ localhost, private IPs
- Was blind SSRF tested? â†’ OOB callback confirmed
- Were all URL parameters tested? â†’ not just `url=`

**IDOR/BOLA:**
- Was horizontal tested? â†’ access another user's resources
- Was vertical tested? â†’ access admin resources as regular user
- Was multi-tenant tested? â†’ cross-tenant access
- Were ALL identifier types tested? â†’ sequential IDs, UUIDs, slugs
- Was batch endpoint tested? â†’ bulk operations for IDOR

**Auth Bypass:**
- Was JWT tested? â†’ alg:none, algorithm confusion, key rotation
- Was OAuth tested? â†’ redirect_uri, state, PKCE
- Was session tested? â†’ fixation, logout not invalidating
- Was 2FA tested? â†’ bypass, brute force, response manipulation
- Was password reset tested? â†’ token predictability, host header injection

**Command Injection:**
- Was OS command injection tested? â†’ `;`, `|`, `||`, `&&`, backticks
- Was blind tested? â†’ time-based, OOB callback
- Were filter bypasses tested? â†’ space alternatives, keyword splitting

**For EACH class, ask: "If I were a real attacker, would I have tried more?"**

### Step 3: Endpoint/Subdomain Coverage

Read the recon data and verify EVERY discovered asset was tested:

```bash
# Compare discovered vs tested
cat "$PROJECT/recon/subdomains.txt" | wc -l     # total discovered
cat "$PROJECT/recon/live-hosts.txt" | wc -l     # live hosts
# Were ALL live hosts port-scanned?
# Were ALL live hosts endpoint-mapped?
# Were ALL endpoints vuln-tested?
```

For each untested endpoint, determine:
- Is it in scope?
- Does it run a different service?
- Does it have different auth?
- Could it be a higher-value target?

### Step 4: Technology-Specific Checks

Based on the tech stack fingerprinted:

- **WordPress**: plugins, themes, XMLRPC, wp-admin, REST API
- **Django**: admin panel, CSRF on state-changing, debug mode
- **Express/Node**: prototype pollution, path traversal, eval()
- **Spring**: SpEL injection, Actuator endpoints, deserialization
- **Laravel**: debug mode, deserialization, SQLi in Eloquent
- **GraphQL**: introspection, batching, field-level auth
- **nginx**: path confusion, alias traversal

### Step 5: Bypass & Edge Case Review

Based on the WAF/CDN detected:

- **Cloudflare**: origin IP discovery, HTTP/2, chunked, Unicode
- **Akamai**: path manipulation, HPP, case variation
- **ModSecurity**: encoding tricks, comment injection
- **AWS WAF**: IP rotation, fragmentation

### Step 6: CVE Check

Based on identified versions:

```bash
# The AI should know these patterns:
# - Struts2 â†’ CVE-2017-5638 (S2-045), CVE-2018-11776
# - Apache â†’ CVE-2021-41773, CVE-2021-42013
# - jQuery â†’ CVE-2020-11022, CVE-2020-11023
# - OpenSSL â†’ CVE-2014-0160 (Heartbleed)
# - etc.
```

Check exploit-db, NVD, or reference the cve-exploit-matcher skill.

### Step 7: Override the Script

After the deep review, make a decision:

**OVERRIDE STOP â†’ RECURSE** if:
- Any vuln class was tested with only 1-2 payloads (not thorough)
- Any endpoint/subdomain was skipped
- Any bypass technique was not attempted
- Any CVE matches the stack but wasn't tested
- Any HTTP method was left untested
- Any content-type variant was ignored
- The "tested" file is just a stub/template, not real results

**CONFIRM STOP** only if:
- Every vuln class was tested with multiple techniques per class
- Every endpoint was tested
- Every relevant CVE was checked
- Every bypass technique was attempted
- The evidence files contain real payloads and responses, not placeholders

### Step 8: Write the Decision

Write to `$PROJECT/state/ai-review.json`:

```json
{
  "timestamp": "2026-01-01T00:00:00Z",
  "script_decision": "STOP",
  "ai_decision": "RECURSE",
  "override_reason": "SQLi tested with only 1 payload, no WAF bypass attempted, 3 endpoints untested",
  "gaps_found": [
    "SQLi: only error-based tested, blind/UNION/OOB missing",
    "XSS: no DOM-based testing",
    "IDOR: cross-tenant not tested",
    "3 endpoints on api.internal.example.com untested"
  ],
  "cve_check": {
    "stack": "Django 3.2 + PostgreSQL",
    "tested": ["CVE-2021-33203"],
    "missing": []
  },
  "coverage_quality": "low â€” scripts passed but testing was superficial"
}
```

**The AI decision overrides the script decision. ALWAYS.**

## Quality Scoring

After the review, rate coverage quality:

| Score | Meaning |
|-------|---------|
| **high** | Every class tested with multiple techniques, every endpoint covered, CVEs checked |
| **medium** | Most classes tested but some gaps in technique depth |
| **low** | Scripts pass but testing was superficial â€” single payloads, missing endpoints |

Only "high" quality should trigger STOP. "medium" = RECURSE with specific gaps.
"low" = RECURSE with urgency.

## Integration with @tracker

The @tracker agent runs this review as part of its progress consolidation.
When @tracker calls `oc-coverage-tracker` and `oc-decision-gate`, it does NOT
blindly trust the output. It performs this full AI review before reporting
the decision to @build.

**@tracker is the second brain. Scripts are the first pass. @tracker decides.**
