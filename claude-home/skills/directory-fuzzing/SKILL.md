---
name: directory-fuzzing
description: Hidden path discovery â€” admin panels, API routes, backup files, source code leaks, config exposure, sensitive files. ffuf/gobuster optimized configs for every scenario.
---

# Directory Fuzzing â€” Hidden Path Discovery

## 1. Philosophy

Directory fuzzing is the highest-ROI single activity in bug bounty. A single discovered endpoint like `/admin`, `/api/v1/internal`, `.git/config`, `/backup`, or `/vendor/phpunit/` can lead directly to a Critical finding. Every engagement starts with aggressive path discovery.

**The rule: fuzz everything, on every host, with every wordlist, at every depth.**

## 2. Wordlists

### 2.1 Built-in Wordlists

```bash
# Project-specific wordlists (already on VM)
ls C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/*.txt
# C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt
# C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/files.txt
# C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/api-endpoints.txt
# C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/parameters.txt
```

### 2.2 SecLists Integration

```bash
# Download SecLists if not present
if [ ! -d C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/SecLists ]; then
  cd C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists
  wget -qO- https://github.com/danielmiessler/SecLists/archive/master.tar.gz | tar xz
  mv SecLists-master SecLists
fi

# Best SecLists wordlists for directory fuzzing
SECLISTS=C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/SecLists

# Directory discovery
$SECLISTS/Discovery/Web-Content/common.txt                  # 4,700 entries â€” start here
$SECLISTS/Discovery/Web-Content/directory-list-2.3-medium.txt  # 220,000 entries (slow, thorough)
$SECLISTS/Discovery/Web-Content/raft-large-directories.txt     # 60,000 entries
$SECLISTS/Discovery/Web-Content/raft-large-files.txt           # 40,000 files

# Admin panel discovery
$SECLISTS/Discovery/Web-Content/CMS/admin-panels.txt          # 500 admin paths
$SECLISTS/Discovery/Web-Content/CMS/wordpress.txt              # WordPress-specific

# API discovery
$SECLISTS/Discovery/Web-Content/api/api-endpoints.txt          # 1,500 API endpoints
$SECLISTS/Discovery/Web-Content/api/graphql.txt                # GraphQL paths

# Technology-specific
$SECLISTS/Discovery/Web-Content/spring-boot.txt                # Spring Boot actuators
$SECLISTS/Discovery/Web-Content/swagger.txt                    # Swagger/OpenAPI paths
$SECLISTS/Discovery/Web-Content/IIS-webserver.txt              # IIS-specific
$SECLISTS/Discovery/Web-Content/Apache.txt                     # Apache-specific
$SECLISTS/Discovery/Web-Content/nginx.txt                      # Nginx-specific
$SECLISTS/Discovery/Web-Content/tomcat.txt                     # Tomcat-specific
$SECLISTS/Discovery/Web-Content/jboss.txt                      # JBoss-specific
```

### 2.3 Custom Wordlist Creation

```bash
# Generate from JS endpoints discovered during recon
cat tmp/js-endpoints.txt | grep -oP '/[a-zA-Z0-9_/.-]+' | sort -u > tmp/js-paths.txt

# Generate from Wayback/GAU URLs
cat tmp/wayback.txt tmp/gau.txt | grep -oP 'https?://[^/]+(\K/[^?#]*)' | \
  grep -vE '\.(js|css|png|jpg|jpeg|gif|svg|woff|woff2|ttf|eot)$' | \
  sort -u > tmp/known-paths.txt

# Merge with wordlist
cat tmp/known-paths.txt C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt | sort -u > tmp/merged-dirs.txt

# Generate path depth variants
cat tmp/known-paths.txt | while read path; do
  echo "$path"
  dirname "$path" 2>/dev/null
  dirname $(dirname "$path") 2>/dev/null
done | sort -u > tmp/path-variants.txt
```

## 3. ffuf â€” Optimal Configurations

### 3.1 Basic Discovery â€” Start Here

```bash
# Fast first-pass: common directories
ffuf -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/SecLists/Discovery/Web-Content/common.txt \
  -t 50 -rate 100 \
  -mc 200,301,302,307,401,403,405,500,502 \
  -ac  # Auto-calibrate filter
  -o tmp/dirs-common.json -of json

# Fast with extensions
ffuf -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -e .php,.asp,.aspx,.jsp,.js,.json,.xml,.txt,.html,.bak,.old,.zip,.tar.gz,.sql,.env \
  -t 40 -rate 80 \
  -mc 200,301,302,307,401,403,500 \
  -ac \
  -o tmp/dirs-extensions.json -of json
```

