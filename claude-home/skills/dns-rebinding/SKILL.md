---
name: dns-rebinding
description: DNS rebinding attacks — defeat hostname-based SSRF protection / same-origin checks to reach internal services and bypass browser SOP. Use when an app validates SSRF destinations by hostname allowlist (SSRF skill) or serves an admin panel protected by host-based access rules.
category: cloud-infra
---

# DNS Rebinding

## Detection
- Target validates SSRF URL against an allowlist using DNS resolution at request time but fetches at a different resolution (TOCTOU) — classic rebinding surface.
- Admin panels gating on `Host`/`Origin` equal to the public domain while reachable from a browser.

## Exploitation
1. **Setup**: register/own a domain with a DNS server that alternates answers (TTL=0): `A → attacker IP` then `A → 169.254.169.254` (or internal IP).
2. **Browsing rebinding**: victim visits `http://attacker-domain:port/`; the app resolves the attacker IP first (page loads), then JS makes a second request that resolves to the internal target → same-origin request to internal service from victim's browser.
3. **SSRF rebinding**: app fetches attacker URL → first resolve allowlist-pass → second resolve internal → internal fetch.
4. **Metadata access**: rebind to `http://169.254.169.254/latest/meta-data/` → steal IAM creds (high impact — only if authorized).
5. **Admin bypass**: rebind `admin.$D` to attacker IP → login form served from attacker, victim types admin creds.

## Payloads
```python
# rebinding nameserver logic (minimal)
import socket, time
last = {}
def resolve(host):
    flip = int(time.time()) % 2
    return "attacker.ip" if flip else "internal.ip"
```

## Tool Commands (Windows)
```powershell
# check target does 2-stage resolution (use with 2 different A answers via owned NS + TTL 0)
Resolve-DnsName -Name "attacker-domain" -Server "your-ns"
curl.exe -s -x http://attacker-proxy "http://attacker-domain/" -v  # watch connect IP change
```

## Verification & Evidence
- Demonstrate first-resolve/second-resolve differ and the second reaches the internal target.
- High severity only when internal service/metadata is actually accessed; otherwise document as TOCTOU gap.