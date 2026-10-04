---
description: Test a specific endpoint for vulnerabilities using Burp Repeater/Intruder.
---

Test the endpoint: $ARGUMENTS

Use Burp MCP to send requests and analyze responses. Test for:

1. **Injection**
   - SQL: `'`, `"`, `1 OR 1=1`, `'; DROP TABLE--`
   - NoSQL: `{"$gt":""}`, `{"$ne":null}`
   - Command: `; ls`, `| cat /etc/passwd`, `` `whoami` ``
   - Template: `{{7*7}}`, `${7*7}`, `<%= 7*7 %>`

2. **XSS**
   - `<script>alert(1)</script>`, `<img onerror=alert(1)>`
   - Event handlers: `" onfocus=alert(1) autofocus="`
   - Context-dependent payloads

3. **Authentication**
   - Missing auth: Remove auth headers/tokens
   - JWT: None algorithm, weak secret, expired token
   - Session: Fixation, predictable tokens

4. **Authorization**
   - IDOR: Change user/resource IDs
   - BAC: Access with lower-privilege user
   - Horizontal/vertical escalation

5. **Logic**
   - Price manipulation
   - Race conditions (parallel requests)
   - Business logic bypass

6. **SSRF**
   - Internal IPs: `127.0.0.1`, `169.254.169.254`
   - Protocol smuggling: `file:///etc/passwd`
   - DNS rebinding

Provide:
- Each test with exact payload
- Response analysis
- Whether the test indicates a vulnerability
- Severity if confirmed