### 3.2 Admin Panel Discovery

```bash
# Dedicated admin panel fuzz
ffuf -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/SecLists/Discovery/Web-Content/CMS/admin-panels.txt \
  -t 30 -rate 50 \
  -mc 200,301,302,401,403 \
  -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)" \
  -o tmp/admin-panels.json -of json

# Recursive admin discovery
ffuf -u https://TARGET.COM/FUZZ \
  -w <(cat C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/SecLists/Discovery/Web-Content/raft-large-directories.txt | sort -u) \
  -t 20 -rate 30 \
  -mc 200,301,302,401,403 \
  -recursion -recursion-depth 3 \
  -H "User-Agent: Mozilla/5.0" \
  -o tmp/admin-recursive.json -of json
```

### 3.3 API Endpoint Discovery

```bash
# REST API fuzzing
ffuf -u https://TARGET.COM/api/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/api-endpoints.txt \
  -t 30 -rate 50 \
  -mc 200,201,401,403,404,405,500 \
  -o tmp/api-endpoints.json -of json

# API version discovery
ffuf -u https://TARGET.COM/FUZZ \
  -w <(echo -e "v1\nv2\nv3\nv1.0\nv2.0\nv3.0\napi\napi/v1\napi/v2\napi/v3\nrest\nrest/v1\ngraphql\ngrpc\n") \
  -t 10 -rate 30 \
  -mc 200,201,301,302,401,403 \
  -o tmp/api-versions.json -of json

# RESTful resource ID fuzzing
ffuf -u https://TARGET.COM/api/v1/FUZZ \
  -w <(echo -e "users\nuser\nadmins\nadmin\naccounts\naccount\nprofiles\nprofile\norders\norder\npayments\npayment\ninvoices\ninvoice\nsubscriptions\nsubscription\norganizations\norganization\ntenants\ntenant\nproducts\nproduct\nitems\nitem\ntransactions\ntransaction\ntokens\ntoken\nsessions\nsession\nevents\nevent\nfiles\nfile\nuploads\nupload\nimages\nimage\ndocuments\ndocument\nnotifications\nnotification\nmessages\nmessage\nreports\nreport\nanalytics\nanalytics\nexports\nexport\nimports\nimport\nlogs\nlog\nsettings\nsetting\nconfig\nconfiguration\nwebhooks\nwebhook\nhooks\nhook\n") \
  -t 30 -rate 50 \
  -mc 200,201,401,403 \
  -o tmp/api-resources.json -of json

# Deep API resource fuzzing
ffuf -u https://TARGET.COM/FUZZ \
  -w <(for prefix in api v1 v2 rest graphql admin internal private public; do
    for resource in users user admins admin accounts account profiles profile settings setting config configuration logs log events event sessions session tokens token; do
      echo "$prefix/$resource"
      echo "$prefix/$resource/"
      echo "$prefix/$resource/list"
      echo "$prefix/$resource/create"
      echo "$prefix/$resource/update"
      echo "$prefix/$resource/delete"
      echo "$prefix/$resource/get"
      echo "$prefix/$resource/search"
      echo "$prefix/$resource/export"
      echo "$prefix/$resource/import"
    done
  done) \
  -t 20 -rate 30 \
  -mc 200,201,401,403,405 \
  -o tmp/deep-api.json -of json
```

### 3.4 Technology-Specific Discovery

```bash
# Spring Boot
ffuf -u https://TARGET.COM/FUZZ \
  -w <(echo -e "actuator\nactuator/\nactuator/health\nactuator/info\nactuator/env\nactuator/beans\nactuator/mappings\nactuator/heapdump\nactuator/loggers\nactuator/metrics\nactuator/prometheus\nactuator/threaddump\nactuator/scheduledtasks\nactuator/httptrace\nactuator/auditevents\nactuator/conditions\nactuator/configprops\nactuator/shutdown\n") \
  -t 10 -rate 20 \
  -mc 200,401,403 \
  -o tmp/spring-boot.json -of json

# Swagger / OpenAPI
ffuf -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/SecLists/Discovery/Web-Content/swagger.txt \
  -t 10 -rate 20 \
  -mc 200,301,302 \
  -o tmp/swagger.json -of json

# PHPMyAdmin / Adminer
ffuf -u https://TARGET.COM/FUZZ \
  -w <(echo -e "phpmyadmin\nphpMyAdmin\npma\nadminer\nadminer.php\nmysql\nphpmyadmin2\nphpmyadmin4\nphpPgAdmin\n") \
  -t 10 -rate 20 \
  -mc 200,301,302 \
  -o tmp/db-admin.json -of json

# GraphQL
ffuf -u https://TARGET.COM/FUZZ \
  -w <(echo -e "graphql\ngraphiql\nv1/graphql\nv2/graphql\nv3/graphql\napi/graphql\napi/v1/graphql\napi/v2/graphql\ngql\ngraph\ngraphql/console\nquery\n") \
  -t 10 -rate 20 \
  -mc 200,301,302,401,403 \
  -o tmp/graphql.json -of json
```

