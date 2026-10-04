---
description: Research a vulnerability class and generate targeted payloads for a specific technology.
---

Generate payloads for: $ARGUMENTS

## Payload Categories

### By Technology
- **PHP**: `phpinfo()`, `system()`, `passthru()`, `shell_exec()`
- **Java**: `Runtime.exec()`, JNDI injection, deserialization gadgets
- **Node.js**: `child_process`, `eval()`, prototype pollution
- **Python**: `os.system()`, `pickle.loads()`, SSTI (Jinja2/Twig)
- **.NET**: `Response.Write()`, `Request.QueryString`, deserialization

### By Vulnerability Class
- **SSRF**: Internal IPs, cloud metadata, protocol smuggling
- **XSS**: Context-specific, filter bypass, polyglots
- **SQLi**: DB-specific, WAF bypass, stacked queries
- **RCE**: Command injection, deserialization, SSTI
- **Path Traversal**: Null bytes, encoding, double traversal

### By WAF
- **Cloudflare**: Origin IP discovery, chunked encoding, case bypass
- **AWS WAF**: JSON encoding, Unicode, chunked transfer
- **Akamai**: Protocol manipulation, HTTP/2 smuggling
- **Custom**: Fuzz with common bypasses

Use the web search MCP to look up:
- Latest CVEs for the target technology
- New bypass techniques
- Tool-specific payloads

Provide organized, copy-paste ready payloads with usage instructions.
