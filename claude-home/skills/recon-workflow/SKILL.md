---
name: recon-workflow
description: Complete reconnaissance workflow from subdomain enumeration to live host mapping. Use at the start of any new engagement, when @recon needs to run, or when fresh enumeration is needed on a discovered subdomain. Covers subfinder, amass, httpx, waybackurls, gau, JS analysis, and directory fuzzing. ALL HTTP tools route through Burp 8080 for sitemap capture.
---

# Recon Workflow

## CRITICAL â€” Burp Proxy (MANDATORY for all HTTP tools)

**Every HTTP request from hunting tools MUST route through Burp 8080.**

- **Tools with `-proxy`/`--proxy` flag**: ffuf, nuclei, httpx, sqlmap, curl â€” use the flag directly
- **Tools without proxy flag**: dalfox, gobuster, arjun, katana, hakrawler, gospider, subjs, LinkFinder, Corsy â€” run `source /usr/local/bin/oc-proxy-env` BEFORE the tool
- **TCP scanners (NO proxy needed)**: naabu, nmap, subfinder, dnsx, amass â€” these don't make HTTP requests to target

**Quick reference â€” how to set proxy:**
```bash
# Option 1: Tool-specific flag (preferred)
ffuf -x http://127.0.0.1:8080 -u ...
httpx -proxy http://127.0.0.1:8080 -u ...
nuclei -proxy http://127.0.0.1:8080 -u ...
sqlmap --proxy=http://127.0.0.1:8080 -u ...
curl --proxy http://127.0.0.1:8080 https://...

# Option 2: Env var for tools without flag
source /usr/local/bin/oc-proxy-env && dalfox pipe ...
source /usr/local/bin/oc-proxy-env && gobuster dir ...
source /usr/local/bin/oc-proxy-env && arjun -u ...
```

## Phase 1: Subdomain Enumeration (combine all sources)
```bash
# Primary sources (passive DNS â€” no proxy needed)
subfinder -d TARGET.COM -silent | tee tmp/subfinder.txt
amass enum -passive -d TARGET.COM -o tmp/amass.txt 2>/dev/null

# Certificate transparency (HTTP to crt.sh â€” safe, not target)
curl -s "https://crt.sh/?q=%.TARGET.COM&output=json" | jq -r '.[].name_value' | sort -u > tmp/crtsh.txt

# DNS brute force (no proxy â€” DNS resolution)
dnsx -d TARGET.COM -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/subdomains.txt -silent | tee tmp/dns-brute.txt

# Combine and deduplicate
cat tmp/subfinder.txt tmp/amass.txt tmp/crtsh.txt tmp/dns-brute.txt | sort -u > tmp/all-subdomains.txt
wc -l tmp/all-subdomains.txt
```

## Phase 2: Live Host Probing â€” AUTO-QUEUE TO BURP
```bash
# Probe for live HTTP/HTTPS â€” auto-queues all 2xx results to Burp sitemap
cat tmp/all-subdomains.txt | oc-burp-queue-httpx <project> -silent \
  -status-code -title -tech-detect -follow-redirects \
  -o tmp/live-hosts.txt

# Quick check for responding hosts â€” auto-queues to Burp
cat tmp/all-subdomains.txt | oc-burp-queue-httpx <project> -silent \
  -fc 404,403 -mc 200,301,302 \
  -o tmp/alive.txt

# Screenshot alive hosts (MUST use Burp proxy)
cat tmp/alive.txt | gowitness file -f - --timeout 15 -P tmp/screenshots/

# Also queue all alive URLs explicitly (belt + suspenders)
cat tmp/alive.txt | oc-burp-queue-urls <project>
```

## Phase 3: Port Scanning
```bash
# Quick top 1000 ports (TCP â€” no proxy needed)
naabu -list tmp/alive.txt -top-ports 1000 -o tmp/ports-quick.txt

# Full port scan (TCP â€” no proxy needed)
naabu -list tmp/alive.txt -p - -o tmp/ports-full.txt

# Service detection on discovered ports (TCP â€” no proxy needed)
nmap -sV -sC -iL tmp/alive.txt -oA tmp/nmap-services --top-ports 100
```

## Phase 4: Endpoint Discovery â€” AUTO-QUEUE TO BURP
```bash
# Wayback Machine (HTTP to archive.org â€” safe)
cat tmp/alive.txt | waybackurls | sort -u > tmp/wayback.txt

# GAU (GetAllURLs â€” HTTP to archive sources â€” safe)
cat tmp/alive.txt | gau --threads 10 | sort -u > tmp/gau.txt

# Combine
cat tmp/wayback.txt tmp/gau.txt | sort -u > tmp/all-urls.txt

# AUTO-QUEUE all discovered URLs to Burp sitemap
cat tmp/all-urls.txt | oc-burp-queue-urls <project>

# Filter for interesting endpoints
cat tmp/all-urls.txt | grep -iE '\.(php|asp|aspx|jsp|json|xml|api|admin|panel|login|upload|backup|config|\.env|\.git)' | sort -u > tmp/interesting-urls.txt
```