### 3.5 Source Code & Sensitive File Discovery

```bash
# Git exposure
ffuf -u https://TARGET.COM/.git/FUZZ \
  -w <(echo -e "HEAD\nconfig\nindex\nrefs/heads/master\nrefs/heads/main\nlogs/HEAD\nobjects\nFETCH_HEAD\nORIG_HEAD\npacked-refs\ninfo/refs\ninfo/exclude\ndescription\nCOMMIT_EDITMSG\n")) \
  -t 10 -rate 20 \
  -mc 200,301,302 \
  -o tmp/git-exposure.json -of json

# Environment/config files
ffuf -u https://TARGET.COM/FUZZ \
  -w <(echo -e ".env\n.env.production\n.env.local\n.env.dev\n.env.staging\n.env.test\n.env.example\n.env.sample\n.env.backup\n.env.old\n.env.txt\n.env.bak\nconfig\nconfig.php\nconfig.json\nconfig.xml\nconfig.yml\nconfig.yaml\nconfig.js\nconfig.env\nconfiguration.php\nsettings.php\nsettings.json\napp.config\napplication.config\ndatabase.yml\ndatabase.json\ncredentials.json\ncredentials.yml\nsecret\nsecrets.yml\nsecret.json\nkey.json\nkeys.json\nservice-account.json\nfirebase.json\n") \
  -t 10 -rate 20 \
  -mc 200,301,302 \
  -o tmp/config-files.json -of json

# Backup files
ffuf -u https://TARGET.COM/FUZZ \
  -w <(cat C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
    | while read path; do
        echo "$path.bak"
        echo "$path.old"
        echo "$path.backup"
        echo "$path~"
        echo "$path.txt"
        echo "$path.tar"
        echo "$path.tar.gz"
        echo "$path.zip"
        echo "$path.gz"
        echo "$path.sql"
      done) \
  -t 20 -rate 30 \
  -mc 200,301,302 \
  -o tmp/backup-files.json -of json

# Source maps (JavaScript source code leaks)
ffuf -u https://TARGET.COM/assets/FUZZ \
  -w <(echo -e "app.js.map\nmain.js.map\nbundle.js.map\nvendor.js.map\nindex.js.map\nscript.js.map\napplication.js.map\nwebpack.js.map\nruntime.js.map\nchunk.js.map\n") \
  -t 10 -rate 20 \
  -mc 200 \
  -o tmp/source-maps.json -of json
```

### 3.6 Method Fuzzing â€” Find Hidden Endpoints via Method

```bash
# Test all HTTP methods on discovered endpoints
for method in GET POST PUT PATCH DELETE OPTIONS HEAD TRACE CONNECT; do
  ffuf -u https://TARGET.COM/FUZZ \
    -w tmp/discovered-paths.txt \
    -X $method \
    -t 20 -rate 30 \
    -mc 200,201,202,204,301,302,307,401,403,405,500 \
    -o tmp/method-$method.json -of json
done

# HEAD/OPTIONS often return different results
curl -X OPTIONS -sI https://TARGET.COM/api/admin/users
curl -X TRACE -s https://TARGET.COM/api/admin/users
```

### 3.7 Virtual Host / Subdomain Fuzzing

```bash
# VHost discovery via Host header
ffuf -u https://TARGET.COM \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/subdomains.txt \
  -H "Host: FUZZ.TARGET.COM" \
  -t 30 -rate 50 \
  -mc 200,301,302,401,403 \
  -fs 1234  # Filter by default response size
  -o tmp/vhosts.json -of json

# Subdomain fuzzing via DNS
ffuf -u https://FUZZ.TARGET.COM \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/subdomains.txt \
  -t 50 -rate 100 \
  -mc 200,301,302,401,403,500,502,503 \
  -o tmp/subdomains-dns.json -of json
```

