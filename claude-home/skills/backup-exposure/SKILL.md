---
name: backup-exposure
description: Backup / sensitive-file exposure — .bak, .sql, .zip, .tar.gz, config files, DB dumps, editor swap files left on the web root. Use during recon on every discovered path/parameter.
category: misconfig-exposure
---

# Backup & Sensitive File Exposure

## Detection
```powershell
$paths = @("db.sql", "db.sql.gz", "backup.zip", "backup.tar.gz", "backup.sql", "dump.sql", "config.php", "config.json", ".env", "app.bak", "index.php.bak", ".htaccess.swp", "web.config", ".DS_Store", "robots.txt")
foreach ($p in $paths) {
    $code = curl.exe -s -o $null -w '%{http_code}' "$U/$p"
    echo "$code /$p"
}
```
- fuzz with extensions: `file.php.bak`, `file.php~`, `file.php.swp`, `file.php.old`, `file.save`, `file~`, `file.zip`, `file.tar.gz`.
- Check status codes: 200 with content-type `application/zip|octet-stream|text/plain` = hits. 403/404 ambiguous — try 2nd request with HEAD or range.

## Exploitation
1. **DB dump** → full data exposure (credentials, PII, session tokens).
2. **Config backup** → secrets, API keys, DSNs.
3. **Source backup** (`.zip`/`.tar`) → full source, hardcoded secrets, internal architecture → chains.
4. **Editor swap** (`.swp`/`.swo`) → partial source reveals function names/paths.
5. **.env / .env.bak** → keys.
6. **Log files** (`access.log`, `error.log`, `*.log`) → tokens in query strings, internal paths.

## Tool Commands
```powershell
# extension fuzz on known endpoints
ffuf -w extensions.txt -u "$U/user/profileFUZZ" -mc 200 -fs 1234
# content-type triage
curl.exe -s -i "$U/db.sql.gz" | findstr /i "content-type content-length"
```

## Verification & Evidence
- Confirmed sensitive content (dump/config/source) — quote a redacted snippet.
- Record exact path, status, content-type, size.