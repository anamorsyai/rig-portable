---
description: "Verify a finding or hypothesis by cold reproduction. Usage: /verify <target> <finding-id-or-dir>"
---

# VERIFY — $ARGUMENTS

Dispatch `verify-agent` to cold-reproduce the given finding (evidence bundle dir) from
scratch: rebuild the request from the artifacts, reproduce ≥2 times, boundary-test the
input, and issue a PASS / DEMOTE / FAIL verdict into `verdict.md`.

Return: verdict, reproduction log (raw requests sent + responses seen), and corrected
severity. If FAIL, list exactly what differed so the hunter can re-test.