## 4. Recursive Fuzzing Strategy

```bash
# Phase 1: High-level directory discovery (fast)
ffuf -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -t 50 -rate 100 \
  -mc 200,301,302,401,403 \
  -o tmp/phase1.json -of json

# Extract discovered directories
cat tmp/phase1.json | jq -r '.results[] | select(.status == 301 or .status == 302) | .input' \
  > tmp/discovered-dirs.txt

# Phase 2: Recurse into discovered directories
cat tmp/discovered-dirs.txt | while read dir; do
  ffuf -u "https://TARGET.COM/$dir/FUZZ" \
    -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
    -t 30 -rate 50 \
    -mc 200,301,302,401,403 \
    -o "tmp/phase2-$dir.json" -of json
done

# Phase 3: Deep recursion on admin/internal directories
cat tmp/discovered-dirs.txt | grep -iE 'admin|internal|private|secure|restricted|dashboard|console|manage' | \
while read dir; do
  for depth in $(seq 1 5); do
    ffuf -u "https://TARGET.COM/$dir$(printf '/FUZZ%.0s' $(seq 1 $depth))" \
      -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/SecLists/Discovery/Web-Content/raft-large-directories.txt \
      -t 15 -rate 20 \
      -mc 200,301,302,401,403 \
      -o "tmp/deep-$dir-depth-$depth.json" -of json
  done
done
```

## 5. Response Analysis â€” Identifying Hidden Gems

### 5.1 Status Code Significance

| Status | Meaning | Action |
|--------|---------|--------|
| 200 OK | Found and accessible | Test immediately for vulns |
| 201 Created | Created (write endpoint) | Test POST/PUT with data |
| 204 No Content | Found, empty body | Check if data needs params |
| 301/302 Redirect | Exists, redirecting | Follow redirect chain |
| 307/308 Redirect | Exists, preserves method | Follow with same method |
| 401 Unauthorized | Exists, needs auth | Test auth bypass |
| 403 Forbidden | Exists, access denied | Test access control bypass |
| 405 Method Not Allowed | Exists, wrong method | Try other HTTP methods |
| 500 Internal Error | Server error triggered | Investigate for injection |
| 502/503 Bad Gateway | Backend error | Investigate for SSRF |
| 429 Too Many Requests | Rate limited | Rotate IPs, slow down |

### 5.2 Response Size Analysis

```bash
# Find endpoints with different response sizes (potential hidden functionality)
cat tmp/phase1.json | jq -r '.results[] | "\(.status) \(.length) \(.input)"' | \
  sort -k2 -n | uniq -f1 --skip-fields=1 | head -50

# Cluster by similar sizes to find outliers
cat tmp/phase1.json | jq -r '.results[] | "\(.length) \(.status) \(.input)"' | \
  sort -n | awk 'NR==1{prev=$1} $1!=prev{print; prev=$1}'

# Filter out noise (same response length as 404)
DEFAULT_LEN=$(curl -s -o /dev/null -w "%{size_download}" "https://TARGET.COM/nonexistentpath123456")
cat tmp/phase1.json | jq -r ".results[] | select(.length != $DEFAULT_LEN) | \"\(.status) \(.length) \(.input)\""
```

### 5.3 Content Diffing

```bash
# Compare responses from different paths to identify hidden differences
# First, get the default 404 page
curl -s "https://TARGET.COM/THISPATHSHOULDNOTEXIST123" > tmp/default-404.html

# Diff against discovered paths
for path in $(cat tmp/discovered-dirs.txt); do
  curl -s "https://TARGET.COM/$path" > "tmp/response-$path.html"
  DIFF=$(diff tmp/default-404.html "tmp/response-$path.html" | wc -l)
  if [ "$DIFF" -gt 5 ]; then
    echo "[!] Path differs: $path ($DIFF lines different)"
  fi
done

# Check for hidden parameters that change response
for param in debug test source admin internal show_all full_details; do
  RESP=$(curl -s "https://TARGET.COM/admin?$param=1" -H "Authorization: Bearer $TOKEN")
  echo "$param -> $(echo "$RESP" | head -c 100)"
done
```

## 6. Wordlist Generation from Target

### 6.1 Extract Paths from JS Files

