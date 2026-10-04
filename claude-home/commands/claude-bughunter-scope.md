---
description: "Report-time scope check — align a confirmed finding with the engagement's in/out-of-scope boundary before reporting. Deny-wins, deterministic, per-finding. NEVER a pre-test gate: testing runs black-box from an empty scope placeholder and scope never halts active hunting (per hunt-operating-contract). Usage: /claude-bughunter-scope <finding-id-or-asset> (or at report time)"
---

# SCOPE CHECK — REPORT TIME only — $ARGUMENTS

Adapted from the claude-bughunter bundle's `/scope` command **to align with our
hunt canon**. This is a **report-time boundary check**, NOT a pre-flight blocker.

## Canon alignment

> Core `hunt-operating-contract` doctrine (authoritative):
> **"Scope is a report-time boundary, never a hunt blocker."** Recon and testing
> begin immediately from an empty `scope.md` placeholder. Scope is consulted only
> AFTER a finding is confirmed, to decide whether to report it. A scope guess is
> never a reason to halt an active test.
>
> This command does NOT change that. It is the deterministic per-finding
> check at report time — deny-wins about *reporting*, never about *testing*.

## When to run

At `report/F-<id>-report.md` time (see `/hunt` stage gates), AFTER confirmation.
Align the confirmed finding's asset against the engagement's scope.

## What it does

1. Load the per-finding scope patterns from `scope/scope.md` (the hunt project's
   scope file written at stage gate 1) or inline patterns.
2. Check each confirmed finding's asset deterministically:
   - `acme.com` → apex + any subdomain
   - `*.acme.com` → any subdomain (NOT bare apex)
   - `api.acme.com` → that exact host
   - `10.0.0.0/8` → any IP in the CIDR
   - `re:^lab[0-9]+\.acme\.io$` → explicit regex
3. Prints `IN-SCOPE` / `OUT-OF-SCOPE <reason>` per finding.

## RFC (report-time flex, canon-correct)

- **In-scope** → include in report.
- **Out-of-scope** → mark `out-of-scope` on the finding, still record evidence in
  `vuln/`; the final in/out-of-scope call at report time belongs to the operator
  (per canon pause-gate (a)).

## Hard rule

This check NEVER runs before or during testing. Testing stays black-box from an
empty scope placeholder. Scope checking at any point other than report-time is a
canon violation.

---

*Adapted from claude-bughunter `scope` (their pre-flight deny-wins gate) to our
report-time boundary doctrine. The colliding `/scope` name is deliberately
avoided — this command is `/claude-bughunter-scope`.*
