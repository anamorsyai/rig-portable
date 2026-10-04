---
name: hunting-methodology
description: Ultimate bug bounty hunting methodology — prevents random walking, systematic 7-phase approach. Business intel, attack surface construction, hypothesis-driven testing, recursive depth, exploit & chain. Load this at engagement start.
---

# Ultimate Hunting Methodology

## Purpose

This is the master framework that prevents random walking. Every agent loads this skill at the start of an engagement and follows these phases systematically. The methodology is structured but not rigid — agents can flex between phases as new information surfaces, but they must NEVER skip a phase without explicitly stating why.

## Phase 0: Business Intelligence & Target Understanding

### Target Profile Card

```markdown
## Target Profile: {TARGET_NAME}

### Business
- Product/Service: {what they sell}
- Customers: {enterprise/SMB/consumer/mixed}
- Revenue Model: {subscription/ads/transaction/marketplace/freemium}
- Geography: {where they operate}

### Risk Profile
- Crown Jewel Data: {PII/payment/health/credentials/IP/credit cards}
- Biggest Breach Risk: {data exposure/financial fraud/account takeover/reputation}
- Bounty Expectation: {$500-$10k range}

### Technical
- Stack: {from job listings, GitHub, BuiltWith}
- Auth: {JWT/OAuth/SAML/cookies/magic links}
- API: {REST/GraphQL/gRPC/SOAP}
- WAF/CDN: {Cloudflare/Akamai/AWS WAF/CloudFront/Fastly}
- DB: {PostgreSQL/MySQL/MongoDB/DynamoDB/Redis}
- Hosting: {AWS/GCP/Azure/on-prem/Heroku/Netlify}
```

### OSINT Collection

```bash
# What does the company do?
echo "Visit the website, read about page, understand the product"
curl -s "https://TARGET.COM/about" | grep -oiP '(?:we\s+(?:are|build|provide|offer|sell|help)\s+).{50,200}' | head -5

# Tech stack from job listings
# Delegate to @intel: "search TARGET job listings for tech stack keywords"
# Common keywords: React, Angular, Vue, Django, Rails, Spring, Laravel, Go, Rust, PostgreSQL, MongoDB, Redis, Kafka, AWS, GCP, Docker, Kubernetes

# Who are competitors?
# Same tech stacks often share same vulnerabilities

# What's the biggest security risk to this business?
# Fintech: payment manipulation, account takeover
# Healthcare: PHI exposure, compliance bypass
# SaaS: multi-tenant data leakage, privilege escalation
# E-commerce: payment bypass, price manipulation
# Social: account takeover, data scraping
```

## Phase 1: Attack Surface Construction

### Step 1.1 — Asset Inventory

```bash
# Subdomain enumeration (combine ALL sources)
subfinder -d TARGET.COM -silent | tee recon/subfinder.txt
amass enum -passive -d TARGET.COM -o recon/amass.txt 2>/dev/null
curl -s "https://crt.sh/?q=%.TARGET.COM&output=json" | jq -r '.[].name_value' | sort -u > tmp/crtsh.txt

# DNS brute force
puredns bruteforce C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/subdomains.txt TARGET.COM | tee tmp/puredns.txt

# Combine all
cat tmp/subfinder.txt tmp/amass.txt tmp/crtsh.txt tmp/puredns.txt | sort -u > recon/all-subdomains.txt

# Live host probing
cat recon/all-subdomains.txt | httpx -silent -status-code -title -tech-detect -o recon/live-hosts.txt
```

### Step 1.2 — Technology Stack

```bash
# WAF detection
wafw00f -a https://TARGET.COM 2>&1 | tee recon/waf-detect.txt

# Tech fingerprinting (BEST single command)
whatweb -a 3 https://TARGET.COM 2>&1 | tee recon/whatweb.txt

# All live hosts
cat recon/live-hosts.txt | whatweb -a 3 --log-verbose=recon/whatweb-all.txt 2>/dev/null

# Check all live hosts for WAF
for host in $(cat recon/live-hosts.txt | awk '{print $1}'); do
  wafw00f -a "$host" 2>&1 | grep -i "detected\|behind" >> recon/waf-all.txt
done
```

### Step 1.3 — Endpoint Discovery