```bash
# Extract all paths from discovered JS files
cat tmp/js-files.txt | while read url; do
  curl -sk "$url" | grep -oP '(?:["'"'"'])(/[a-zA-Z0-9_/.-]+)(?:["'"'"'])' | \
    tr -d '"'"'"' | sort -u
done | sort -u > tmp/js-paths-target.txt

# Extract API endpoints from JS
cat tmp/js-files.txt | while read url; do
  curl -sk "$url" | grep -oP '(?:["'"'"'])(/api/[a-zA-Z0-9_/.-]+)(?:["'"'"'])' | \
    tr -d '"'"'"' | sort -u
done | sort -u > tmp/js-api-paths.txt
```

### 6.2 Extract Paths from HTML

```bash
# Extract from all discovered pages
cat tmp/live-hosts.txt | while read url; do
  curl -sk "$url" | grep -oP 'href=["'"'"']?\K/[a-zA-Z0-9_/.-]+(?=["'"'"']|>)' | sort -u
  curl -sk "$url" | grep -oP 'action=["'"'"']?\K/[a-zA-Z0-9_/.-]+(?=["'"'"']|>)' | sort -u
  curl -sk "$url" | grep -oP 'src=["'"'"']?\K/[a-zA-Z0-9_/.-]+(?=["'"'"']|>)' | sort -u
done | sort -u > tmp/html-paths.txt
```

### 6.3 Generate Fuzzing List from Sitemap / Robots

```bash
# Check robots.txt
curl -s "https://TARGET.COM/robots.txt" | grep -i 'Disallow:' | \
  awk '{print $2}' | sort -u > tmp/robots-paths.txt

# Check sitemap.xml
curl -s "https://TARGET.COM/sitemap.xml" | \
  grep -oP '<loc>\K[^<]+' | grep -oP 'https?://[^/]+\K/.*' | sort -u > tmp/sitemap-paths.txt

# Check sitemap index
curl -s "https://TARGET.COM/sitemap_index.xml" | \
  grep -oP '<loc>\K[^<]+' | while read submap; do
    curl -s "$submap" | grep -oP '<loc>\K[^<]+' | grep -oP 'https?://[^/]+\K/.*'
done | sort -u > tmp/sitemap-paths.txt

# Check security.txt
curl -s "https://TARGET.COM/.well-known/security.txt"
curl -s "https://TARGET.COM/security.txt"
```

## 7. WAF / Rate Limit Evasion for Fuzzing

```bash
# Rotate User-Agents
UA_LIST=(
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"
  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
  "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
)

ffuf -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -t 20 -rate 30 \
  -H "User-Agent: ${UA_LIST[$RANDOM % ${#UA_LIST[@]}]}" \
  -mc 200,301,302,401,403 \
  -o tmp/dirs-rotating-ua.json -of json

# Add random delays
for path in $(cat C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt | head -100); do
  curl -s -o /dev/null -w "%{http_code} $path\n" "https://TARGET.COM/$path" \
    -H "User-Agent: ${UA_LIST[$RANDOM % ${#UA_LIST[@]}]}"
  sleep 0.$((RANDOM % 5))  # 0-0.5s random delay
done

# Use Proxy Rotation (if available)
ffuf -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -t 10 -rate 15 \
  -x http://127.0.0.1:8080  # Burp proxy for logging, not rotation
  -mc 200,301,302,401,403 \
  -o tmp/dirs-via-burp.json -of json

# IP rotation via proxy pool (if available)
for ip in $(cat proxy_pool.txt); do
  ffuf -u https://TARGET.COM/FUZZ \
    -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
    -t 5 -rate 10 \
    -x "http://$ip" \
    -mc 200,301,302,401,403 \
    -o "tmp/dirs-ip-$ip.json" -of json
done
```

## 8. Python Automated Fuzzer

