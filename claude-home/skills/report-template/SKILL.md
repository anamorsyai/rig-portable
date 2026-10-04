---
name: report-template
description: Standardized bug bounty vulnerability report format and writing guidelines â€” structure, evidence formatting, impact articulation, and triage-survival tactics for maximum acceptance.
---
---

# Report Template

Write reports that survive hostile triage. Every finding must pass: "would I pay for this as a triager?"

## Report Structure

```markdown
# [Vuln Class] â€” [Brief Description]

**Severity:** [Critical/High/Medium/Low/Info]
**CVSS:** [Score] ([Vector])
**Affected Asset:** [Exact URL/endpoint/parameter]
**Endpoint:** [Full request method + URL]
**Parameter:** [Which parameter if applicable]

## Summary

[One sentence. What the attacker can do, to what, with what impact.]

## Steps to Reproduce

1. [Action the attacker takes]
2. [Next step]
3. [Continue until impact is demonstrated]

## Evidence

### Request
```http
[Raw HTTP request showing the attack]
```

### Response
```http
[Raw HTTP response showing the impact]
```

### Supporting Evidence
[Screenshots, video, or additional proof]

## Impact

[Specific business impact â€” data exposure scope, user count affected,
financial impact, compliance implications]

## Remediation

[Specific fix, not just "validate input"]
```

## MANDATORY: Demonstrated Impact Standard

**Every finding MUST demonstrate real, concrete damage before it is written up.**

Theoretical "an attacker could..." statements are NOT impact â€” they are hypotheses.

### The gate: "Would I pay for this as a triager?"
If the answer is no because the impact is theoretical â†’ don't report it. Keep testing.

### What counts as demonstrated impact

| Finding type | Demonstrated impact (what you MUST show) | NOT acceptable |
|---|---|---|
| IDOR | Show actual data of another user (name, email, etc.) | "An attacker could access other users' data" |
| SQLi | Show database contents extracted (table names, rows, credentials) | "An attacker could execute arbitrary SQL" |
| XSS | Show cookie/session token stolen, or page rendered with attacker's content | "An attacker could inject malicious script" |
| Auth bypass | Show access to a protected resource as another user/role | "An attacker could bypass authentication" |
| SSRF | Show response from internal service (metadata, internal API) | "An attacker could reach internal services" |
| Privilege escalation | Show admin/higher-priv action performed | "An attacker could escalate privileges" |
| Race condition | Show double-spend, duplicate credit, or state corruption | "An attacker could race the request" |
| Business logic | Show the logic flaw exploited with real values (price, quantity, state) | "An attacker could manipulate the business logic" |
| Open redirect | Show OAuth code/token stolen, or redirect to attacker domain confirmed | "An attacker could redirect users" |
| Account takeover | Show full ATO chain working end-to-end | "An attacker could potentially take over accounts" |

### Impact report format
```
Impact: [What the attacker gets] by [how they get it]

Demonstrated:
- [Concrete evidence 1: e.g., "Retrieved 47 user records including emails, passwords hashes, and API keys"]
- [Concrete evidence 2: e.g., "Accessed admin panel at /admin/users with full CRUD on user accounts"]
- [Concrete evidence 3: e.g., "Extracted AWS access key from metadata endpoint"]

Damage: [What this means for the business â€” data breach, account takeover, financial loss, etc.]
```

## Writing the One-Sentence Summary

Formula: **[Attack method] on [endpoint/parameter] allows [impact] affecting [scope].**

Examples:
- "IDOR on `/api/v1/users/{id}` parameter allows unauthorized access to any user's PII, affecting all platform users."
- "Stored XSS in the profile bio field executes arbitrary JavaScript in other users' browsers, enabling session hijacking."
- "SQL injection in the search parameter allows database extraction, potentially exposing all user credentials."

Bad: "The application has a SQL injection vulnerability."
Good: "SQL injection in the login form's `username` parameter allows full database extraction via time-based blind technique, exposing 50,000+ user records including passwords."

## Evidence Formatting Rules

1. **Always include raw requests** â€” not paraphrased, not screenshots-only
2. **Include timestamps** â€” proves the finding is current
3. **Show the delta** â€” before/after, normal/attack, authorized/unauthorized
4. **Redact only what's necessary** â€” don't redact proof-of-impact
5. **Sequence your evidence** â€” numbered steps matching the reproduction steps
6. **Include response codes and sizes** â€” `200 OK` vs `403 Forbidden` proves access control bypass

## Impact Articulation

Map to concrete consequences, not abstract risks:

| Finding Type | Good Impact Statement | Bad Impact Statement |
|-------------|----------------------|---------------------|
| IDOR | "Access to PII of 10,000+ users including emails, phone numbers, and addresses" | "Broken access control" |
| XSS | "Session token theft via JavaScript exfiltration, enabling full account takeover" | "Cross-site scripting" |
| SQLi | "Extract of all user credentials including bcrypt-hashed passwords" | "Database could be compromised" |
| SSRF | "Access to AWS metadata endpoint exposing IAM credentials for S3 bucket with customer data" | "Server-side request forgery" |
| Business Logic | "Purchase any item for $0.01 by modifying price parameter, affecting all product listings" | "Price can be manipulated" |

## Triage Survival Checklist

