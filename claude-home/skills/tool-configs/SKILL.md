---
name: tool-configs
description: Optimal copy-paste-ready tool configurations for common bug bounty scenarios â€” ffuf, sqlmap, nuclei, httpx, subfinder, dalfox, gobuster, and naabu with recommended flags. ALL HTTP tools route through Burp 8080 for sitemap capture.
---
---

# Tool Configurations

Copy-paste ready commands for each tool. Each section has the most common use case plus context-specific variations.

## CRITICAL â€” Burp Proxy Routing (MANDATORY)

**Every HTTP request from hunting tools MUST route through Burp 8080 for sitemap capture.**

There are three ways to route tools through Burp, depending on the tool:

| Method | When to use |
|--------|-------------|
| Tool-specific flag (preferred) | ffuf, sqlmap, httpx, nuclei, curl |
| `source /usr/local/bin/oc-proxy-env` then run tool | dalfox, gobuster, arjun (no proxy flag) |
| `oc-proxy-cmd <tool> [args]` | Any tool as a last resort |

**Proxy flag reference by tool:**
| Tool | Proxy flag | Example |
|------|-----------|---------|
| `ffuf` | `-x http://127.0.0.1:8080` | `ffuf -x http://127.0.0.1:8080 -u ...` |
| `sqlmap` | `--proxy=http://127.0.0.1:8080` | `sqlmap --proxy=http://127.0.0.1:8080 -u ...` |
| `nuclei` | `-proxy http://127.0.0.1:8080` | `nuclei -proxy http://127.0.0.1:8080 -u ...` |
| `httpx` | `-proxy http://127.0.0.1:8080` | `httpx -proxy http://127.0.0.1:8080 -u ...` |
| `curl` | `--proxy http://127.0.0.1:8080` | `curl --proxy http://127.0.0.1:8080 https://...` |
| `dalfox` | env var: `HTTP_PROXY` | `source /usr/local/bin/oc-proxy-env && dalfox pipe ...` |
| `gobuster` | env var: `HTTP_PROXY` | `source /usr/local/bin/oc-proxy-env && gobuster dir ...` |
| `arjun` | env var: `HTTP_PROXY` | `source /usr/local/bin/oc-proxy-env && arjun -u ...` |

**TOOLS THAT DO NOT NEED PROXY (TCP scanners, not HTTP):**
- `naabu` â€” TCP port scanner, no HTTP proxy
- `nmap` â€” port scanner, no HTTP proxy
- `subfinder` â€” passive DNS, no HTTP requests to target
- `dnsx` â€” DNS resolver, no HTTP requests
- `amass` â€” passive enum, no HTTP requests to target

**TOOLS THAT MUST ALWAYS USE PROXY:**
- `ffuf` â€” HTTP fuzzing
- `sqlmap` â€” SQL injection (HTTP)
- `nuclei` â€” vulnerability scanning (HTTP)
- `httpx` â€” HTTP probing
- `dalfox` â€” XSS scanning (HTTP)
- `gobuster` â€” directory fuzzing (HTTP)
- `curl` â€” any manual HTTP request to target
- `arjun` â€” parameter discovery (HTTP)
- `subjs` â€” fetches JS files (HTTP)
- `katana` â€” web crawling (HTTP)
- `hakrawler` â€” web crawling (HTTP)
- `gospider` â€” web crawling (HTTP)
- `kxss` â€” XSS detection (HTTP)
- `Gxss` â€” XSS detection (HTTP)
- `Corsy` â€” CORS testing (HTTP)

## ffuf â€” Directory & Parameter Fuzzing

### Directory Discovery
```bash
ffuf -x http://127.0.0.1:8080 \
  -u "https://target.com/FUZZ" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -fc 404,403 -fs 0 -fw 0 \
  -mc 200,301,302,403 \
  -t 50 -rate 100 \
  -o tmp/ffuf-dirs.json -of json
```

### Parameter Discovery
```bash
ffuf -x http://127.0.0.1:8080 \
  -u "https://target.com/page" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/parameters.txt \
  -mc 200,301,302 \
  -fs 0 \
  -t 50 \
  -o tmp/ffuf-params.json -of json
```