```bash
# Directory fuzzing (aggressive)
ffuf -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -t 30 -rate 50 \
  -mc 200,301,302,401,403,405,500 \
  -ac -o map/directories.json

# API endpoint discovery
ffuf -u https://TARGET.COM/api/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/api-endpoints.txt \
  -t 30 -rate 50 \
  -mc 200,201,401,403,405 \
  -o map/api-endpoints.json

# Wayback/GAU for historical endpoints
cat recon/live-hosts.txt | waybackurls | sort -u > map/wayback.txt
cat recon/live-hosts.txt | gau --threads 10 | sort -u > map/gau.txt
cat map/wayback.txt map/gau.txt | sort -u > map/all-urls.txt

# Swagger/OpenAPI discovery
ffuf -u https://TARGET.COM/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/SecLists/Discovery/Web-Content/swagger.txt \
  -mc 200 -o map/swagger.json

# JS analysis
cat map/all-urls.txt | grep -i '\.js$' | sort -u > map/js-files.txt
cat map/js-files.txt | subjs | tee map/subjs.txt
```

### Step 1.4 — Authentication Flow Mapping

```bash
# Map ALL authentication endpoints
# Check every auth path
for path in login signup register auth oauth saml sso callback \
            password-reset forgot-password reset-password \
            verify verification confirm confirmation \
            logout session 2fa mfa totp; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "https://TARGET.COM/$path")
  echo "$path -> HTTP $STATUS" >> map/auth-endpoints.txt
done

# Social login discovery
curl -s "https://TARGET.COM/login" | grep -oiP 'google|github|facebook|apple|twitter|linkedin|microsoft|gitlab|bitbucket' | sort -u

# SSO discovery
curl -s "https://TARGET.COM/login" | grep -oiP 'saml|oauth|oidc|openid|sso|okta|auth0|onelogin|azure|ping' | sort -u
```

### Step 1.5 — Authorization Boundary Mapping

```bash
# Create accounts.md with role/tenant structure (see /multi-tenant-accounts)

# Test access boundaries manually
# Anonymous access
for path in /admin /api/admin /dashboard /internal /private; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "https://TARGET.COM$path")
  echo "Anonymous: $path -> HTTP $STATUS"
done

# User access
for path in /admin /api/admin /dashboard /api/users /api/settings /api/billing; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "https://TARGET.COM$path" \
    -H "Authorization: Bearer $USER_TOKEN")
  echo "User: $path -> HTTP $STATUS"
done
```

## Phase 2: Attack Surface Ranking

### Priority Matrix

```python
# Every discovered endpoint is ranked by potential impact
PRIORITY_TIERS = {
    "P0 - Critical (test first)": [
        "/api/login", "/api/auth/*", "/oauth/*", "/saml/*",
        "/api/charge", "/api/payment", "/api/billing", "/api/checkout",
        "/api/admin/**", "/api/internal/**",
        "/graphql", "/api/graphql",
        "/api/users/**", "/api/organizations/**",
    ],
    "P1 - High": [
        "/api/password-reset", "/api/2fa", "/api/mfa",
        "/api/upload", "/api/import", "/api/export",
        "/api/profile/**", "/api/settings/**",
        "/api/search", "/api/filter",
        "/api/execute", "/api/rpc", "/api/command",
    ],
    "P2 - Medium": [
        "/api/**/list", "/api/**/search",
        "/api/notifications", "/api/webhooks",
        "/api/reports", "/api/analytics",
        "/api/feedback", "/api/comments",
    ],
    "P3 - Low (brief check)": [
        "/*.html", "/*.php", "/*.aspx",
        "/robots.txt", "/sitemap.xml",
        "/api/health", "/api/version",
        "/api/docs", "/api/status",
    ],
}
```

## Phase 3: Hypothesis-Driven Testing

### Hypothesis Template

```markdown
### Hypothesis: {ID-HXXX}
**Target**: {endpoint}
**Vulnerability Class**: {SQLi/IDOR/SSRF/XSS/BAC/Auth bypass/BizLogic/Other}
**Reasoning**: {why this might be vulnerable - specific observation}
**Impact if True**: {what the attacker gains}
**Test Plan**:
1. {step 1}
2. {step 2}
3. {step 3}
```