Before submitting, verify:

- [ ] **One-sentence summary is specific** â€” names the endpoint, parameter, and impact
- [ ] **Reproduction steps are numbered** â€” triager can follow blind
- [ ] **Evidence includes raw HTTP** â€” request AND response, not just one
- [ ] **Impact is quantified** â€” user count, data types, financial exposure
- [ ] **No theory** â€” every claim is backed by evidence
- [ ] **Remediation is specific** â€” "validate input" is bad; "implement IDOR-resistant reference mapping with server-side authorization checks" is good
- [ ] **Scope confirmed** â€” asset is in-scope per program policy

## Common Rejection Reasons & Avoidance

| Rejection Reason | How to Avoid |
|-----------------|--------------|
| "Informational only" | Chain findings or demonstrate concrete impact |
| "Not reproducible" | Include exact steps, tokens, timestamps |
| "Out of scope" | Double-check `scope.md` before writing |
| "Best practice finding" | Don't report without exploitable impact |
| "Requires authentication" | Provide full auth context, or test unauthenticated paths |
| "Rate limited" | Show abuse potential beyond normal usage |
| "Self-XSS" | Only report if another user can trigger it |
| "Already known" | Search existing reports, add unique angle |

## Type-Specific Templates

### IDOR Template
```markdown
# IDOR â€” [Resource Type] Access via [Parameter]

**Severity:** [High/Critical]
**Endpoint:** `GET /api/[resource]/[id]`

## Summary
IDOR on `[parameter]` in `GET /api/[resource]/[id]` allows access to any user's [resource] by modifying the ID value, affecting [user count] users.

## Steps to Reproduce
1. Authenticate as User A (low-privilege account)
2. Send request: `GET /api/[resource]/[USER_B_ID]`
3. Observe: User B's [resource] data is returned with HTTP 200

## Evidence
[Raw request/response showing access to another user's data]
[Comparison: own resource vs other user's resource]
```

### XSS Template
```markdown
# [Reflected/Stored] XSS â€” [Location]

**Severity:** [High]
**Endpoint:** `[Method] [URL]`

## Summary
[Reflected/Stored] XSS in [parameter/field] executes arbitrary JavaScript in [victim context], enabling [cookie theft/session hijacking/redirect].

## Steps to Reproduce
1. [Navigation or injection step]
2. [Payload delivery step]
3. [Execution trigger step]

## Evidence
[Payload used]
[Response showing payload reflected without encoding]
[Browser screenshot showing execution (if stored)]
```

### SSRF Template
```markdown
# SSRF â€” [Internal Resource Access]

**Severity:** [Critical]
**Endpoint:** `[Method] [URL with parameter]`

## Summary
SSRF via [parameter] allows [internal resource access/cloud metadata retrieval], exposing [specific sensitive data].

## Steps to Reproduce
1. Set up callback: [interactsh/collaborator URL]
2. Send request with callback URL as parameter value
3. Observe callback received at [callback URL]

## Evidence
[Request with SSRF payload]
[Callback evidence showing internal request]
[Data accessed from internal resource]
```

## Machine-readable sidecar (required alongside the markdown report)

Every finding also gets a `<id>-evidence.json` next to the markdown report,
in the exact shape `@skeptic` consumes (see `adversarial-verification`
skill) and `@report` should already have on hand from `@exploit`'s output â€”
don't regenerate it, just carry it forward:

```json
{
  "finding_id": "F014",
  "vuln_class": "IDOR",
  "asset": "https://api.target.com/v2/invoices/{id}",
  "test_accounts": ["disposable_a1@mail.tm", "disposable_b1@mail.tm"],
  "request": "<raw request>",
  "response": "<raw response>",
  "claimed_impact": "<one sentence>",
  "skeptic_verdict": "PASS",
  "severity_recommendation": "High"
}
```

This is what lets `@chain`, `@tracker`, and `oc-checkpoint` cross-reference
findings programmatically instead of re-reading prose â€” small/cheap models
are far more consistent at filling one fixed JSON shape than at writing
free-form summaries a downstream agent then has to re-interpret.

---

## MANDATORY: Strict Submission Doctrine Gate (load LAST, before every report)

Before ANY report is written out, run the candidate through `strict-submission-doctrine`
(plus `mandatory-triage-gate` and `false-positive-filter`). Nothing is reportable until it
passes. This gate is the reputation firewall: **you prefer 0 findings over weak findings.**

### Kill questions (ALL must pass)
1. Real impact DEMONSTRATED, not theorized — name the concrete artifact (data/JS exec/
   internal service/funds).
2. Reproducible under 5 min by a zero-context hostile triager from raw requests.
3. "So what?" answered — a paying triager accepts, not closes-as-Informational.

### Severity honesty rule
Never inflate. Actual severity = evidenced impact. Map to: Critical / High / Medium as
defined by `strict-submission-doctrine`. If a class proof bar (XSS=JS execution, IDOR=other
user's data, SSRF=internal interaction, disclosure=real attack path) is NOT met → not a
finding, at most a Rejected observation or "Needs more validation."

### When nothing qualifies
State plainly: **"No high-quality findings identified that meet submission standards."**
Then list Rejected/Low-Value observations (with reasons) and Still-Untested next steps.
