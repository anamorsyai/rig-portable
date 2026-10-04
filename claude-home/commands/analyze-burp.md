---
description: Analyze Burp proxy history for interesting endpoints and potential vulnerabilities.
---

Analyze Burp Suite proxy history for: $ARGUMENTS

Use the Burp MCP tools to:

1. **Pull proxy history** — Get all captured requests/responses
2. **Filter by interest** — Look for:
   - Admin panels and debug endpoints
   - Parameter injection points
   - Error messages (500s, stack traces)
   - Authentication-related endpoints
   - File upload handlers
   - API endpoints (especially JSON/GraphQL)
   - WebSocket connections
3. **Identify attack surface** — Group findings by:
   - Authentication/authorization endpoints
   - User input injection points
   - Sensitive data exposure
   - Missing security headers
4. **Prioritize** — Recommend the top 5 targets for manual testing

If $ARGUMENTS is provided, focus the analysis on that specific host, path, or vulnerability class.

Format findings as a prioritized list with:
- URL
- Method
- Interesting parameters
- Why it's interesting (vuln class)
- Recommended test
