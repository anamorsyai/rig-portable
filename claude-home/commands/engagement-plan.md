---
description: Generate a structured TODO list for a bug bounty engagement on a target.
---

Create a bug bounty engagement plan for: $ARGUMENTS

## Structured TODO

### Recon
- [ ] Subdomain enumeration (subfinder, crt.sh, amass)
- [ ] HTTP probing (httpx)
- [ ] Web crawling (katana)
- [ ] Historical URLs (waybackurls, gau)
- [ ] Technology fingerprinting
- [ ] JavaScript analysis
- [ ] Port scanning (if in scope)

### Testing
- [ ] Security headers check
- [ ] CORS testing
- [ ] Authentication bypass
- [ ] IDOR testing
- [ ] XSS testing (reflected, stored, DOM)
- [ ] SQL injection
- [ ] SSRF testing
- [ ] File upload testing
- [ ] Race conditions
- [ ] Business logic flaws

### Deep Dive
- [ ] Parameter fuzzing on discovered endpoints
- [ ] API endpoint discovery
- [ ] Subdomain takeover check
- [ ] GraphQL introspection (if applicable)
- [ ] JWT analysis (if used)
- [ ] OAuth flow analysis

### Reporting
- [ ] Draft findings with PoCs
- [ ] Write formal reports
- [ ] Calculate CVSS scores
- [ ] Submit to program

Prioritize based on:
1. Highest impact vulnerabilities first
2. Quick wins (missing headers, info disclosure)
3. Technology-specific tests
4. Business logic deep dive

Provide a time-boxed plan (e.g., 4 hours, 8 hours, 1 day).