### API Endpoint Fuzzing
```bash
ffuf -x http://127.0.0.1:8080 \
  -u "https://target.com/api/v1/FUZZ" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/api-endpoints.txt \
  -mc 200,201,400,401,403 \
  -fs 0 \
  -H "Authorization: Bearer $TOKEN" \
  -t 30 -rate 50 \
  -o tmp/ffuf-api.json -of json
```

### Subdomain Takeover (vhost)
```bash
ffuf -x http://127.0.0.1:8080 \
  -u "https://TARGET.com" \
  -H "Host: FUZZ.target.com" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/subdomains.txt \
  -fs 0 -mc 200 \
  -t 50 \
  -o tmp/ffuf-vhost.json -of json
```

### Key Flags Reference
| Flag | Purpose | Common Value |
|------|---------|-------------|
| `-x` | Burp proxy | `http://127.0.0.1:8080` |
| `-fc` | Filter by status code | `404,403` |
| `-fs` | Filter by response size | `0` (catch empty 404s) |
| `-fw` | Filter by word count | `0` |
| `-mc` | Match status codes | `200,301,302` |
| `-t` | Threads | `50` (default: 40) |
| `-rate` | Requests per second | `100` (be polite) |
| `-recursion` | Recursive fuzzing | `-recursion-depth 2` |
| `-e` | Extensions | `.php,.html,.js` |

## sqlmap â€” SQL Injection

### Basic Detection
```bash
sqlmap -u "https://target.com/page?id=1" \
  --proxy=http://127.0.0.1:8080 \
  --batch --level=3 --risk=2 \
  --random-agent --threads=4 \
  --output-dir=tmp/sqlmap-out
```

### POST Data
```bash
sqlmap -u "https://target.com/login" \
  --proxy=http://127.0.0.1:8080 \
  --data="user=admin&pass=test" \
  --batch --level=3 --risk=2 \
  --random-agent \
  --output-dir=tmp/sqlmap-out
```

### Cookie-Based
```bash
sqlmap -u "https://target.com/dashboard" \
  --proxy=http://127.0.0.1:8080 \
  --cookie="session=abc123; user=test" \
  --batch --level=2 --risk=1 \
  --random-agent \
  --output-dir=tmp/sqlmap-out
```

### WAF Bypass
```bash
sqlmap -u "https://target.com/page?id=1" \
  --proxy=http://127.0.0.1:8080 \
  --batch --level=3 --risk=2 \
  --tamper=space2comment,between,randomcase,charencode \
  --random-agent \
  --output-dir=tmp/sqlmap-out
```

### Specific DBMS
```bash
# MySQL
sqlmap -u "https://target.com/page?id=1" --proxy=http://127.0.0.1:8080 --dbms=mysql --batch --level=3

# PostgreSQL
sqlmap -u "https://target.com/page?id=1" --proxy=http://127.0.0.1:8080 --dbms=postgresql --batch --level=3

# MSSQL
sqlmap -u "https://target.com/page?id=1" --proxy=http://127.0.0.1:8080 --dbms=mssql --batch --level=3
```

### Tamper Script Reference
| Tamper Script | Use Case |
|--------------|----------|
| `space2comment` | Bypass space filters |
| `between` | Replace `>` and `<` with `BETWEEN` |
| `randomcase` | Randomize keyword case |
| `charencode` | URL-encode all characters |
| `equaltolike` | Replace `=` with `LIKE` |
| `greatest` | Replace `>` with `GREATEST` |
| `apostrophemask` | Replace `'` with `%EF%BC%87` |

## nuclei â€” Vulnerability Scanning

### By Severity (Recommended Approach)
```bash
# Critical only â€” quick sweep
echo "https://target.com" | nuclei -proxy http://127.0.0.1:8080 -severity critical -silent

# High + Critical
echo "https://target.com" | nuclei -proxy http://127.0.0.1:8080 -severity critical,high -silent

# Specific technology templates
echo "https://target.com" | nuclei -proxy http://127.0.0.1:8080 -t technologies/ -silent

# CVE scanning
echo "https://target.com" | nuclei -proxy http://127.0.0.1:8080 -t cves/ -severity critical,high -silent
```

### Rate Limiting & Exclusions
```bash
echo "https://target.com" | nuclei \
  -proxy http://127.0.0.1:8080 \
  -severity critical,high \
  -rate-limit 50 \
  -bulk-size 25 \
  -concurrency 10 \
  -timeout 5 \
  -silent
```

