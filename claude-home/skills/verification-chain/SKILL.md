---
name: verification-chain
description: The verification pipeline as a chained meta — every candidate finding must traverse self-check → exploit → truth-gate → skeptic (adversarial) → triage verdict before it is reportable. Operates the F001-style chain index across all open findings.
category: meta-orchestration
---

# Verification Chain

> The VERIFY phase as a deterministic chain of gates. Every candidate finding
> advances ONLY by passing each gate in order; a failure returns it to the owning
> agent with a specific reason. This is the chained meta for the verify pipeline.

## The chain (order is mandatory)

```
candidate → 1.self-check → 2.reproduce(@exploit) → 3.truth-gate → 4.skeptic(adversarial-verification) → 5.triage verdict → ACCEPT / REJECT / DEMOTE
```

| Gate | Runs as | Tool/Skill | Exit condition |
|---|---|---|---|
| 1. Self-check | discovering agent | evidence triple under `evidence/<id>/` (request.txt/response.txt/reproduction.ps1/notes.md) | evidence artifacts present and non-empty |
| 2. Reproduce | @exploit (fresh session) | adversarial-verification input contract | deterministic reproduce.ps1 re-runs clean 3/3 |
| 3. Truth-gate | @skeptic/@verifier | `rig.py gate truth <claim_file>` | ≥2 independent sources (artifact + log-correlate + peer) |
| 4. Adversarial review | @skeptic (cold, different agent) | `adversarial-verification` | all 5 kill-attempt gates PASS |
| 5. Triage verdict | triage-lead | `verdict.json` (verified by `rig.py gate verdict <target>`) | ACCEPT only; REJECT/DEMOTE archived with reason |

> **Gate 5 is machine-enforced.** `rig.py gate verdict <target>` FAILS unless every
> finding in `vuln/` has a verdict file that proves the full skeptic contract: PASS
> verdict, all 5 kill-tests PASS with reasoning notes, repro 3/3, severity +
> next_action=send_to_report, and the evidence artifacts present on disk. A bare
> `{"verdict":"PASS"}` is not a verdict.

## Chain index (F001-style living ledger)

Maintain `evidence/F-<id>/cross-check/CHAIN-INDEX.md` — one row per finding,
updated at every gate:

```
| F-id | Class | Gate1 | Gate2 | Gate3 | Gate4 | Gate5 | Verdict | Next |
|------|-------|-------|-------|-------|-------|-------|-------|---------|------|
| F014 | IDOR  | PASS  | PASS  | PASS  | PASS  | PASS  | ACCEPT  | report |
| F022 | SSRF  | PASS  | PASS  | PASS  | FAIL  | —     | REJECT  | return_to_testing |
```

## Rules
- **No skipping.** A finding cannot jump from gate 1 to gate 5; if it does, the
  chain is broken and the finding is frozen until re-verified.
- **No self-verification.** Gate 2 and gate 4 must run in a fresh session by a
  different agent/model than the finder. Single-source claims are not findings.
- **REJECT/DEMOTE archive.** A rejected finding is archived with the failing
  gate + the exact evidence that would flip it (see adversarial-verification
  output contract) — it may re-enter at the failed gate only, not from scratch.
- **Chain atoms.** Only ACCEPT findings become chain atoms for `exploit-chains`
  (P6). A chain built on a REJECT is invalid by construction.
- **Cold double-blind.** Gate 5 (triage-lead) reasons from the evidence package
  alone — no backstory, no "trust me it's bad." Business-impact reasoning only.

## Output
- Per finding: `verdict.json` + `cross-check/CROSS-VERIFICATION-REPORT.md` +
  entry in `CHAIN-INDEX.md`. No verdict → the finding is not reportable.