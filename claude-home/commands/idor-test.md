---
description: Test for IDOR vulnerabilities across user/resource endpoints.
---

IDOR testing on: $ARGUMENTS

Use Burp MCP and tools to:

1. **Map resource endpoints** — Pull proxy history and find endpoints with numeric/string IDs in:
   - URL paths: `/api/user/123`, `/profile/456`
   - Query params: `?id=789`, `?user_id=101`
   - POST bodies: `{"account_id": "112"}`

2. **Horizontal IDOR tests**
   - Change IDs to other valid users: 101 -> 102, 100 -> 200
   - Try sequential, random, and common IDs
   - Test with authenticated vs unauthenticated
   - Check if responses differ (200 vs 403 vs 404)

3. **Vertical IDOR tests**
   - Access admin endpoints with regular user token
   - Try admin ID ranges (0, 1, 999, -1)
   - Check for parameter pollution: `?admin=true`

4. **Global IDOR tests**
   - UUID/GUID endpoints: predictable or brute-forceable?
   - Reference ID exposure in responses
   - Leaked in JavaScript or API responses

5. **Chaining**
   - IDOR + information disclosure
   - IDOR + missing auth
   - IDOR + privilege escalation

For each finding provide:
- Request/response pair
- Impact assessment (what data/access is exposed)
- Working PoC
- Severity rating
