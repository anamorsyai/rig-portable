---
name: parameter-fuzzing
description: Hidden parameter discovery — debug parameters, admin overrides, feature flags, mass parameter injection, parameter pollution. Arjun/ffuf optimized configs for parameter discovery.
---

# Parameter Fuzzing — Hidden Parameter Discovery

## 1. Why Parameter Fuzzing is High-Value

A single hidden parameter like `?debug=1`, `?admin=true`, `?source=1`, `?test=1`, or `?__internal=true` can instantly escalate a finding from Low to Critical. Hidden parameters unlock admin functionality, bypass access controls, expose debug information, and reveal internal API behavior that was intentionally hidden from the frontend.

**The rule: fuzz every endpoint with every parameter wordlist before concluding a vulnerability class doesn't apply.**

## 2. Parameter Wordlists

### 2.1 Built-in Wordlist
```bash
ls C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/parameters.txt
```

### 2.2 Comprehensive Hidden Parameter List

```bash
cat > C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/params-all.txt << 'EOF'
# Debug / Testing
debug
debug=true
debug=1
debug_mode
test
testing
test_mode
mock
mock_mode
staging
dev
development
dry_run
simulate
preview
preview_mode
sandbox
trial

# Admin / Override
admin
adm
is_admin
isAdmin
admin_mode
adminOverride
superuser
super_user
su
root
internal
private
bypass
bypass_auth
skip_auth
skip_authorization
skip_verification
skipValidation
override
force
force_execute
force_update
forceDelete
force_delete
no_auth
noAuth
no_check
noCheck
disable_auth
disableAuth
allow_all

# Config / Environment
env
environment
config
configuration
settings
setup
install
migrate
rebuild
reset
restore
method
action
cmd
command
exec
execute
run
source
src
edit
mode
view

# Info Disclosure
phpinfo
info
version
status
health
ping
stats
statistics
metrics
about
detail
details
full
all
show_all
list_all
debug_info
verbose
trace
profile
perf
performance
memory
sql
query
explain
plan

# IDOR / Access
id
user_id
uid
userid
account_id
accountid
customer_id
customerid
profile_id
profileid
org_id
orgid
company_id
companyid
team_id
teamid
workspace_id
type
object_type
resource_type
include
scope
filter
fields
select
embed
expand

# File operations
file
filepath
path
dir
directory
folder
filename
name
template
page
include_path
require
load
import
export
download
upload
read
write
save
backup
restore

# Auth / Tokens
token
api_key
apikey
apiKey
access_token
accessToken
secret
key
key_id
client_id
client_secret
session
auth
authorization
signature
hmac
checksum
hash
nonce
timestamp

# Pagination / Limits
limit
per_page
page
offset
count
total
max
min
start
end
since
before
after
from
to
sort
order
dir
direction

# API versioning
api_version
version
v
ver

# Content negotiation
format
fmt
ext
extension
callback
jsonp
callback
redirect
return_url
returnTo
next
url
goto
destination

# Feature flags
feature
features
flag
flags
ff
feature_flag
toggle
experiment
ab
bucket
rollout
enabled
disabled
active
inactive
enable
disable

# Misc high-value
__proto__
constructor
prototype
callback
ng
eval
expression
template
partial
fragment
section
component
widget
module
plugin
addon
integration
webhook
hook
notification
alert
webhook_url
callback_url
redirect_url
return_url
EOF

echo "params-all.txt: $(wc -l < C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/params-all.txt) params"
```

### 2.3 Target-Specific Parameter Extraction

```bash
# Extract parameters from JS files
cat tmp/js-files.txt | while read url; do
  curl -sk "$url" | grep -oP '(?:["'"'"'])([a-zA-Z_][a-zA-Z0-9_]*)(?:["'"'"'])\s*[:=]\s' | \
    tr -d '"'"'"':= ' | sort -u
done | sort -u > tmp/js-params.txt

# Extract parameters from URL paths
cat tmp/all-urls.txt | grep -oP '\?[^#]+' | tr '?&' '\n' | grep '=' | \
  cut -d= -f1 | sort -u > tmp/url-params.txt

# Merge for target-specific parameter fuzzing
cat tmp/js-params.txt tmp/url-params.txt C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/params-all.txt | \
  sort -u > tmp/target-params.txt
```

