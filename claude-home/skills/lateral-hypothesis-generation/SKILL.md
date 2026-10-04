---
name: lateral-hypothesis-generation
description: Generates target-SPECIFIC, unusual attack hypotheses before any generic checklist testing begins — the antidote to "just swap the ID and try the standard payload list." Mandatory first step for @vuln and @bizlogic before T1-T3 testing starts. Forces business-context capture, feature-inventory + assumption-reversal, a fixed set of adversarial personas, and retrieval of analogous creative writeups — because "be creative" is not an instruction a model can execute directly, but each of these four sub-steps is.
---

# Lateral Hypothesis Generation — Think About THIS Target, Not The Generic Class

## Read this first — what this skill is actually solving

Telling a model "think of creative scenarios" produces nothing useful,
because "creative" isn't an action — it's a description of the result you
want. What actually produces unusual, target-specific hypotheses is a
handful of concrete, repeatable moves that a human senior hunter does
without noticing they're doing them. This skill makes each move explicit
so a smaller model does them one at a time instead of skipping straight to
"try IDOR on every numeric ID I can find" — which is what happens by
default when the instruction is just "test for vulnerabilities."

**Do not start T1-T4 checklist testing until this skill's four steps
produce a written hypothesis list.** The checklist testing is still how
you confirm each hypothesis — this skill is what decides WHERE to point it.

## Step 1 — Business-context capture (target-specific, not generic)

Before looking at a single endpoint, answer these in 4-5 concrete sentences
about THIS target specifically (not "SaaS apps generally"):

- What is the one thing this product does that makes it money or matters
  to its users? (Not "it's a CRM" — "sales reps log deals and the CRM
  auto-calculates commission payouts.")
- What data or action in this app would a user MOST want to fake, inflate,
  or steal if they could? (commission numbers, review scores, referral
  credit, seat counts, usage limits, ranking position)
- What's the one feature this app has that most competitors DON'T? That's
  usually where the custom, untested-by-anyone-else logic lives.
- Who are the distinct actor types (not just "user" and "admin" — think
  free-tier vs paid, individual vs org-owner, internal staff view, partner/
  reseller, API consumer) and what does each assume the others can't do?

This is the anchor for every hypothesis you generate next — a hypothesis
that doesn't trace back to one of these answers is probably generic and
should be deprioritized versus one that does.

## Step 2 — Feature inventory + assumption reversal

From recon/mapping, list every feature that is NOT generic CRUD (skip
plain login/list/edit — focus on): referral/invite systems, webhooks/
callbacks, bulk import/export, "view as user"/impersonation, multi-currency
or multi-region logic, coupon/loyalty/credit systems, notification/email
templating, integrations/SSO/domain-based auto-join, usage-based billing or
rate limits, file conversion/PDF/report generation, search/autocomplete,
API keys or webhooks issued to third parties, any admin override switch.

For each one found on this target, write the single sentence: **"The
developer built this assuming ___; what happens if that's false?"**
Examples of the pattern (not the answer — apply it to what THIS target has):
- Webhook signature check assumes only the real payment processor knows
  the signing secret — is the secret guessable, leaked in JS, or is the
  check skippable via a legacy/v1 endpoint still running?
- Referral credit assumes one person can't be both referrer and referee —
  can you create the illusion of two identities that are actually you?
- "View as user" (support impersonation) assumes only support staff reach
  it — is it gated by role check or just by a UI button that anyone can
  route around?
- Multi-currency pricing assumes the conversion always happens server-side
  — is the converted price ever trusted from the client?

## Step 3 — Persona sweep (fixed lenses, not "be creative")

Run the feature inventory through each of these five fixed personas, one
pass each. A persona is a narrow lens, not a vague creativity prompt — each
one asks a specific, different question of the same feature list:

1. **The Freeloader** — "How do I get the paid thing for free, forever, or
   repeatedly?" (trial abuse, coupon stacking, downgrade-then-upgrade
   loopholes, quota resets)
2. **The Impersonator** — "How do I become, or act as, someone I'm not?"
   (session/identity confusion, support-view abuse, SSO domain claiming,
   invite-link identity binding)
3. **The Automator** — "What happens if I do this 10,000 times per second
   / with a script instead of by hand?" (races, rate-limit gaps, bulk
   enumeration via a feature that assumes manual one-at-a-time use)
4. **The Data Broker** — "What data here is worth scraping/selling, and
   what's the laziest way to get all of it instead of one record?"
   (pagination without auth re-check, export features, search that returns
   more than the UI displays, GraphQL batching)
5. **The Insider** — "If I were a disgruntled employee/partner with
   legitimate-but-limited access, what's one step further than I'm
   supposed to reach?" (internal tooling exposed at the edge, partner API
   scope creep, admin panels reachable by non-admin auth tokens)

Write one hypothesis per persona per relevant feature — this alone
produces 5-15 target-specific hypotheses from a feature list that generic
checklist testing would have reduced to "test IDOR, test XSS, test SQLi."

## Step 4 — Analogy retrieval (ground creativity in real precedent)

Before finalizing the hypothesis list, use `@writeup-summarizer` /
the writeup archive to search for **prior creative
bugs on the SAME FEATURE TYPE** (not the same company) — e.g. if this
target has a referral system, search "referral program bug bounty",
"loyalty points vulnerability", "webhook SSRF bug bounty" for whichever
features you inventoried in Step 2. Real disclosed writeups are the
single best source of "weird but real" attack shapes — a smaller model
adapting a proven creative bug to a similar feature will consistently beat
the same model trying to invent something equally weird from nothing.
For each retrieved writeup that matches a feature this target has, add one
adapted hypothesis: *"[target]'s [feature] resembles [company]'s in
[writeup] — try the same angle: [adapted technique]."*

## Output contract

Write `$PROJECT/vuln/hypotheses.json` before any T1-T4 testing starts:

```json
{
  "business_context": "<Step 1 answers>",
  "hypotheses": [
    {
      "id": "H01",
      "feature": "referral credit system",
      "persona_or_source": "Freeloader",
      "assumption_reversed": "assumes referrer_id != referee_id enforced server-side",
      "hypothesis": "Create account A, refer account B where B's signup uses A's email+ alias, check if referral credit is granted to A for referring 'itself'",
      "priority": "test before generic IDOR sweep — directly tied to Step 1 business context (this app's revenue model depends on referral cost control)"
    }
  ]
}
```

Feed this list to `@vuln`/`@bizlogic` as the FIRST things tested, ahead of
the generic T1-T3 checklist — the checklist still runs for coverage, but
these hypotheses get first attention and get logged as findings with a
note that they came from Step 1-4 reasoning, which is also useful evidence
for `@skeptic`'s realism gate (a hypothesis traced to the target's actual
business model reads as far less "theoretical" than a generic payload hit).

## Honest limit of this skill

This makes the hypothesis generation *systematic and target-grounded*
instead of generic — it does not manufacture genuine novel insight the
underlying model has no basis for. A bug that requires noticing something
truly unprecedented, with no analogous prior writeup and no clean mapping
to Steps 1-3, is still more likely to be caught by a stronger model doing
open-ended reasoning. What this skill buys you is: stop missing the
"obvious once you see it" creative bugs that generic checklist testing
walks right past, because now something is actually forcing the model to
look at what makes THIS target different before it starts running the
same test list it would run on any target.