```python
#!/usr/bin/env python3
"""
Automated directory fuzzer with recursive discovery, status analysis, and report generation.
"""
import requests
import concurrent.futures
import argparse
import json
import sys
import time
from urllib.parse import urljoin
from colorama import init, Fore, Style

init()

class DirectoryFuzzer:
    SIGNIFICANT_STATUSES = {200, 201, 202, 204, 301, 302, 307, 401, 403, 405, 500, 502, 503}
    
    def __init__(self, base_url, wordlist, extensions=None, headers=None, threads=20, delay=0):
        self.base_url = base_url.rstrip('/')
        self.wordlist = wordlist
        self.extensions = extensions or []
        self.headers = headers or {}
        self.threads = threads
        self.delay = delay
        self.session = requests.Session()
        self.session.headers.update(self.headers)
        self.results = []
        self.default_404_size = None
        self._calibrate_404()
    
    def _calibrate_404(self):
        """Get the default 404 response size for filtering"""
        try:
            r = self.session.get(f"{self.base_url}/__calibrate_404_{int(time.time())}__", timeout=10)
            self.default_404_size = len(r.content)
            print(f"[*] Calibrated 404 size: {self.default_404_size} bytes")
        except:
            self.default_404_size = 0
    
    def fuzz_path(self, path):
        """Test a single path"""
        if self.delay:
            time.sleep(self.delay)
        
        urls_to_test = [f"{self.base_url}/{path.lstrip('/')}"]
        for ext in self.extensions:
            urls_to_test.append(f"{self.base_url}/{path.lstrip('/')}.{ext.lstrip('.')}")
        
        for url in urls_to_test:
            try:
                r = self.session.get(url, timeout=15, allow_redirects=False)
                
                if r.status_code in self.SIGNIFICANT_STATUSES:
                    content_len = len(r.content)
                    is_diff = abs(content_len - (self.default_404_size or 0)) > 50
                    
                    result = {
                        "url": url,
                        "path": path,
                        "status": r.status_code,
                        "length": content_len,
                        "is_diff": is_diff,
                        "content_type": r.headers.get("Content-Type", ""),
                        "location": r.headers.get("Location", ""),
                        "server": r.headers.get("Server", ""),
                    }
                    self.results.append(result)
                    
                    color = Fore.GREEN if r.status_code == 200 else (
                        Fore.YELLOW if r.status_code in (301, 302, 401, 403) else Fore.RED
                    )
                    icon = "OK" if r.status_code == 200 else (
                        "RD" if r.status_code in (301, 302) else (
                            "AU" if r.status_code == 401 else (
                                "FB" if r.status_code == 403 else "ER"
                            )
                        )
                    )
                    print(f"{color}[{icon}] {r.status_code} {content_len:>7}B {url}{Style.RESET_ALL}")
                    
            except requests.exceptions.ConnectionError:
                pass
            except Exception as e:
                print(f"{Fore.RED}[ERR] {path}: {e}{Style.RESET_ALL}")
    
    def run(self):
        """Run fuzzing against the wordlist"""
        print(f"\n{'='*60}")
        print(f"Directory Fuzzing: {self.base_url}")
        print(f"Wordlist: {len(self.wordlist)} entries")
        print(f"Extensions: {self.extensions or 'none'}")
        print(f"{'='*60}\n")
        
        with concurrent.futures.ThreadPoolExecutor(max_workers=self.threads) as executor:
            list(executor.map(self.fuzz_path, self.wordlist))
        
        self._report()
        return self.results
    
    def _report(self):
        """Generate fuzzing report"""
        print(f"\n{'='*60}")
        print(f"Fuzzing Complete: {len(self.results)} significant responses")
        print(f"{'='*60}\n")
        
        for status_code in sorted(set(r["status"] for r in self.results)):
            count = len([r for r in self.results if r["status"] == status_code])
            print(f"  HTTP {status_code}: {count} responses")
        
        # Group by status for analysis
        for status in [200, 201, 301, 302, 401, 403, 405, 500]:
            matches = [r for r in self.results if r["status"] == status]
            if matches:
                print(f"\n--- HTTP {status} ({len(matches)} matches) ---")
                for r in matches[:20]:
                    print(f"  {r['url']}")
        
        # Save results
        output_file = f"fuzz_results_{int(time.time())}.json"
        with open(output_file, "w") as f:
            json.dump(self.results, f, indent=2)
        print(f"\n[+] Results saved to: {output_file}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Directory Fuzzer")
    parser.add_argument("url", help="Target URL")
    parser.add_argument("-w", "--wordlist", required=True, help="Wordlist file")
    parser.add_argument("-e", "--extensions", nargs="+", help="File extensions to try")
    parser.add_argument("-t", "--threads", type=int, default=20, help="Thread count")
    parser.add_argument("--delay", type=float, default=0, help="Delay between requests (seconds)")
    parser.add_argument("-H", "--header", action="append", help="Custom header (format: 'Name: Value')")
    args = parser.parse_args()
    
    headers = {}
    if args.header:
        for h in args.header:
            key, val = h.split(": ", 1)
            headers[key] = val
    
    with open(args.wordlist) as f:
        wordlist = [line.strip() for line in f if line.strip() and not line.startswith("#")]
    
    fuzzer = DirectoryFuzzer(args.url, wordlist, args.extensions, headers, args.threads, args.delay)
    fuzzer.run()
```

