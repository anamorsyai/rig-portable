---
description: "Run hostile-triage skeptic review on a finding before report. Usage: /skeptic <target> <finding-id-or-dir>"
---

# SKEPTIC — $ARGUMENTS

Dispatch `skeptic` to review the given finding as a hostile triager: run ≥3 kill-tests
(reproducibility, scope, preconditions, impact, novelty) and write the verdict
(KILLED / DEMOTE / NOTE / PASS) + justification into the finding's `skeptic.md`.

Return: verdict + one-paragraph triager-view justification. KILLED findings must not be
reported unless chained or re-tested first.