---
description: "Canonical definition of the full autonomous bug bounty engagement. Usage: /hunt <target> [program-name] (or just say \"hunt on <target>\")"
---

# AUTONOMOUS HUNT on: $ARGUMENTS

This is the CANONICAL definition of the autonomous hunt loop; it is NOT required to be
invoked as a slash command. Stated hunt intent (see hunt-operating-contract §1) starts
this loop automatically — `/hunt <target>` is just an explicit alias. The loop is also
defined in `hunt-operating-contract`, which is the single source of truth when the two
differ; keep them in sync.

DEFAULT narration is `live-hunt-mentor`: think aloud, explain the tech/flow, propose
attack surfaces and scenarios, and run the real tests live as you go — like watching a
hunter hunt. Narration is automatic unless the user explicitly asks for silent mode.

SELECTION is situational, not blanket: at every step read STATE.md, pick the highest-value
next action for the CURRENT moment, and fire only the matching tool/skill/feature via
`skill-router` + `situational-decision`. Idle instruments stay off until their moment
arrives; hunt autonomously, never run everything all at once, and keep the 3 pause gates.

Execute the complete stage-gated hunt loop on this target with NO further questions. This
is an authorized bug-bounty engagement.

**FIRST: load the `hunt-operating-contract` skill** (the full doctrine — stages, dispatch,
failure doctrine, **expert-mode depth contract §4b, dynamic scenario generation §4c**, proof
rule, evidence trail, event log). Then follow it exactly.

## 0. Setup (stage gate 1)
Create the project under `~/workspaces/hunts/<target>/` with dirs:
`state/ scope/ recon/ map/ hypotheses/ vuln/ evidence/ reports/`.
Write an EMPTY `scope.md` placeholder and a `STATE.md` with the sections from AGENTS.md §6.
Log `SESSION_START` to the event log. **Do not check scope yet** — recon and testing begin
black-box immediately.

## 1. Recon (stage gate 2)
Dispatch `recon-agent` (load skill `recon-methodology`). It maps the surface into a
prioritized target list written to `recon/map.md`. In parallel dispatch `intel-agent` for
known CVEs/writeups on the target's tech. Do not begin testing until recon has produced the
prioritized list. **Coverage before close:** subdomains + historical URLs + JS mining +
directory fuzzing + tech fingerprint — no single source. Missing any = stage not complete.

## 2. Fuzzing (stage gate 3)
From the map, fuzz directories/params on the highest-priority hosts (skill recon-methodology
§directory fuzzing). Fold discovered endpoints into the target list. Gate: no next stage
until fuzz output is merged into `recon/map.md`.

## 3. Test — unauthenticated (stage gate 4)
Dispatch `hunter` (skills: `idor-bola`, `ssrf`, `sqli`, `xss` as each endpoint warrants).
Test EVERY prioritized endpoint against the classes that fit it, WITHOUT any account.
**Enforce expert mode:** ≥3 different techniques per endpoint×class, unique scenario
generation, anomalies chased, ruled-out only after 3+ logged techniques. Every confirmed
finding → evidence bundle → `verify-agent` → `skeptic`. **Unauth surface is exhausted
before any authenticated testing.** Log negatives and ruled-out surface. **Scope is NOT
checked here and never halts testing** — scope check happens at finding confirmation only.

## 4. Test — authenticated (stage gate 5)
Only after stage 4 is exhausted: use/create the test account(s), then dispatch `hunter`
with `auth-bypass`, `idor-bola` (authed), `bizlogic`, `xss` (stored/blind) skills.
Record accounts in STATE.md.

## 5. Bizlogic (stage gate 6)
Dispatch `hunter` with skill `bizlogic` on stateful/monetary flows (checkout, transfers,
coupons, verification, limits).

## 6. Verify (stage gate 7)
Every finding without a PASS verdict → `verify-agent` (cold reproduction, ≥2 reps, boundary
test). FAIL → back to hunter with the diff. Update evidence bundles with `verdict.md`.

## 7. Chains (stage gate 8)
Dispatch `chain-agent` on all PASSed findings. Verified chains → evidence + skeptic.

## 8. Skeptic + report (stage gate 9)
Dispatch `skeptic` on every PASSed finding and chain (kill-tests ≥3). KILLED → archive with
written reason; DEMOTE → correct severity; PASS → keep. Then dispatch `report-writer` to
produce `reports/F-<id>-report.md` + `reports/index.md`. Run `scope check` per finding —
record IN-SCOPE / not-listed in notes.md — and present the final reportable list to the
user for the in/out-of-scope call.

## Recursion
After the full pass, review STATE.md. If the pass added zero findings, zero escalations,
and zero new surface → re-verify once with fresh eyes (new technique angles) before
declaring done. Otherwise reset the weakest stage (and downstream) and redo
(max 3 passes total). A stage closes only when its coverage checks pass AND expert-mode
depth was genuinely exhausted.

## Closeout
Update STATE.md (every section), log `SESSION_END`, and report: final report paths,
findings + severities, active leads, ruled-out surface, and what needs the user's decision.

## When to pause (ONLY these)
(a) final in/out-of-scope confirmation at report time; (b) anything that could
irreversibly destroy data or systems; (c) budget/termination. Everything else: decide and
execute autonomously.