## 3. Parameter Discovery with Arjun

### 3.1 Basic Arjun Usage

```bash
# Single endpoint
arjun -u https://TARGET.COM/api/endpoint \
  --get \
  -t 10 \
  -o tmp/arjun-get.json

# POST endpoint with JSON body
arjun -u https://TARGET.COM/api/endpoint \
  --post \
  --headers "Content-Type: application/json" \
  -d '{"known_param": "value"}' \
  -t 10 \
  -o tmp/arjun-post.json

# POST with form data
arjun -u https://TARGET.COM/api/login \
  --post \
  --headers "Content-Type: application/x-www-form-urlencoded" \
  -d "username=test&password=test" \
  -t 10 \
  -o tmp/arjun-form.json
```

### 3.2 Bulk Arjun from URL List

```bash
# Test multiple endpoints in sequence
cat tmp/live-endpoints.txt | while read url; do
  echo "[*] Testing: $url"
  arjun -u "$url" \
    --get \
    -t 5 \
    -o "tmp/arjun-$(echo $url | md5sum | cut -c1-8).json" 2>/dev/null
done

# Parallel arjun (limited by rate limits)
cat tmp/live-endpoints.txt | xargs -P3 -I{} sh -c '
  arjun -u "$1" --get -t 5 -o "tmp/arjun-$(echo "$1" | md5sum | cut -c1-8).json" 2>/dev/null
' _ {}
```

### 3.3 Arjun with Custom Wordlist

```bash
arjun -u https://TARGET.COM/api/endpoint \
  --get \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/params-all.txt \
  -t 10 \
  -o tmp/arjun-custom.json
```

## 4. Parameter Fuzzing with ffuf

### 4.1 GET Parameter Discovery

```bash
# Discover hidden GET parameters
ffuf -u "https://TARGET.COM/api/endpoint?FUZZ=1" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/parameters.txt \
  -t 30 -rate 50 \
  -mc 200,201,401,403,500 \
  -fc 404 \
  -fs $(curl -s "https://TARGET.COM/api/endpoint" | wc -c) \
  -o tmp/params-get.json -of json

# Filter by response size to find params that change behavior
ffuf -u "https://TARGET.COM/api/endpoint?FUZZ=1" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/params-all.txt \
  -t 30 -rate 50 \
  -mc 200 \
  -fs $(curl -s "https://TARGET.COM/api/endpoint" | wc -c) \
  -o tmp/params-get-diff.json -of json
```

### 4.2 POST Parameter Discovery

```bash
# JSON body parameters
ffuf -u "https://TARGET.COM/api/endpoint" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/params-all.txt \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"FUZZ": "test"}' \
  -t 20 -rate 30 \
  -mc 200,201,400,401,403,500 \
  -o tmp/params-post.json -of json

# Multiple parameters at once
ffuf -u "https://TARGET.COM/api/endpoint" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/params-all.txt \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"test": "value", "FUZZ": "test"}' \
  -t 20 -rate 30 \
  -mc 200,201,400,500 \
  -o tmp/params-post-multi.json -of json
```

### 4.3 Bulk Endpoint Parameter Fuzzing

```bash
# Fuzz parameters across all discovered endpoints
cat tmp/api-endpoints.txt | while read endpoint; do
  ffuf -u "$endpoint?FUZZ=1" \
    -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/params-all.txt \
    -t 15 -rate 20 \
    -mc 200,401,403,500 \
    -fs $(curl -s "$endpoint" | wc -c) \
    -o "tmp/params-$(echo $endpoint | md5sum | cut -c1-8).json" -of json
done
```

## 5. Parameter Pollution Testing

### 5.1 HTTP Parameter Pollution (HPP)

