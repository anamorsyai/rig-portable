---
name: adversarial-verification
description: Second-opinion adversarial review of a finding by a different agent/model than whoever found and technically verified it â€” dedicated to the "would a triager actually pay" judgment call, with a strict machine-checkable output contract. Use before ANY finding reaches @report. Complements mandatory-triage-gate (same gates) but run as an independent pass, not self-review.
---

# Adversarial Verification â€” The Skeptic Pass

## Why this exists as a SEPARATE step from @exploit's verification

`@exploit` verifies a finding is *technically real* (reproduces, causes real
effect). That's necessary but not sufficient â€” the same agent that chased a
lead for 20 minutes is bad at judging "is this actually worth $500+, or did I
talk myself into it?" That's a known bias, not a flaw specific to any model.

This skill runs as `@skeptic` â€” an agent that did NOT find or verify the bug,
seeing ONLY the finished evidence package, with ONE job: try to kill the
finding. If it survives a genuine kill attempt, it goes to `@report`. If not,
it goes back to the testing loop with a specific, named reason.

**This is a judgment pass, not a re-test.** `@skeptic` does not re-run
requests or touch the target â€” it reads the evidence @exploit already
produced and reasons about whether that evidence would survive a real
triager. No target access, no new payloads, no tool calls to the target.

## Input contract (what @skeptic receives)

A single evidence package, nothing else â€” no backstory, no "trust me it's
bad", no prior conversation about how hard this was to find:

```json
{
  "finding_id": "F014",
  "vuln_class": "IDOR",
  "asset": "https://api.target.com/v2/invoices/{id}",
  "test_accounts": ["disposable_a1@mail.tm", "disposable_b1@mail.tm"],
  "request": "GET /v2/invoices/8841 HTTP/1.1\nHost: api.target.com\nAuthorization: Bearer <A1_TOKEN>\n",
  "response": "HTTP/1.1 200 OK\n{\"invoice_id\":8841,\"owner\":\"b1@realcorp.com\",\"amount\":4200.00,\"pdf_url\":\"...\"}",
  "claimed_impact": "Account A1 can read Account B1's private invoice data by ID enumeration, no authorization check on tenant/owner."
}
```

## The kill attempt (run all five, in order, stop at first kill)

1. **Impact check** â€” Is `response` actually another identity's real data, or
   is it A1's own data, a fixture/demo value, or an empty/null field that
   *looks* like a leak but isn't? If it's not concretely someone else's real
   data/money/access â†’ **KILL: not a finding.**
2. **Reproducibility check** â€” Does the evidence contain everything a
   stranger would need (exact endpoint, exact headers/tokens referenced by
   role not by raw secret, exact expected vs actual result)? If a step is
   implicit ("then get an admin token" with no explanation of how) â†’
   **KILL: insufficient repro, send back for a cleaner PoC.**
3. **"So what" check** â€” State the single sentence a hostile triager would
   read. If that sentence is "the server responded" or "behavior differs"
   without a concrete harm noun (data/money/access/availability) â†’
   **KILL: no articulated impact.**
4. **Fresh-account check** â€” Was this reproduced from accounts provisioned
   specifically for this test, not reused/pre-existing sessions? If reused â†’
   **KILL: re-verify from a clean identity before this counts.**
5. **Realism check** â€” Does this require a precondition a real attacker
   wouldn't have (e.g. admin already had to give you the ID out of band, or
   it only works with a race window under 2ms with no automation shown)? If
   the precondition is unrealistic and no PoC of automation is attached â†’
   **KILL: preconditions unrealistic, needs either a realistic path or an
   automation PoC.**

If none of the five kill the finding, it passes.

## Output contract (strict â€” this is what @report and oc-checkpoint consume)

Always emit exactly this JSON, plus one line of human-readable reasoning per
gate. Nothing else. This is intentionally rigid â€” a rigid format is what lets
a smaller model produce a consistently useful judgment instead of a rambling
one.

```json
{
  "finding_id": "F014",
  "verdict": "PASS",
  "gates": {
    "impact":        {"result": "PASS", "note": "response.owner is disposable_b1's real email, not A1's â€” confirmed cross-account data."},
    "reproducibility":{"result": "PASS", "note": "exact request/response present, roles clearly labeled."},
    "so_what":        {"result": "PASS", "note": "\"Any authenticated user can read any other user's invoice + PII by incrementing an integer ID.\""},
    "fresh_account":  {"result": "PASS", "note": "both identities are session-tagged as freshly provisioned for this engagement."},
    "realism":        {"result": "PASS", "note": "sequential ID, no special timing or precondition required."}
  },
  "severity_recommendation": "High",
  "next_action": "send_to_report"
}
```

If any gate fails, `"verdict": "REJECTED"`, `"next_action": "return_to_testing"`,
and the failing gate's `"note"` must say exactly what additional evidence
would flip it to PASS â€” not just "insufficient," but the specific missing
piece (e.g. "show B1's response returned to A1's session, not A1's own data").

## Worked example â€” REJECTED case

Input claim: *"SSRF found â€” server made an outbound request to our
Interactsh listener when we set `webhook_url` to our domain."*

```json
{
  "finding_id": "F022",
  "verdict": "REJECTED",
  "gates": {
    "impact": {"result": "FAIL", "note": "Outbound callback confirms the SSRF primitive exists, but no internal resource, cloud metadata, or internal service response was retrieved â€” this is 'server made a request', not 'attacker read/accessed something'."},
    "reproducibility": {"result": "PASS", "note": "request/response clear."},
    "so_what": {"result": "FAIL", "note": "current sentence is 'server contacted our listener' â€” that's Informational alone. Needs a follow-up request to http://169.254.169.254/latest/meta-data/ or an internal-only endpoint with the actual returned content shown."},
    "fresh_account": {"result": "PASS", "note": "n/a for this class."},
    "realism": {"result": "PASS", "note": "no unusual precondition."}
  },
  "severity_recommendation": "Info (pending escalation)",
  "next_action": "return_to_testing"
}
```

`return_to_testing` note for `@vuln`/`@ssrf`: *"Confirmed SSRF primitive
on `webhook_url`. Next step to reach reportable impact: point it at cloud
metadata endpoint or an internal admin panel and capture the returned body â€”
that's the difference between Informational and High."*

## Wiring into the hunt

`@skeptic` sits between `@exploit`/`@chain` and `@report` in every engagement.
See `/hunt` Phase 4. A finding that never passes `@skeptic` never reaches a
report file â€” this is what keeps the report-acceptance rate high even when
the underlying model doing the hunting is a smaller/cheaper one: the model
doesn't need to be brilliant at judgment on the first pass if a second,
independent pass reliably catches what the first missed.
