---
name: coverage-tracker
description: Track tested vs untested endpoints, vuln-classes, parameters. Maintains coverage matrix. Gap analysis after every phase. Use when checking progress or deciding next dispatch.
---

# Coverage Tracker

Maintains a coverage matrix showing exactly what's been tested and what hasn't.

## Coverage Matrix Format (in PROGRESS.md)

```markdown
## Coverage Matrix

| Endpoint | T1-RCE | T1-AuthBypass | T1-SQLi | T1-SSRF | T2-IDOR | T2-Privesc | T2-XSS | T3-CSRF | T3-Race | Status |
|----------|--------|---------------|---------|---------|---------|------------|--------|---------|---------|--------|
| /api/users | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | — | — | partial |
| /api/admin | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | **GAP** |
| /login | ✓ | ✓ | — | — | — | — | — | — | — | partial |
```

Legend:
- ✓ = tested (pass or fail, either way it's covered)
- ✗ = NOT tested (this is a gap)
- — = not applicable (e.g., no user input = no SQLi surface)
- Status: done / partial / gap

## How to update

### After every agent returns
1. Agent reports which endpoint/param it tested
2. Update the matrix row for that endpoint
3. Mark tested vuln-classes with ✓
4. Mark untested ones with ✗

### After every phase completion
1. Scan matrix for any row with Status = "gap" or "partial"
2. For each gap: identify which vuln-class is untested
3. Dispatch the responsible agent:
   - RCE/SQLi/SSRF/Auth bypass → @vuln
   - IDOR/Privesc → @vuln or @bizlogic
   - Race/Logic → @bizlogic
   - XSS/CSRF → @vuln
4. Don't move to next priority tier until current tier is fully covered

### Gap priority (what to test first)
1. P0 endpoints with gaps (auth, payment, API, admin)
2. P1 endpoints with gaps (upload, profile, settings)
3. P2 endpoints with gaps (search, filter, import/export)
4. P3 endpoints with gaps (everything else)

## Coverage summary (at bottom of PROGRESS.md)

```markdown
## Coverage Summary
- Total endpoints mapped: N
- Fully tested (all applicable T1-T3): N (X%)
- Partially tested: N
- Untested (gaps): N
- T1 coverage: X/Y endpoints
- T2 coverage: X/Y endpoints
- T3 coverage: X/Y endpoints
- Findings found: N
- Findings chained: N
- Findings reported: N
```

## The rule
**If "Untested (gaps)" > 0, keep dispatching.** Only stop when all applicable vuln-classes are covered on all mapped endpoints.
