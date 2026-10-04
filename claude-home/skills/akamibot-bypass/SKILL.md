---
name: akamibot-bypass
description: Akamai Bot Manager / BotKiller detection evasion for bug bounty and penetration testing â€” covers TLS fingerprint (JA3/JA4), HTTP/2 fingerprint (SETTINGS/WINDOW_UPDATE/PRIORITY frame order), sensor data (/_abck cookie v3), WebGL/canvas fingerprint, behavioral synthesis, residential proxy rotation, and curl-impersonate/curl_cffi/tls-client tool usage.
---

# Akamai Bot Manager Bypass â€” Real-World Techniques

## 1. TLS Fingerprint (JA3/JA3S/JA4)

Akamai fingerprints the TLS handshake â€” cipher suites, TLS extensions, elliptic curves, signature algorithms. If these don't match a real browser, you get blocked immediately.

### curl-impersonate (CLI, C based, BoringSSL fork)
The gold standard for Akamai. Replaces libcurl's TLS stack with BoringSSL patched to match Chrome/Firefox's exact TLS fingerprint.

```bash
# Chrome impersonation (Chrome 124+):
curl-impersonate-chrome -sk --compressed \
  -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36" \
  -H "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8" \
  -H "Accept-Language: en-US,en;q=0.9" \
  -H "Sec-Ch-Ua: \"Chromium\";v=\"124\", \"Google Chrome\";v=\"124\", \"Not-A.Brand\";v=\"99\"" \
  -H "Sec-Ch-Ua-Mobile: ?0" \
  -H "Sec-Ch-Ua-Platform: \"Windows\"" \
  -H "Sec-Fetch-Site: none" \
  -H "Sec-Fetch-Mode: navigate" \
  -H "Sec-Fetch-User: ?1" \
  -H "Sec-Fetch-Dest: document" \
  "https://target.com/"

# Firefox impersonation:
curl-impersonate-ff -sk --compressed \
  -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0" \
  -H "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8" \
  -H "Accept-Language: en-US,en;q=0.5" \
  -H "Accept-Encoding: gzip, deflate, br" \
  "https://target.com/"
```

### curl_cffi (Python)
For scripting â€” wraps the same BoringSSL fork. Supports session reuse and cookie jars.

```python
import curl_cffi

session = curl_cffi.Session(impersonate="chrome124")
resp = session.get(
    "https://target.com/",
    headers={
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.9",
        "Sec-Ch-Ua": '"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"',
        "Sec-Ch-Ua-Mobile": "?0",
        "Sec-Ch-Ua-Platform": '"Windows"',
    },
)
print(resp.status_code, resp.text[:200])
```

## 2. HTTP/2 Fingerprint (Akamai's Specialty)

Akamai checks HTTP/2 frame behavior more aggressively than any other CDN:

- **SETTINGS frame** â€” Chrome sends specific settings in a specific order (initial flow control window 65535, max concurrent streams 1000, etc.)
- **WINDOW_UPDATE frame** â€” initial update frame size and timing
- **PRIORITY frame** â€” stream priorities must match Chrome's pattern
- **Pseudo-header order** â€” `:method`, `:path`, `:authority`, `:scheme`
- **HEADERS frame compression** â€” HPACK dictionary state and dynamics

curl-impersonate handles this automatically via BoringSSL patches. Normal curl cannot bypass Akamai even with matching TLS ciphers.

## 3. _abck Cookie & Sensor Data V3

Akamai Bot Manager v3 drops `_abck` after running `sensor.js` â€” a JS challenge that detects:

- **Polynomial constants**: Generated at VM level, matched server-side
- **Stack trace analysis**: Headless browsers have different call-stack depths
- **console.log hook detection**: Checks for devtools overrides
- **Accumulator value**: Computed from all sensor tests, embedded in `_abck`
- **Canvas/WebGL fingerprint**: Text rendering + WebGL rendering must match real device
- **AudioContext fingerprint**: Oscillator output used as entropy
- **Navigator properties**: `webdriver`, `plugins`, `languages`, `hardwareConcurrency`
- **Performance timing**: JS execution timing diffs between real and headless

### Strategy A: Real browser handshake (most reliable)
Export `_abck` cookie via Playwright, reuse with curl_cffi:
```python
# browser = playwright.chromium.launch(headless="shell")
# page = browser.new_page()
# page.goto("https://target.com/")
# cookies = page.context.cookies()
# abck = [c for c in cookies if c["name"] == "_abck"][0]
# session = curl_cffi.Session(impersonate="chrome124")
# session.cookies.set("_abck", abck["value"])
```

### Strategy B: Cookie replay trick
Sometimes Akamai only checks TLS/H2 on the first request. Reuse `_abck` for ~30-60 min:
```bash
oc-playwright "https://target.com/"
grep _abck requests.jsonl | head -1
curl-impersonate-chrome -sk --cookie "_abck=VALUE" "https://target.com/api/..."
```

### Strategy C: Proxy real browser requests through Burp
```bash
oc-playwright "https://target.com/sensitive" --burp
# Traffic captured with real TLS + cookie, replay via Repeater
```

## 4. IP Reputation & Residential Proxies

Datacenter IPs die fast on Akamai. Residential/mobile IPs survive longer.

```bash
# With oxylabs/BrightData residential:
curl-impersonate-chrome -sk \
  -x "http://customer-USER:PASS@proxy.provider.io:7777" \
  "https://target.com/"

# Tor (SOCKS5) â€” mixed results, Akamai flags Tor exit IPs:
curl-impersonate-chrome -sk -x "socks5://127.0.0.1:9050" "https://target.com/"

# Origin IP bypass (if historical DNS reveals it):
curl-impersonate-chrome -sk --resolve "target.com:443:ORIGIN_IP" "https://target.com/"
```

## 5. Behavioral Synthesis (Phase 3)

If you hit deep sensor analysis (phase 3), you need:
- Random delays (1-3s between requests)
- Variable scroll speed, mouse path patterns
- Realistic page lifecycle events (focus/blur/visibilitychange)
- Playwright with real browser â€” no synthetic behavior

## 6. Testing Strategy

```bash
# Check for Akamai:
curl -sI "https://target.com/" | grep -iE "akamai|akamaighost|X-Akamai"

# Check for bot challenge:
curl -s "https://target.com/" | grep -iE "abck|akamai|botmanager|challenge"

# Test with curl-impersonate-chrome:
curl-impersonate-chrome -sk "https://target.com/" -o /dev/null -w "%{http_code}"

# If 200: bypass works.
# If 403/503: test with residential proxy or Playwright.

# If all CLI fails â€” Playwright real browser:
oc-playwright "https://target.com/api/..."
```

## 7. Known Limitations

- curl-impersonate is not perfect â€” HTTP/2 frame timing/continuation patterns differ from real Chrome
- `_abck` cookie TTL: 30-60 min, then needs fresh sensor handshake
- Akamai phases: Phase 1 (TLS/H2/IP) ~70% block; Phase 2 (sensor JS) ~95%; Phase 3 (behavioral+ML) ~99%
- No public tool fully bypasses Phase 3 â€” use real browser proxied through Burp
