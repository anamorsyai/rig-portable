---
name: headers-audit
description: HTTP security headers audit — missing CSP/HSTS/X-Frame-Options/X-Content-Type-Options/Referrer-Policy/Permissions-Policy and cookie flags. Use on every host; low-severity by itself, chainable with XSS/clickjacking/session issues.
category: misconfig-exposure
---

# HTTP Security Headers Audit

## Detection
```powershell
curl.exe -s -I "$U/" 
```
Check for: `Content-Security-Policy`, `Strict-Transport-Security` (with includeSubDomains + preload + max-age≥31536000), `X-Frame-Options`/`frame-ancestors`, `X-Content-Type-Options: nosniff`, `Referrer-Policy`, `Permissions-Policy`, `Cross-Origin-Opener-Policy`, `Cross-Origin-Resource-Policy`.

## Exploitation
1. **Missing X-Frame-Options / frame-ancestors** → clickjacking (build proof-of-concept overlay page; only a finding if sensitive action is clickjackable AND UI has no additional protections).
2. **Missing CSP** → reflected XSS not mitigated (chain with a real XSS).
3. **Missing HSTS** → MITM downgrade for the domain (report only if the site handles sensitive data; usually informational).
4. **Weak cookie flags**: session cookie without `HttpOnly`/`Secure`/`SameSite` → chain with XSS/subdomain issues.
5. **X-Content-Type-Options missing** → MIME sniffing (polyglot XSS in uploads).

## Tool Commands (Windows)
```powershell
curl.exe -s -I -H 'User-Agent: Mozilla/5.0' "$U/" | Select-String -Pattern 'set-cookie|content-security|strict-transport|x-frame|content-type|x-content-type|referrer-policy' -CaseSensitive:$false
# cookie audit
curl.exe -s -i "$U/login" -X POST -d 'u=a&p=b' | Select-String -Pattern 'set-cookie' -CaseSensitive:$false
```

## Verification & Evidence
- Report as its own low only for clickjacking PoC or policy/flag gaps on auth/session cookies.
- Otherwise fold into a chained finding.