### URL List Scanning
```bash
cat tmp/live-hosts.txt | nuclei \
  -proxy http://127.0.0.1:8080 \
  -severity critical,high \
  -rate-limit 100 \
  -o tmp/nuclei-results.txt \
  -silent
```

### Key Flags
| Flag | Purpose |
|------|---------|
| `-proxy` | Burp proxy |
| `-severity` | Filter by severity: critical,high,medium,low,info |
| `-tags` | Filter by template tag: cve,xss,sqli etc |
| `-t` | Specific template path |
| `-rate-limit` | Max requests per second |
| `-bulk-size` | Templates to run in parallel |
| `-concurrency` | Hosts to scan in parallel |
| `-exclude-tags` | Skip template categories |

## httpx â€” Probing & Fingerprinting

### Standard Probe
```bash
cat recon/subdomains.txt | httpx -silent \
  -proxy http://127.0.0.1:8080 \
  -status-code -title -tech-detect \
  -follow-redirects -timeout 10 \
  -threads 50 \
  -o tmp/httpx-probe.txt
```

### Quick Live Check
```bash
cat recon/subdomains.txt | httpx -silent \
  -proxy http://127.0.0.1:8080 \
  -fc 404,403 \
  -follow-redirects \
  -o recon/live-hosts.txt
```

### Full Fingerprinting
```bash
cat recon/subdomains.txt | httpx -silent \
  -proxy http://127.0.0.1:8080 \
  -status-code -title -tech-detect \
  -content-length -web-server \
  -cdn -ip -cname \
  -follow-redirects \
  -threads 50 \
  -json \
  -o tmp/httpx-full.json
```

### Key Flags
| Flag | Purpose |
|------|---------|
| `-proxy` | Burp proxy |
| `-status-code` | Show HTTP status |
| `-title` | Extract page title |
| `-tech-detect` | Wappalyzer-style detection |
| `-follow-redirects` | Follow 3xx redirects |
| `-fc` | Filter status codes |
| `-json` | JSON output for parsing |
| `-threads` | Concurrent requests (default: 25) |

## subfinder â€” Subdomain Enumeration

> subfinder is passive DNS â€” it does NOT make HTTP requests to the target.
> No proxy needed. But subfinder uses HTTP to query DNS sources (crt.sh, etc.)
> which is safe and doesn't leak to target.

### Basic
```bash
subfinder -d TARGET.com -all -o tmp/subfinder.txt
```

### With API Keys
```bash
subfinder -d TARGET.com -all \
  -config ~/.config/subfinder/provider-config.yaml \
  -o tmp/subfinder.txt
```

### Provider Config Format (`~/.config/subfinder/provider-config.yaml`)
```yaml
binaryedge:
  - YOUR_KEY
censys:
  - YOUR_ID:YOUR_SECRET
virustotal:
  - YOUR_KEY
shodan:
  - YOUR_KEY
```

### Recursive
```bash
subfinder -d TARGET.com -recursive -all -o tmp/subfinder-recursive.txt
```

## dalfox â€” XSS Detection

> dalfox has no --proxy flag. Use env var.
```bash
# Reflected XSS
source /usr/local/bin/oc-proxy-env && echo "https://target.com/search?q=test" | dalfox pipe \
  --blind "https://YOUR_COLLABORATOR_URL" \
  --skip-bav \
  -o tmp/dalfox-results.txt

# With Custom Payloads
source /usr/local/bin/oc-proxy-env && echo "https://target.com/search?q=test" | dalfox pipe \
  --custom-payload '<script>alert(1)</script>' \
  --skip-bav \
  -o tmp/dalfox-results.txt

# Pipe Mode (Multiple URLs)
source /usr/local/bin/oc-proxy-env && cat tmp/urls-with-params.txt | dalfox pipe \
  --blind "https://YOUR_COLLABORATOR_URL" \
  --skip-bav \
  --timeout 10 \
  -o tmp/dalfox-batch.txt
```

### Key Flags
| Flag | Purpose |
|------|---------|
| `--blind` | Callback URL for blind XSS |
| `--skip-bav` | Skip basic XSS checks (faster) |
| `--custom-payload` | Use specific payload |
| `--timeout` | Request timeout |
| `-o` | Output file |