### Testing Loop

Every hypothesis goes through the recursive testing loop:

1. **Generate** — create 5-10 variations before firing the first request
2. **Execute** — send the most promising variant
3. **Observe** — examine status code, response length, timing, error content
4. **Chain** — feed results into the next variant
5. **Branch** — if blocked, try a different vuln class on the same input
6. **Recurse** — any new endpoint/param/token goes through the same loop

## Phase 4: Systematic Class Coverage

For EVERY endpoint, test in this order (never skip higher-impact classes):

### T1 — Critical Classes
- [ ] Broken Access Control (IDOR/BOLA, privilege escalation, mass assignment)
- [ ] Authentication Bypass (JWT, OAuth, session, 2FA, password reset)
- [ ] Injection (SQLi, NoSQLi, command injection, SSTI, XXE, deserialization)
- [ ] SSRF (metadata endpoints, internal services, OOB detection)

### T2 — High Classes
- [ ] Business Logic (price manipulation, race conditions, workflow bypass)
- [ ] Stored XSS (admin-targeted, cookie theft, session hijacking)
- [ ] File Upload (RCE via webshell, path traversal, XXE in SVG)

### T3 — Medium Classes  
- [ ] Reflected XSS, CSRF on state-changing endpoints
- [ ] Open redirect (OAuth callback chaining)
- [ ] Cache poisoning / WebSocket hijack

### T4 — Low (brief check only)
- [ ] Information disclosure, verbose errors, version leaks
- [ ] Missing security headers, CORS misconfig

## Phase 5: Recursive Depth

When a finding or interesting response appears, immediately test:

```bash
# For an IDOR on /api/users/123 -> test ALL of these:
test_endpoint "GET /api/users/me"
test_endpoint "GET /api/users?limit=100&offset=0"
test_endpoint "GET /api/users/123/profile"
test_endpoint "GET /api/users/123/settings"
test_endpoint "GET /api/users/123/billing"
test_endpoint "GET /api/admins"
test_endpoint "PUT /api/users/123"
test_endpoint "PATCH /api/users/123"
test_endpoint "DELETE /api/users/123"
test_endpoint "POST /api/users/123/impersonate"

# For a SQLi on /search?q= -> test ALL params that hit the DB:
test_param "filter"
test_param "sort"
test_param "order"
test_param "category"
test_param "type"
test_param "status"
test_param "from"
test_param "to"
test_param "range"
```

## Phase 6: Exploit & Chain

Every confirmed finding MUST be escalated through at least 3 chain attempts:

### Escalation Ladder

```python
def escalate_finding(finding):
    """Run 3+ chain attempts for every confirmed finding"""
    chains_tried = []
    
    # Attempt 1: Horizontal escalation (same class, different scope)
    chains_tried.append(try_horizontal_chain(finding))
    
    # Attempt 2: Vertical escalation (combine with another class)
    chains_tried.append(try_vertical_chain(finding))
    
    # Attempt 3: Business-logic chain
    chains_tried.append(try_bizlogic_chain(finding))
    
    # After 3 attempts, check chain library for known patterns
    load_skill("exploit-chains")
    chains_tried.append(try_known_chains(finding))
    
    return max(chains_tried, key=lambda c: c.severity)
```

### Chain Examples

| Base Finding | Chain With | Resulting Impact |
|-------------|-----------|------------------|
| IDOR on profile | + Email change | Account takeover |
| SQLi on search | + File read | Source code + secrets |
| SSRF on fetch | + Cloud metadata | IAM credentials |
| XSS on profile | + CORS misconfig | Data theft |
| Race on coupon | + Multiple coupons | Financial loss |
| Auth bypass on API | + Admin panel | Full admin access |

## Phase 7: Coverage Validation

Before leaving an endpoint, pass through this checklist:

### Endpoint Clearance Checklist

