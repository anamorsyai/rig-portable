---
name: improper-inventory
description: Improper asset inventory (API9) — shadow API endpoints, old versions, debug/staging routes, deprecated methods left exposed. Use when the app has versioned APIs, multiple hosts, or legacy routes that admins forgot to retire.
category: api
---

# Improper Inventory (Shadow/Deprecated APIs)

## Detection
- Version enumeration: `/v1`, `/v2`, `/v3`, `/api/1.x`, `/api/latest`.
- Legacy routes: `/api/old`, `/migration`, `/batch-legacy`, `/internal-import`.
- Debug endpoints: `/debug`, `/actuator`, `/metrics`, `/test`, `/sandbox`, `/staging`.
- Compare behavior across versions — old versions often lack new authz/injection fixes.

## Exploitation
1. **Old version missing authz fix**: `/v1/admin/users` unauthenticated while `/v2/...` requires token.
2. **Deprecated endpoint returns sensitive data**: `/v1/users/export` bypasses `/v2` rate limits/filters.
3. **Shadow host**: discover `*-staging.*`, `*-dev.*`, `api-legacy.*` via subdomain enumeration (recon-workflow) — often weaker auth.
4. **Debug endpoints leak**: `/debug/vars`, `/status`, `/metrics` expose stack/config/secrets.
5. **Mixed API styles**: SOAP/XML legacy alongside REST new — test both (XXE surface).

## Tool Commands (Windows)
```powershell
# enumerate API version prefixes (requires ffuf)
# ffuf -w versions.txt -u "$U/FUZZ/users" -mc 200,401,403 -o shadow.json
# subdomain shadow inventory (requires subfinder + httpx)
# subfinder -d example.com -silent | Select-String -Pattern 'dev|staging|test|old|legacy|internal'
# httpx -l shadow-subs.txt -mc 200 -silent
# Manual enumeration alternative
$versions = @("v1", "v2", "v3", "api/1.x", "api/latest", "api/old", "debug", "actuator", "metrics", "test", "sandbox", "staging")
foreach ($v in $versions) { $code = curl.exe -s -o NUL -w "%{http_code}" "$U/$v"; Write-Host "$code $v" }
```

## Verification & Evidence
- Proof = legacy/shadow endpoint reachable with weaker security (data access or authz difference vs current API).
- Map endpoint → access level → impact.