## gobuster â€” Directory/DNS/VHost

> gobuster has no --proxy flag. Use env var.
```bash
# Directory Mode
source /usr/local/bin/oc-proxy-env && gobuster dir -u "https://target.com" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -t 50 -q \
  --no-error -b 404,403 \
  -o tmp/gobuster-dirs.txt

# DNS Mode (no proxy needed â€” DNS resolution)
gobuster dns -d "target.com" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/subdomains.txt \
  -t 50 -q \
  -o tmp/gobuster-dns.txt

# VHost Mode
source /usr/local/bin/oc-proxy-env && gobuster vhost -u "https://target.com" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/subdomains.txt \
  -t 50 -q \
  --append-domain \
  -o tmp/gobuster-vhost.txt

# Wildcard Handling
source /usr/local/bin/oc-proxy-env && gobuster dir -u "https://target.com" \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  --wildcard -t 25 \
  -o tmp/gobuster-dirs.txt
```

## naabu â€” Port Scanning

> naabu is a TCP port scanner â€” it does NOT make HTTP requests.
> No proxy needed.

### Top Ports
```bash
cat recon/subdomains.txt | naabu -silent \
  -top-ports 1000 \
  -exclude-cdn \
  -o tmp/naabu-ports.txt
```

### Full Port Scan
```bash
cat recon/subdomains.txt | naabu -silent \
  -p - \
  -exclude-cdn \
  -o tmp/naabu-full.txt
```

### Specific Ports
```bash
cat recon/subdomains.txt | naabu -silent \
  -p 80,443,8080,8443,3000,5000 \
  -exclude-cdn \
  -o tmp/naabu-specific.txt
```

### Key Flags
| Flag | Purpose |
|------|---------|
| `-top-ports` | Scan top N ports |
| `-p` | Specific port(s): `80,443` or `-` for stdin |
| `-exclude-cdn` | Skip CDN IPs |
| `-type` | Scan type: `syn` (default) or `tcp` |
| `-timeout` | Connection timeout |
| `-retries` | Retry count |

## curl â€” Manual HTTP Requests

```bash
# Basic GET through Burp
curl --proxy http://127.0.0.1:8080 -s https://target.com/api/endpoint

# POST with JSON through Burp
curl --proxy http://127.0.0.1:8080 -s -X POST https://target.com/api/endpoint \
  -H "Content-Type: application/json" \
  -d '{"key":"value"}'

# With auth token through Burp
curl --proxy http://127.0.0.1:8080 -s https://target.com/api/endpoint \
  -H "Authorization: Bearer $TOKEN"

# Verbose (for debugging) through Burp
curl --proxy http://127.0.0.1:8080 -svk https://target.com/api/endpoint 2>&1 | head -50

# Follow redirects through Burp
curl --proxy http://127.0.0.1:8080 -sL https://target.com/redirect-url
```

## arjun â€” Parameter Discovery

> arjun has no --proxy flag. Use env var.
```bash
# GET parameters
source /usr/local/bin/oc-proxy-env && arjun -u https://TARGET.COM/page -o tmp/params-get.json

# POST parameters
source /usr/local/bin/oc-proxy-env && arjun -u https://TARGET.COM/api/endpoint -m POST -o tmp/params-post.json

# JSON parameters
source /usr/local/bin/oc-proxy-env && arjun -u https://TARGET.COM/api/data -m JSON -o tmp/params-json.json
```

## Other HTTP Tools

> All HTTP tools MUST route through Burp. Quick reference:

| Tool | Proxy method |
|------|-------------|
| `katana` | `source /usr/local/bin/oc-proxy-env && katana -u ...` |
| `hakrawler` | `source /usr/local/bin/oc-proxy-env && echo URL \| hakrawler` |
| `gospider` | `source /usr/local/bin/oc-proxy-env && gospider -s URL` |
| `subjs` | `source /usr/local/bin/oc-proxy-env && cat urls \| subjs` |
| `kxss` | `source /usr/local/bin/oc-proxy-env && cat urls \| kxss` |
| `Corsy` | `source /usr/local/bin/oc-proxy-env && python3 corsy.py -u URL` |
| `LinkFinder` | `source /usr/local/bin/oc-proxy-env && python3 LinkFinder.py -i URL -o cli` |
