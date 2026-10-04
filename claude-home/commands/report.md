---
description: "Write final report for verified, skeptic-passed findings. Usage: /report <target>"
---

# REPORT — $ARGUMENTS

Dispatch `report-writer` to convert all verified, skeptic-PASSED findings (and verified
chains) into `reports/F-<id>-report.md` files plus `reports/index.md`. Honor skeptic
severity corrections. Present the reportable list to the user for the final scope call.

Return: report paths, findings + severities, and anything the user must confirm before
submission.