## Phase 5: JavaScript Analysis â€” AUTO-QUEUE TO BURP
```bash
# Find JS files
cat tmp/all-urls.txt | grep -i '\.js$' | sort -u > tmp/js-files.txt

# Get JS files (MUST use Burp proxy â€” fetches from target)
source /usr/local/bin/oc-proxy-env && cat tmp/js-files.txt | subjs | tee tmp/subjs.txt

# AUTO-QUEUE all discovered JS file URLs to Burp
cat tmp/js-files.txt | oc-burp-queue-urls <project>
cat tmp/subjs.txt | oc-burp-queue-urls <project>

# Extract endpoints from JS (MUST use Burp proxy)
for js in $(cat tmp/js-files.txt | head -50); do
  curl --proxy http://127.0.0.1:8080 -sk "$js" 2>/dev/null
done | grep -oP '(?:["'"'"']/api/|["'"'"']/v\d/|["'"'"']/admin/|["'"'"']/user/|["'"'"']/auth/)[^"'"'"'\s]*' | sort -u > tmp/js-endpoints.txt

# AUTO-QUEUE extracted JS endpoints to Burp
cat tmp/js-endpoints.txt | oc-burp-queue-urls <project>

# Find secrets in JS (MUST use Burp proxy)
for js in $(cat tmp/js-files.txt | head -50); do
  curl --proxy http://127.0.0.1:8080 -sk "$js" 2>/dev/null
done | grep -oiE '(api[_-]?key|secret|token|password|aws[_-]?access|private[_-]?key)["'"'"']*\s*[:=]\s*["'"'"'][^"'"'"']+' | sort -u > tmp/js-secrets.txt

# LinkFinder for endpoint extraction (MUST use Burp proxy)
source /usr/local/bin/oc-proxy-env && for js in $(cat tmp/js-files.txt | head -20); do
  python3 ./tools/LinkFinder/LinkFinder.py -i "$js" -o cli 2>/dev/null
done | sort -u > tmp/linkfinder-endpoints.txt

# AUTO-QUEUE LinkFinder endpoints to Burp
cat tmp/linkfinder-endpoints.txt | oc-burp-queue-urls <project>
```

## Phase 6: Directory & File Fuzzing â€” AUTO-QUEUE TO BURP
```bash
# Directory discovery â€” auto-queues 2xx results to Burp
oc-ffuf <project> -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -mc 200,301,302,403 \
  -o tmp/dir-ffuf.json -of json

# File discovery with extensions â€” auto-queues to Burp
oc-ffuf <project> -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/files.txt \
  -mc 200 -e .php,.html,.js,.txt,.bak,.old,.zip,.tar.gz,.env,.git/config \
  -o tmp/files-ffuf.json -of json

# Hidden directories (deeper) â€” auto-queues to Burp
oc-ffuf <project> -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -mc 200,301,302,403 -recursion -recursion-depth 2 \
  -o tmp/dir-deep-ffuf.json -of json
```

## Phase 7: Parameter Discovery
```bash
# GET parameters (MUST use Burp proxy)
source /usr/local/bin/oc-proxy-env && arjun -u https://TARGET.COM/page -o tmp/params-get.json

# POST parameters (MUST use Burp proxy)
source /usr/local/bin/oc-proxy-env && arjun -u https://TARGET.COM/api/endpoint -m POST -o tmp/params-post.json

# JSON parameters (MUST use Burp proxy)
source /usr/local/bin/oc-proxy-env && arjun -u https://TARGET.COM/api/data -m JSON -o tmp/params-json.json
```

## Phase 8: Auth Flow Mapping
```bash
# Find login endpoints (from collected URLs â€” no HTTP)
cat tmp/all-urls.txt | grep -iE 'login|signin|auth|oauth|sso|callback|token|session' | sort -u > tmp/auth-urls.txt

# Find registration
cat tmp/all-urls.txt | grep -iE 'register|signup|create.?account|new.?user' | sort -u > tmp/register-urls.txt

# Find password reset
cat tmp/all-urls.txt | grep -iE 'reset|forgot|recover|password' | sort -u > tmp/reset-urls.txt
```

## Output: Update recon/STATUS.md
After each phase, update STATUS.md with:
- What was completed
- How many results (subdomains, live hosts, endpoints, etc.)
- What's queued for next
- Any tool failures or issues