## 9. Directory Fuzzing by Context

### 9.1 E-commerce Targets

```bash
ffuf -u https://TARGET.COM/FUZZ \
  -w <(echo -e "cart\ncheckout\norders\norder\nproducts\nproduct\ncategories\ncategory\nsearch\naccount\nlogin\nsignup\nregister\nauth\npayment\nbilling\nshipping\nreturns\ncoupons\ndiscounts\nreviews\nratings\nadmin\nadmin/products\nadmin/orders\nadmin/customers\nadmin/coupons\napi\napi/products\napi/orders\napi/customers\napi/cart\napi/checkout\napi/payment\napi/shipping\n")) \
  -t 30 -rate 50 \
  -mc 200,301,302,401,403,405 \
  -o tmp/ecom-paths.json -of json
```

### 9.2 SaaS Targets

```bash
ffuf -u https://TARGET.COM/FUZZ \
  -w <(echo -e "dashboard\nprojects\nproject\nworkspace\nworkspaces\norganization\norganizations\nteam\nteams\nsettings\nbilling\nsubscription\nplans\npricing\naccount\nprofile\nnotifications\nintegrations\nwebhooks\napi-keys\ntokens\naudit-logs\nactivity\nmembers\ninvite\nimport\nexport\nadmin\nadmin/users\nadmin/organizations\nadmin/billing\nadmin/settings\nadmin/logs\nadmin/features\nadmin/plans\napi\napi/v1\napi/v2\ngraphql\n")) \
  -t 30 -rate 50 \
  -mc 200,301,302,401,403,405 \
  -o tmp/saas-paths.json -of json
```

### 9.3 Fintech Targets

```bash
ffuf -u https://TARGET.COM/FUZZ \
  -w <(echo -e "account\naccounts\nbalance\ntransactions\ntransfer\nwithdraw\ndeposit\npayment\npayments\ncards\ncard\nloans\nloan\ninvestments\nportfolio\nholdings\ntrades\ntrade\nexchange\nrates\nfees\nlimits\nkyc\nverification\ncompliance\naudit\naudit-logs\nreports\nstatements\ninvoices\nbill\nbills\nadmin\nadmin/users\nadmin/transactions\nadmin/limits\nadmin/compliance\nadmin/audit\nadmin/fees\nadmin/kyc\nadmin/reports\napi\napi/v1\napi/v2\ngraphql\ninternal\ninternal/transactions\ninternal/users\n")) \
  -t 20 -rate 30 \
  -mc 200,301,302,401,403,405 \
  -o tmp/fintech-paths.json -of json
```

## 10. Post-Fuzzing Actions

### 10.1 Immediate Testing on Discovered Endpoints

```bash
# For every 200/201 discovered, run immediate vulnerability probes
cat tmp/dirs-common.json | jq -r '.results[] | select(.status == 200 or .status == 201) | .url' | \
while read url; do
  echo "=== Testing: $url ==="
  
  # Method testing
  curl -s -o /dev/null -w "POST %{http_code}\n" -X POST "$url"
  curl -s -o /dev/null -w "PUT %{http_code}\n" -X PUT "$url"
  curl -s -o /dev/null -w "PATCH %{http_code}\n" -X PATCH "$url"
  curl -s -o /dev/null -w "DELETE %{http_code}\n" -X DELETE "$url"
  curl -s -o /dev/null -w "OPTIONS %{http_code}\n" -X OPTIONS "$url"
  
  # Content-type fuzzing
  curl -s -o /dev/null -w "JSON: %{http_code}\n" -X POST "$url" \
    -H "Content-Type: application/json"
  curl -s -o /dev/null -w "XML: %{http_code}\n" -X POST "$url" \
    -H "Content-Type: application/xml"
    
  # Auth bypass headers
  curl -s -o /dev/null -w "X-Admin: %{http_code}\n" "$url" \
    -H "X-Admin: true"
  curl -s -o /dev/null -w "X-Forwarded-For: %{http_code}\n" "$url" \
    -H "X-Forwarded-For: 127.0.0.1"
done
```

### 10.2 Queuing for Burp

