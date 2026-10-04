---
name: capability-graph
description: Mechanical method for finding exploit chains across N findings without relying on holistic reasoning over all of them at once. Tags every finding with what it GRANTS and what it REQUIRES, then computes candidate chains as a graph-matching problem before spending any reasoning on them. Use this BEFORE the prose cross-referencing in exploit-chains — it's how you find chains a smaller model would otherwise miss once N findings exceed what fits comfortably in working attention.
---

# Capability Graph — Chain-Finding as Search, Not Insight

## The problem this solves

Ask a model to "look at these 12 findings and find chains" and it will
skim, notice the two or three most obviously-related ones, and miss
combinations buried in the middle of the list — not because it's not
smart enough to see the connection in isolation, but because holding 12
findings in working attention *simultaneously* while cross-referencing all
66 pairs is a different kind of task than reasoning about any one pair.
Senior human red-teamers don't do this by staring harder either — they
externalize it: they write down what each bug gives them, then look at the
list.

This skill is that externalization. It turns "find chains" from "think
really hard about all of this at once" into "fill in a table, then read
off the matches" — a task a smaller model does reliably because it's
mechanical, not because it requires deep one-shot insight.

## Step 1 — Tag every finding (do this the moment @exploit confirms it)

For every confirmed or even ruled-out-but-real finding, fill in:

```json
{
  "finding_id": "F014",
  "grants": {
    "access_level": "read:other_user_data",
    "identity_confusion": "none",
    "data_classes": ["pii", "billing_address"],
    "token_or_secret": null,
    "state_change": null
  },
  "requires": {
    "access_level": "authenticated:any_role",
    "identity_confusion": "none",
    "prior_token_or_secret": null
  },
  "trust_boundary_crossed": "tenant/user isolation"
}
```

`grants` = what the attacker HAS after this bug that they didn't have before.
`requires` = the minimum starting position needed to use this bug at all.
`trust_boundary_crossed` = which assumption the app is silently relying on
that this bug breaks (isolation between users, isolation between roles,
client-side state trusted server-side, rate/quantity assumed non-negative,
etc.) — this field is what lets you spot chains across *different*
vulnerability classes that share a root cause.

## Step 2 — Compute candidates mechanically (not by re-reading everything)

A pair `(A, B)` is a **chain candidate** if any of these is true — check
these four conditions exactly, don't freelance additional ones:

1. `A.grants.access_level` satisfies `B.requires.access_level` (A gets you
   the position B needs to fire) — this is the standard "A unlocks B" chain.
2. `A.grants.token_or_secret` is non-null and matches what `B.requires`
   consumes (a leaked token/secret from A feeds directly into B).
3. `A.grants.data_classes` overlaps with data `B` needs to construct its
   attack (e.g. A leaks a user's email, B is a password-reset flow keyed
   on email).
4. `A.trust_boundary_crossed == B.trust_boundary_crossed` on the SAME
   resource — two different bugs breaking the same isolation boundary
   often compose into a worse version of both (e.g. an IDOR that reveals
   IDs plus a second IDOR that reveals data by ID = full enumeration even
   if neither alone reveals both).

List every pair that satisfies at least one condition. This is your
candidate list — usually 3-6 pairs even out of 12 findings, not 66.
**Only now do you spend reasoning effort — on the candidates, not on the
full set.**

## Step 3 — Test each candidate, then push one level further

For each candidate pair that tests positive: before writing it up, ask the
single most important senior-hunter question — **"now that I have this
combined capability, does a THIRD finding (or a repeat of this same
technique) push it further?"** Concretely:
- Confirmed read access to another tenant's data → can the same trust-
  boundary break also grant write access somewhere?
- Confirmed account takeover on a low-priv account → does an org/team
  structure mean this account can invite/promote itself?
- Confirmed price-zero on one item → does the same logic flaw apply to
  the checkout total, not just the line item?

Do this **once, explicitly, as its own step** — not as an afterthought. A
chain that stops at "I can read another user's invoice" when it could have
gone one step further to "I can also mark it paid" is the single most
common way a real chain gets under-reported at Low/Medium instead of
Critical/High.

## Worked example

Four findings tagged:

| ID | grants.access_level | grants.data_classes | grants.token_or_secret | requires.access_level | trust_boundary |
|----|---|---|---|---|---|
| F010 | read:other_user_profile | email, user_id | null | authenticated:any | user isolation |
| F011 | none (info only) | null | password_reset_token (via predictable generation) | unauthenticated | none |
| F012 | write:own_profile_only | null | null | authenticated:any | none |
| F013 | read:internal_admin_panel | role_list, org_id | null | authenticated:any | role isolation |

Mechanical pass (condition checks only, no free reasoning yet):
- F010 + F011: F010.grants.data_classes includes `email` → email is exactly
  what F011 needs to target a specific victim's predictable reset token
  (condition 3 match). **Candidate.**
- F010 + F013: same `trust_boundary_crossed` category (isolation) but on
  different resources (user vs role) — weak match, note but don't prioritize.
- F012 + anything: `grants.access_level` is write-to-own-data only, `requires`
  nothing special — no other finding's `requires` matches what F012 grants.
  **Not a candidate**, correctly excluded without needing to "think hard"
  about it.

Result: test F010→F011 first (real chain: leak email → target that user's
predictable reset token → account takeover). If confirmed, push further —
does the takeover give access to `F013`'s admin panel because the victim
happened to be an org admin? If yes, this is now a 3-node chain and a much
higher severity than any of the three findings alone.

## Output contract

Write `$PROJECT/chain/capability-tags.json` (array of Step-1 objects, one
per finding) and `$PROJECT/chain/candidates.json` (array of `{pair, matched_condition}`
from Step 2) before writing any chain report. `@report` and `@skeptic` should
be able to see the mechanical justification for why a chain was even
attempted, not just the final narrative.
