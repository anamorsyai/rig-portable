---
name: security-misconfiguration
description: General security misconfiguration (OWASP A05 / API8) — default credentials, verbose errors, directory listing, exposed debug consoles, stack traces, missing security controls. Use as a catch-all checklist during recon on every host.
category: misconfig-exposure
---

# Security Misconfiguration

## Detection
```powershell
curl.exe -s -i "$U/" -H 'Accept: application/json'
curl.exe -s -i "$U/sitemap.xml" | Select-Object -First 3
curl.exe -s -i "$U/" | Select-String -Pattern 'server:|x-powered-by:|set-cookie|stack trace|debug' -CaseSensitive:$false
```
- Look for: default creds pages (phpmyadmin, grafana, jenkins, admin), verbose errors (400/500 with stack/PII), directory listing (autoindex), unused but live features (upload, debug, test), demo/example endpoints.

## Exploitation
1. **Default credentials**: `admin:admin`, `admin:password`, `root:root`, vendor defaults (HackerOne-visible default docs exist for many apps).
2. **Directory listing**: `/files/`, `/uploads/`, `/backups/` with `Index of` → sensitive files.
3. **Verbose errors**: stack traces leak framework/paths/versions → targeted exploit selection; sometimes SQL queries with data.
4. **Debug consoles**: `/debug`, `/console`, `/phpinfo.php`, `/actuator`, `/metrics`.
5. **Missing security controls**: no authz on feature toggles, no CSRF on state-changing GETs, CORS `*` with credentials.
6. **Exposed test/demo data**: `/test`, `/demo`, sample users.

## Tool Commands (Windows)
```powershell
# ffuf -w misconfig.txt -u "$U/FUZZ" -mc 200,204,302,401 -o misconf.json
curl.exe -s "$U/nonexistent" -i | Select-String -Pattern 'stack|trace|error' -CaseSensitive:$false
curl.exe -s -i "$U/uploads/" | Select-Object -First 5   # dir listing
```

## Verification & Evidence
- Must show concrete exposure (data, console, creds) — not just "missing header".
- Capture exact path + response snippet.