```bash
# Every discovered endpoint goes to Burp sitemap
cat tmp/dirs-common.json | jq -r '.results[] | select(.status == 200 or .status == 201 or .status == 401 or .status == 403) | .url' | \
while read url; do
  oc-burp-queue "$PROJECT" "$url"
done
```

## 11. File Extension Mapping

| Extension | What it reveals | Priority |
|-----------|----------------|----------|
| `.php` | PHP application, potential RCE via file upload | P0 |
| `.asp` / `.aspx` | ASP.NET application | P0 |
| `.jsp` / `.do` | Java application | P0 |
| `.json` | API endpoint | P0 |
| `.xml` | API endpoint, XXE potential | P0 |
| `.action` | Struts/Spring MVC | P0 |
| `.wsdl` | SOAP web service | P0 |
| `.git/` | Full source code exposure | Critical |
| `.env` | Environment secrets | Critical |
| `.sql` | Database dump | Critical |
| `swagger.json` | API documentation | P0 |
| `openapi.json` | API documentation | P0 |
| `graphql` | GraphQL endpoint | P0 |
| `.js` | JavaScript source | P1 |
| `.js.map` | JavaScript source map | P1 |
| `.ts` | TypeScript source | P1 |
| `.bak` | Backup file | P1 |
| `.old` | Old version | P1 |
| `.tar.gz` | Archive | P1 |
| `.zip` | Archive | P1 |
| `robots.txt` | Hidden paths | P2 |
| `sitemap.xml` | Site structure | P2 |
| `.txt` | Notes/doc | P2 |
| `.md` | Documentation | P2 |
| `package.json` | Dependency info | P2 |
| `Dockerfile` | Container info | P2 |
| `docker-compose.yml` | Infrastructure | P2 |
| `Makefile` | Build info | P2 |
| `composer.json` | PHP dependencies | P2 |
| `Gemfile` | Ruby dependencies | P2 |
| `requirements.txt` | Python dependencies | P2 |
| `pom.xml` | Maven/Java dependencies | P2 |
| `build.gradle` | Gradle build | P2 |
| `webpack.config.js` | Build config | P2 |
| `.htaccess` | Apache config | P2 |
| `nginx.conf` | Nginx config | P2 |
| `web.config` | IIS config | P2 |

## 12. Response Content Analysis Script

```bash
#!/bin/bash
# Analyze all fuzzing results for interesting content
RESULTS_DIR="tmp"

echo "=== Interesting Content Found ==="
for f in "$RESULTS_DIR"/*.json; do
  cat "$f" | jq -r '.results[] | select(.status == 200) | .url' | while read url; do
    content=$(curl -sk "$url" 2>/dev/null)
    
    # Check for sensitive patterns
    echo "$content" | grep -qi "password\|secret\|key\|token\|auth\|admin\|root\|database\|sql\|config\|credential\|api_key" && \
      echo "[!] KEYWORDS: $url"
    
    echo "$content" | grep -qi "phpinfo\|system_root\|SERVER_ADDR\|_SERVER\|_ENV\|_GET\|_POST" && \
      echo "[!] PHPINFO: $url"
    
    echo "$content" | grep -qi "stack trace\|Exception\|SyntaxError\|ParseError\|Error:" | head -3 && \
      echo "[!] DEBUG: $url"
    
    len=$(echo "$content" | wc -c)
    echo "[$len bytes] $url"
  done
done
```

## 13. Speed vs. Coverage Decision Matrix

| Scenario | Wordlist | Threads | Rate | Time Estimate |
|----------|----------|---------|------|---------------|
| Quick first pass | common.txt (4.7K) | 50 | 100/s | ~1 min |
| Standard discovery | directories.txt (10K) | 40 | 80/s | ~2 min |
| Thorough discovery | raft-large (60K) | 30 | 50/s | ~20 min |
| Deep content audit | directory-list-2.3-medium (220K) | 20 | 30/s | ~2 hours |
| Admin panel specific | admin-panels.txt (500) | 20 | 30/s | ~30 sec |
| API endpoint discovery | api-endpoints.txt (1.5K) | 30 | 50/s | ~30 sec |
| Sensitive files | custom (100) | 10 | 20/s | ~10 sec |
| Technology-specific | custom per-stack | 10 | 15/s | ~1 min per tech |
| Recursive (3 levels) | directories.txt | 20 | 30/s | ~30 min+ |
| All extensions | directories.txt + 10 ext | 30 | 40/s | ~5 min |