- [ ] Tested before auth (anonymous access)
- [ ] Tested with lowest role
- [ ] Tested with admin/highest role
- [ ] Tested ALL HTTP methods (GET, POST, PUT, PATCH, DELETE, OPTIONS)
- [ ] Tested content type variations (JSON, XML, form, multipart)
- [ ] Tested IDOR (parameter swap between two accounts)
- [ ] Tested injection (SQLi, NoSQLi, command, SSTI, XXE)
- [ ] Tested mass assignment (role, isAdmin, permissions fields)
- [ ] Tested rate limit / brute force potential
- [ ] Discovered parameters fuzzed (debug, admin, override, test)
- [ ] Checked for hidden paths/directories around the endpoint
- [ ] Error messages checked for info disclosure
- [ ] Response headers checked for leaks (Server, X-Powered-By, etc.)
- [ ] Documented in STATUS.md with what was tested and result

## Methodology Enforcement

The orchestrator (@build) enforces methodology compliance:

```python
# At every checkpoint, verify:
compliance = {
    "phase_0_business_intel": file_exists("recon/target-profile.md"),
    "phase_1_attack_surface": file_exists("recon/all-subdomains.txt"),
    "phase_1_tech_stack": file_exists("recon/whatweb.txt"),
    "phase_2_ranking": file_exists("map/priority-matrix.md"),
    "phase_3_hypotheses": file_has_content("vuln/hypotheses.md"),
    "phase_4_coverage": file_has_content("vuln/coverage-checklist.md"),
    "phase_5_recursive": file_has_content("vuln/findings-graph.md"),
    "phase_7_clearance": file_exists("map/endpoint-clearance.md"),
}
```

**Agents can flex between phases but can NEVER skip Phase 0 (business intel) or Phase 7 (coverage validation).**

## Phase 8: Submission Doctrine Gate (quality over quantity)

**Every completed, proven finding passes the doctrine gate BEFORE anything is labeled
"reportable." Loading `strict-submission-doctrine` is mandatory before the report phase.**
No finding leaves the engagement unless it clears this gate.

### The strict doctrine in one line

> **You prefer 0 findings over weak or low-value findings. Impact must be demonstrated,
> not theorized; severity must be accurate, not inflated.**

### Class proof bars (minimum to even consider reporting)

| Class | You MUST show | Not acceptable |
|---|---|---|
| XSS | Actual JS execution in a realistic context | HTML injection / reflection alone |
| IDOR / BOLA | Another user's or tenant's data/actions | Same-data-for-all-IDs, your own data |
| SSRF | Interaction with internal services / meaningful impact | "Server tried to connect" |
| SQLi / NoSQLi | Data access or modification (extracted rows, creds) | "Could execute arbitrary SQL" |
| Info disclosure | A real downstream attack path (creds, tokens, exploitable internal endpoint) | Hostnames / config / version alone |
| Auth bypass / ATO | Full end-to-end chain, minimal interaction | "Could potentially take over" |
| Privesc | A real admin/higher-priv action from a lower role | "Could escalate privileges" |

### Hard-ban checklist (never report, no matter how "interesting")

- [ ] User enumeration alone
- [ ] Missing security headers
- [ ] Framework / server version disclosure
- [ ] Missing rate limiting without proven impact
- [ ] HTML injection without JS execution
- [ ] Theoretical / "potential" issues
- [ ] Staging refs, internal hostnames, or config leaks without an exploitation path
- [ ] Open dirs, robots.txt, security.txt
- [ ] Self-signed certs / weak TLS without a real attack
- [ ] Best-practice recommendations dressed as vulns
- [ ] Admin-only "issues" that aren't a real privesc
- [ ] Mass assignment / missing validation without demonstrated impact
- [ ] 500 errors or error messages alone

### The three gates (all must pass) — combine with `mandatory-triage-gate`
1. **Real impact demonstrated** — I DID access other-user data / execute JS / reach an
   internal service / move money. Name the concrete artifact, not a scenario.
2. **Reproducible under 5 min** by a zero-context hostile triager from raw requests.
3. **"So what?" answered** — I'd pay $500+ for this; it's not closed Informational in 2 min.

### Decision
- Pass all gates → report it (Critical findings: stop, show evidence, wait for approval).
- Fail any gate → **Rejected / Low-Value**, listed with the reason, or "Needs more validation"
  if impact is merely unclear. Never promote it without new proof.
- Nothing passes → state plainly: **"No high-quality findings identified that meet submission standards."**
  This is a win. Silence on weak findings is a feature, not a failure.