```bash
# Duplicate parameters — which one wins?
curl -s "https://TARGET.COM/api/search?q=test&q=admin"
curl -s "https://TARGET.COM/api/users?role=user&role=admin"
curl -s "https://TARGET.COM/api/admin?access=false&access=true"

# Array-style parameters
curl -s "https://TARGET.COM/api/users?id[]=1&id[]=2&id[]=3"

# PHP parameter pollution
curl -s "https://TARGET.COM/page.php?%23=test"  # Fragment injection
curl -s "https://TARGET.COM/page.php?a=1&a=2"    # Last param wins (PHP)
curl -s "https://TARGET.COM/page.php?a[x]=1&a[y]=2"  # Array injection

# ASP.NET parameter pollution
curl -s "https://TARGET.COM/page.aspx?a=1&a=2"   # First param wins (ASP)
```

### 5.2 JSON Parameter Pollution

```bash
# Duplicate JSON keys
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"role": "user", "role": "admin", "email": "test@test.com"}'

# Nested parameter pollution
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"user": {"role": "user"}, "user": {"role": "admin", "id": 1042}}'

# JSON prototype pollution
curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"__proto__": {"isAdmin": true}, "email": "test@test.com"}'

curl -s -X POST "https://TARGET.COM/api/update" \
  -H "Content-Type: application/json" \
  -d '{"constructor": {"prototype": {"isAdmin": true}}, "email": "test@test.com"}'
```

## 6. Hidden Parameter Categories

### 6.1 Debug Parameters (Highest Value)

```bash
# Parameters that reveal debug/stack trace information
for param in debug test error verbose trace dev development \
             staging mock sandbox preview dry_run simulate \
             phpinfo info status health metrics stats; do
  
  # GET
  RESP=$(curl -s "https://TARGET.COM/api/endpoint?$param=1" \
    -H "Authorization: Bearer $TOKEN")
  LEN=$(echo "$RESP" | wc -c)
  echo "$param (GET) = $LEN bytes"
  echo "$RESP" | grep -qi "error\|exception\|stack\|debug\|password\|secret\|key\|token" && \
    echo "  [!] INTERESTING: $param"
done

# Debug with verbose mode
for param in verbose detailed full all show_all expand include embed; do
  RESP=$(curl -s "https://TARGET.COM/api/users/me?$param=true" \
    -H "Authorization: Bearer $TOKEN")
  LEN=$(echo "$RESP" | wc -c)
  echo "$param = $LEN bytes (normal: $(curl -s "https://TARGET.COM/api/users/me" | wc -c))"
done
```

### 6.2 Admin Override Parameters

```bash
# Parameters that bypass access controls
for param in admin is_admin adminOverride superuser su internal \
             override bypass force no_auth skip_auth skip_validation; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/admin/users?$param=true" \
    -H "Authorization: Bearer $TOKEN")
  echo "$param=true -> HTTP $STATUS"
done

# Role override
for role in admin superadmin root administrator sysadmin owner staff; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/me?role=$role" \
    -H "Authorization: Bearer $TOKEN")
  echo "role=$role -> HTTP $STATUS"
done
```

### 6.3 Content Override Parameters

```bash
# Parameters that change response format (info disclosure)
for param in format fmt ext json xml html text raw source src; do
  curl -s "https://TARGET.COM/api/users/me?$param=json" \
    -H "Authorization: Bearer $TOKEN" | head -c 200
  echo "---"
done

# Parameters that show hidden content
for param in show_all all full detail details expand include embed \
             scope filter fields select view mode; do
  RESP=$(curl -s "https://TARGET.COM/api/users/me?$param=true" \
    -H "Authorization: Bearer $TOKEN")
  LEN=$(echo "$RESP" | wc -c)
  DEF=$(curl -s "https://TARGET.COM/api/users/me" | wc -c)
  [ "$LEN" -gt "$DEF" ] && echo "[!] $param expands response: $DEF -> $LEN bytes"
done
```

### 6.4 Execution / Command Parameters

```bash
# Parameters that may execute actions
for param in action cmd command exec execute run method mode do; do
  for value in list create update delete export import backup restart \
               shutdown reload clear flush reset test verify validate; do
    curl -s "https://TARGET.COM/api/endpoint?$param=$value" \
      -H "Authorization: Bearer $TOKEN" | head -c 100
    echo " ($param=$value)"
  done
done
```

## 7. Automated Parameter Fuzzing Script

```python
#!/usr/bin/env python3
"""
Automated hidden parameter discovery scanner.
Tests GET, POST (JSON + form), and header parameters.
"""
import requests
import concurrent.futures
import argparse
import json
import sys
from urllib.parse import urlparse, urlencode
from colorama import init, Fore, Style

init()

class ParameterScanner:
    def __init__(self, url, method="GET", headers=None, body=None, threads=20):
        self.url = url
        self.method = method.upper()
        self.headers = headers or {}
        self.body = body or {}
        self.threads = threads
        self.session = requests.Session()
        self.session.headers.update(self.headers)
        self.baseline = self._get_baseline()
        self.findings = []
    
    def _get_baseline(self):
        """Get baseline response for comparison"""
        try:
            if self.method == "GET":
                r = self.session.get(self.url, timeout=10)
            elif self.method in ("POST", "PUT", "PATCH"):
                r = self.session.request(self.method, self.url, json=self.body, timeout=10)
            return {"status": r.status_code, "length": len(r.content), "body": r.text[:500]}
        except Exception as e:
            return {"status": 0, "length": 0, "body": ""}
    
    def test_param(self, param_name):
        """Test a single parameter"""
        try:
            if self.method == "GET":
                separator = "&" if "?" in self.url else "?"
                test_url = f"{self.url}{separator}{param_name}=1"
                r = self.session.get(test_url, timeout=10)
            elif self.method == "POST":
                test_body = {**self.body, param_name: "test"}
                r = self.session.post(self.url, json=test_body, timeout=10)
            elif self.method == "PUT":
                test_body = {**self.body, param_name: "test"}
                r = self.session.put(self.url, json=test_body, timeout=10)
            elif self.method == "PATCH":
                test_body = {**self.body, param_name: "test"}
                r = self.session.patch(self.url, json=test_body, timeout=10)
            
            # Check for differences
            status_diff = r.status_code != self.baseline["status"]
            length_diff = abs(len(r.content) - self.baseline["length"]) > 50
            content_diff = r.text[:500] != self.baseline["body"]
            
            has_error = any(w in r.text.lower() for w in [
                "error", "exception", "warning", "fatal", "stack", "trace",
                "sql", "syntax", "unexpected", "invalid"
            ])
            
            has_debug = any(w in r.text.lower() for w in [
                "debug", "admin", "secret", "password", "token", "api_key",
                "config", "environment", "env", "internal"
            ])
            
            if status_diff or length_diff or has_error or has_debug:
                result = {
                    "param": param_name,
                    "method": self.method,
                    "status": r.status_code,
                    "length": len(r.content),
                    "baseline_length": self.baseline["length"],
                    "has_error": has_error,
                    "has_debug": has_debug,
                    "status_diff": status_diff,
                    "length_diff": length_diff,
                }
                self.findings.append(result)
                
                color = Fore.RED if has_error else Fore.YELLOW if has_debug else Fore.GREEN
                icon = "ERR" if has_error else "DBG" if has_debug else "DIF"
                print(f"{color}[{icon}] {param_name} -> HTTP {r.status_code} "
                      f"({self.baseline['length']} -> {len(r.content)}B){Style.RESET_ALL}")
                
        except Exception as e:
            pass
    
    def run(self, wordlist):
        print(f"\n{'='*60}")
        print(f"Parameter Scanning: {self.url}")
        print(f"Method: {self.method}")
        print(f"Baseline: HTTP {self.baseline['status']} ({self.baseline['length']}B)")
        print(f"Testing {len(wordlist)} parameters")
        print(f"{'='*60}\n")
        
        with concurrent.futures.ThreadPoolExecutor(max_workers=self.threads) as executor:
            executor.map(self.test_param, wordlist)
        
        self._report()
        return self.findings
    
    def _report(self):
        print(f"\n{'='*60}")
        print(f"Scan Complete: {len(self.findings)} interesting parameters found")
        print(f"{'='*60}\n")
        
        if self.findings:
            for f in self.findings[:30]:
                tags = []
                if f["has_error"]: tags.append("ERRORS")
                if f["has_debug"]: tags.append("DEBUG_INFO")
                if f["status_diff"]: tags.append(f"STATUS ({f['baseline_length']}->{f['status']})")
                if f["length_diff"]: tags.append(f"LENGTH ({f['baseline_length']}->{f['length']}B)")
                print(f"  {f['param']}: {', '.join(tags)}")
        
        with open("param_findings.json", "w") as f:
            json.dump(self.findings, f, indent=2)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Parameter Discovery Scanner")
    parser.add_argument("url", help="Target URL")
    parser.add_argument("-m", "--method", default="GET", choices=["GET", "POST", "PUT", "PATCH"])
    parser.add_argument("-w", "--wordlist", default="C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/params-all.txt")
    parser.add_argument("-t", "--threads", type=int, default=20)
    parser.add_argument("-H", "--header", action="append")
    parser.add_argument("-d", "--data", default="{}", help="JSON body for POST requests")
    args = parser.parse_args()
    
    headers = {}
    if args.header:
        for h in args.header:
            k, v = h.split(": ", 1)
            headers[k] = v
    
    body = json.loads(args.data) if args.data else {}
    
    with open(args.wordlist) as f:
        wordlist = [l.strip() for l in f if l.strip() and not l.startswith("#")]
    
    scanner = ParameterScanner(args.url, args.method, headers, body, args.threads)
    scanner.run(wordlist)
```

## 8. Header-Based Parameter Fuzzing

```bash
# Fuzz custom headers that might enable hidden functionality
HEADERS=(
  "X-Admin: true"
  "X-Internal: true"
  "X-Debug: true"
  "X-Test: true"
  "X-Override: true"
  "X-Forwarded-For: 127.0.0.1"
  "X-Real-IP: 127.0.0.1"
  "X-Forwarded-Host: internal.local"
  "X-Proxy-User: admin"
  "X-Auth-Token: admin"
  "X-Custom-Auth: admin"
  "X-Role: admin"
  "X-User-Role: admin"
  "X-Access-Level: admin"
  "X-Permission: admin"
  "X-Internal-Secret: true"
  "X-API-Version: internal"
  "X-Bypass-Auth: true"
  "X-No-Auth: true"
  "X-Skip-Validation: true"
  "X-Dev-Mode: true"
)

for header in "${HEADERS[@]}"; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/admin/users" \
    -H "Authorization: Bearer $TOKEN" \
    -H "$header")
  
  BASELINE=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/admin/users" \
    -H "Authorization: Bearer $TOKEN")
  
  if [ "$STATUS" != "$BASELINE" ]; then
    echo "[!] Header changes response: $header -> HTTP $STATUS (baseline $BASELINE)"
  fi
done
```

## 9. Response Parameter Discovery

```bash
# Some APIs reveal hidden parameters in error messages
curl -s "https://TARGET.COM/api/users?invalidParam=1" | \
  grep -oP '["'"'"']\w+["'"'"']\s*[:=]' | tr -d '": ' | sort -u > tmp/discovered-params.txt

# Check error responses for parameter hints
curl -s -X POST "https://TARGET.COM/api/endpoint" \
  -H "Content-Type: application/json" \
  -d '{}' | grep -oiP '(required|missing|unknown|expected|parameter|field|property).{0,50}' | \
  grep -oP '["'"'"']\w+["'"'"']' | tr -d '"' | sort -u > tmp/required-params.txt

# Check verbose errors for available params
curl -s -X POST "https://TARGET.COM/api/endpoint" \
  -H "Content-Type: application/json" \
  -d '{"test": true}' -v 2>&1
```

## 10. API Documentation Parameter Mining

```bash
# Extract all parameters from OpenAPI/Swagger docs
if [ -f tmp/swagger-found.txt ]; then
  for swagger_url in $(cat tmp/swagger-found.txt); do
    curl -s "$swagger_url" | jq -r '
      .paths[]? // {} | to_entries[]? | .value.parameters[]?.name,
      .value.requestBody.content["application/json"]?.schema.properties | keys[]
    ' 2>/dev/null | sort -u >> tmp/api-spec-params.txt
  done
fi

# Extract from GraphQL schema
curl -s "https://TARGET.COM/graphql" \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"query":"{__schema{types{name fields{name args{name}}}}}"}' | \
  jq -r '.data.__schema.types[].fields[].args[].name' 2>/dev/null | \
  sort -u > tmp/graphql-params.txt
```

## 11. Parameter Value Fuzzing

```bash
# For discovered parameters, fuzz different value types
# Boolean variations
for val in true false 1 0 yes no on off enabled disabled enable disable y n; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/endpoint?debug=$val")
  LEN=$(curl -s "https://TARGET.COM/api/endpoint?debug=$val" | wc -c)
  echo "debug=$val -> HTTP $STATUS ($LEN bytes)"
done

# Numeric variations
for val in 0 1 100 -1 -100 999999 9999999999 0.5 1.5; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/endpoint?limit=$val")
  echo "limit=$val -> HTTP $STATUS"
done

# String variations
for val in null undefined none empty admin all any first last; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/api/endpoint?filter=$val")
  echo "filter=$val -> HTTP $STATUS"
done
```

## 12. Cookie / Session Parameter Fuzzing

```bash
# Fuzz cookie parameters
for cookie in "admin=true" "role=admin" "user_type=admin" "access_level=admin" \
              "debug=true" "bypass=true" "is_admin=true" "verified=true"; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://TARGET.COM/admin" \
    -H "Cookie: session=$SESSION; $cookie")
  echo "$cookie -> HTTP $STATUS"
done
```

## 13. Real-World Hidden Parameter Discoveries

| Parameter | Found On | Impact | Bounty |
|-----------|---------|--------|--------|
| `?debug=1` | Facebook API | Revealed internal stack traces, DB queries | $5,000 |
| `?__internal=1` | Uber API | Admin functionality exposed | $3,000 |
| `?bypass=true` | Shopify admin | Bypassed authorization checks | $10,000+ |
| `?is_admin=1` | WordPress API | Privilege escalation | $500+ |
| `?show_all=true` | Twitter API | Exposed hidden user data | $2,000+ |
| `?env=true` | GitLab API | Environment variables (secrets) | $20,000 |
| `?test=1` | PayPal API | Sandbox mode with real data | $5,000+ |
| `?include=all` | Slack API | Revealed private channel data | $3,000+ |
| `?su=1` | Discord API | Superuser escalation | $5,000+ |
| `?source=true` | HackerOne API | Returned source code snippets | $2,000+ |

## 14. Checklist

- [ ] Run Arjun on every discovered endpoint
- [ ] Fuzz GET parameters on every endpoint
- [ ] Fuzz POST body parameters (JSON + form)
- [ ] Fuzz header parameters (X-* headers)
- [ ] Fuzz cookie parameters
- [ ] Test parameter pollution (duplicates, arrays)
- [ ] Test JSON prototype pollution
- [ ] Check error messages for parameter hints
- [ ] Extract parameters from JS source code
- [ ] Extract parameters from API documentation
- [ ] Extract parameters from GraphQL schema
- [ ] Test each discovered parameter with multiple values (boolean, numeric, string, null)
- [ ] Test for hidden debug parameters
- [ ] Test for admin override parameters
- [ ] Test for feature flag parameters
- [ ] Queue discoveries to Burp sitemap
