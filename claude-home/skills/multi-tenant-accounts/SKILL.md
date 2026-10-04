---
name: multi-tenant-accounts
description: Multi-tenant identity management — identity pool architecture, role/tenant provisioning, account lifecycle testing, session rotation, cross-tenant IDOR testing methodology, automated Python identity manager, disposable email integration.
---

# Multi-Tenant / Roles / Accounts — Identity Management

## 1. The Identity Challenge

Modern applications have complex identity structures. A single engagement may need:
- Multiple tenants (org-a, org-b, org-c)
- Multiple roles per tenant (user, admin, owner, billing admin, etc.)
- Different account states (active, suspended, expired, deleted)
- Multiple auth methods (email, OAuth, SSO, magic links)
- Session tokens that expire and need rotation

Without systematic identity management, agents waste time re-creating accounts, testing without cross-tenant access, and failing to catch authorization gaps.

## 2. Identity Pool Architecture

### 2.1 Minimum Viable Identity Set

```bash
# For ANY engagement, provision this minimum:
# 1. Two accounts on Tenant A:
#    - Account A1: base role (user/member)
#    - Account A2: elevated role (admin/owner)

# 2. One account on Tenant B:
#    - Account B1: base role (user/member)

# 3. Additional based on scope:
#    - SSO/OAuth linked account
#    - Deleted/suspended account
#    - Expired subscription account
#    - Different region/locale account
#    - API-key only account
#    - Read-only / restricted account
```

### 2.2 Disposable Email Integration

```bash
# Use mail.tm API for disposable email
# Email creation
create_temp_email() {
  local response=$(curl -s -X POST "https://api.mail.tm/accounts" \
    -H "Content-Type: application/json" \
    -d "{\"address\":\"$1\",\"password\":\"$2\"}")
  echo "$response" | jq -r '.id'
}

# Get auth token for inbox access
get_mail_token() {
  local response=$(curl -s -X POST "https://api.mail.tm/token" \
    -H "Content-Type: application/json" \
    -d "{\"address\":\"$1\",\"password\":\"$2\"}")
  echo "$response" | jq -r '.token'
}

# Poll inbox for verification links/codes
poll_inbox() {
  local token=$1
  local account_id=$2
  
  for i in $(seq 1 30); do
    sleep 2
    local messages=$(curl -s "https://api.mail.tm/messages" \
      -H "Authorization: Bearer $token")
    
    local count=$(echo "$messages" | jq -r '.hydra:totalItems // 0')
    if [ "$count" -gt 0 ]; then
      local msg_id=$(echo "$messages" | jq -r '.["hydra:member"][0].id')
      local content=$(curl -s "https://api.mail.tm/messages/$msg_id" \
        -H "Authorization: Bearer $token")
      echo "$content"
      return 0
    fi
  done
  echo "No messages received within 60 seconds"
  return 1
}

# Extract verification link
extract_verify_link() {
  echo "$1" | grep -oP 'https?://[^"'"'"'\s]+verify[^"'"'"'\s]+'
}

# Extract OTP code
extract_otp() {
  echo "$1" | grep -oP '\b\d{4,8}\b' | head -1
}
```

### 2.3 Identity Ledger Format

```markdown
# Accounts Ledger — {TARGET}
| ID | Email | Password | Role | Tenant | Status | Token/Cookie | Created | Provider | Notes |
|----|-------|----------|------|--------|--------|-------------|---------|----------|-------|
| A01 | user@tenant-a.com | Pass123! | user | tenant-a | active | eyJ... | 2024-01-01 | mail.tm | Standard base account |
| A02 | admin@tenant-a.com | Pass123! | admin | tenant-a | active | eyJ... | 2024-01-01 | mail.tm | Admin on same tenant |
| A03 | user@tenant-b.com | Pass123! | user | tenant-b | active | eyJ... | 2024-01-01 | mail.tm | Cross-tenant account |
| A04 | enterprise@a.com | Pass123! | enterprise | tenant-a | active | eyJ... | 2024-01-01 | mail.tm | Enterprise features |
| A05 | expired@a.com | Pass123! | user | tenant-a | expired | - | 2024-01-01 | mail.tm | Expired subscription |
| A06 | sso-user@a.com | Pass123! | user | tenant-a | active | eyJ... | 2024-01-01 | OAuth:Google | OAuth-linked account |
```

## 3. Automated Provisioning Script

```python
#!/usr/bin/env python3
"""
Automated identity provisioner for multi-tenant engagements.
Creates accounts across tenants, handles verification flows,
and maintains the identity ledger.
"""
import requests
import json
import time
import re
import sys
from datetime import datetime

class IdentityProvisioner:
    def __init__(self, target_url):
        self.target = target_url.rstrip('/')
        self.accounts = []
        self.mail_api = "https://api.mail.tm"
    
    def create_disposable_email(self):
        """Create a temporary email via mail.tm"""
        import random
        import string
        
        prefix = ''.join(random.choices(string.ascii_lowercase, k=10))
        email = f"{prefix}@tm.tm"
        password = "BugBounty2024!"
        
        # Create account
        r = requests.post(f"{self.mail_api}/accounts", json={
            "address": email,
            "password": password
        })
        if r.status_code != 201:
            print(f"[-] Failed to create email: {r.text}")
            return None, None
        
        account_id = r.json().get("id")
        
        # Get token
        r = requests.post(f"{self.mail_api}/token", json={
            "address": email,
            "password": password
        })
        token = r.json().get("token")
        
        return email, token, account_id
    
    def poll_for_link(self, token, timeout=60, interval=3):
        """Poll inbox for verification link"""
        start = time.time()
        while time.time() - start < timeout:
            r = requests.get(f"{self.mail_api}/messages", 
                           headers={"Authorization": f"Bearer {token}"})
            messages = r.json()
            count = messages.get("hydra:totalItems", 0)
            
            if count > 0:
                msg_id = messages["hydra:member"][0]["id"]
                msg = requests.get(f"{self.mail_api}/messages/{msg_id}",
                                  headers={"Authorization": f"Bearer {token}"})
                content = msg.json()
                
                # Extract HTML content
                html = content.get("html", [""])[0]
                
                # Find verification links
                links = re.findall(r'https?://[^"\']*verify[^"\'\s]*', html)
                otp = re.findall(r'\b(\d{4,8})\b', html)
                
                return {"links": links, "otp": otp, "content": html}
            
            time.sleep(interval)
        
        return None
    
    def create_account(self, email, password, role="user", tenant=None):
        """Register an account on the target"""
        signup_endpoints = [
            f"{self.target}/api/signup",
            f"{self.target}/api/register",
            f"{self.target}/api/v1/auth/register",
            f"{self.target}/signup",
            f"{self.target}/accounts/signup",
        ]
        
        payloads = [
            {"email": email, "password": password, "password_confirmation": password},
            {"email": email, "password": password, "confirmPassword": password},
            {"email": email, "password": password, "password2": password},
            {"user": {"email": email, "password": password}},
        ]
        
        # Try tenant-specific fields
        if tenant:
            for p in payloads:
                p["organization_name"] = tenant
                p["company"] = tenant
                p["tenant_id"] = tenant
        
        for endpoint in signup_endpoints:
            for payload in payloads:
                try:
                    r = requests.post(endpoint, json=payload, timeout=10)
                    if r.status_code in [200, 201, 204]:
                        return endpoint, r.json()
                except:
                    continue
        
        return None, None
    
    def login(self, email, password):
        """Login and get session token"""
        login_endpoints = [
            f"{self.target}/api/login",
            f"{self.target}/api/auth/login",
            f"{self.target}/api/v1/auth/login",
            f"{self.target}/login",
            f"{self.target}/auth",
            f"{self.target}/api/token",
            f"{self.target}/oauth/token",
        ]
        
        payloads = [
            {"email": email, "password": password},
            {"username": email, "password": password},
            {"email": email, "password": password, "grant_type": "password"},
        ]
        
        for endpoint in login_endpoints:
            for payload in payloads:
                try:
                    r = requests.post(endpoint, json=payload, timeout=10)
                    if r.status_code in [200, 201]:
                        data = r.json()
                        # Try to extract token
                        for key in ["token", "access_token", "accessToken", 
                                   "jwt", "session", "id_token"]:
                            if key in data:
                                cookies = r.cookies.get_dict()
                                return data[key], cookies, endpoint
                except:
                    continue
        
        return None, None, None
    
    def provision_identities(self, config):
        """
        Provision multiple identities based on config.
        
        config = {
            "tenants": ["tenant-a", "tenant-b"],
            "roles_per_tenant": {"tenant-a": ["user", "admin"], "tenant-b": ["user"]},
            "additional": ["sso", "expired"]
        }
        """
        identities = []
        
        for tenant in config["tenants"]:
            for role in config["roles_per_tenant"][tenant]:
                print(f"[*] Provisioning {role}@{tenant}...")
                
                email, mail_token, _ = self.create_disposable_email()
                if not email:
                    print(f"[-] Failed to create email for {role}@{tenant}")
                    continue
                
                password = "BugBounty2024!"
                
                # Create account
                endpoint, response = self.create_account(email, password, role, tenant)
                if not endpoint:
                    print(f"[-] Failed to create account for {role}@{tenant}")
                    continue
                
                # Handle verification
                verification = self.poll_for_link(mail_token)
                if verification:
                    for link in verification.get("links", []):
                        try:
                            requests.get(link, timeout=10)
                            print(f"[+] Verified {role}@{tenant}")
                        except:
                            pass
                
                # Login
                token, cookies, login_ep = self.login(email, password)
                
                identity = {
                    "id": f"A{len(identities)+1:02d}",
                    "email": email,
                    "password": password,
                    "role": role,
                    "tenant": tenant,
                    "status": "active",
                    "token": token[:50] + "..." if token else None,
                    "cookies": json.dumps(cookies) if cookies else None,
                    "created": datetime.now().isoformat(),
                }
                identities.append(identity)
                print(f"[+] {role}@{tenant} provisioned: {email}")
                
                time.sleep(1)  # Rate limit between signups
        
        return identities
    
    def export_ledger(self, identities, filename="accounts.md"):
        """Export to markdown ledger"""
        with open(filename, 'w') as f:
            f.write(f"# Accounts Ledger\n\n")
            f.write(f"| ID | Email | Role | Tenant | Status | Token | Created |\n")
            f.write(f"|----|-------|------|--------|--------|-------|--------|\n")
            for acct in identities:
                token_short = (acct.get("token", "") or "")[:20] + "..."
                f.write(f"| {acct['id']} | {acct['email']} | {acct['role']} | {acct['tenant']} | "
                       f"{acct['status']} | {token_short} | {acct['created']} |\n")
        print(f"[+] Ledger exported to {filename}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 provision.py <target_url>")
        sys.exit(1)
    
    provisioner = IdentityProvisioner(sys.argv[1])
    
    config = {
        "tenants": ["tenant-a", "tenant-b"],
        "roles_per_tenant": {
            "tenant-a": ["user", "admin"],
            "tenant-b": ["user"]
        },
        "additional": []
    }
    
    identities = provisioner.provision_identities(config)
    provisioner.export_ledger(identities)
```

## 4. Multi-Tenant Testing Methodology

### 4.1 Cross-Tenant IDOR

```bash
# Tenant A session, Tenant B resources
# For every API endpoint, test across tenant boundaries

# User ID from Tenant B
TENANT_B_USER_ID=$(grep "user.*tenant-b" accounts.md | head -1 | awk '{print $6}')
TENANT_A_TOKEN=$(grep "admin.*tenant-a" accounts.md | head -1 | awk '{print $6}')

# Test as Tenant A admin, targeting Tenant B user
curl -s "https://TARGET.COM/api/users/$TENANT_B_USER_ID/profile" \
  -H "Authorization: Bearer $TENANT_A_TOKEN"

# Test org-level access
curl -s "https://TARGET.COM/api/organizations/2/settings" \
  -H "Authorization: Bearer $TENANT_A_TOKEN"

# Test resource ID belonging to other tenant
curl -s "https://TARGET.COM/api/projects?org=2" \
  -H "Authorization: Bearer TENANT_A_TOKEN"
```

### 4.2 Tenant ID Manipulation

```bash
# Try tenant ID in various locations
curl -s "https://TARGET.COM/api/dashboard" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 2"

curl -s "https://TARGET.COM/api/dashboard?tenant=2" \
  -H "Authorization: Bearer $TOKEN"

curl -s "https://TARGET.COM/api/dashboard?organization=2" \
  -H "Authorization: Bearer $TOKEN"

curl -s "https://TARGET.COM/api/dashboard?org=2" \
  -H "Authorization: Bearer $TOKEN"

curl -s "https://TARGET.COM/api/dashboard" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Organization-ID: 2"

# JWT tenant claim manipulation
# Decode JWT, modify org/tenant/company claim, re-encode, test
python3 -c "
import jwt, base64, json
token = '$TOKEN'
parts = token.split('.')
header = json.loads(base64.b64decode(parts[0] + '=='))
payload = json.loads(base64.b64decode(parts[1] + '=='))

# Common tenant claim names
for claim in ['org', 'org_id', 'organization', 'organization_id', 'tenant', 'tenant_id', 
              'company', 'company_id', 'workspace', 'workspace_id', 'team', 'team_id',
              'account', 'account_id', 'customer_id', 'cid']:
    if claim in payload:
        print(f'Found tenant claim: {claim} = {payload[claim]}')
        old = payload[claim]
        payload[claim] = 2  # Try other tenant
        new_token = base64.urlsafe_b64encode(json.dumps(header).encode()).rstrip(b'=').decode() + '.' + \
                    base64.urlsafe_b64encode(json.dumps(payload).encode()).rstrip(b'=').decode() + '.'
        print(f'Modified JWT: {new_token[:50]}...')
"
```

### 4.3 Role Escalation Across Tenants

```bash
# Test if a user in Tenant A can assume admin role in Tenant B
ADMIN_TENANT_B=$(grep "admin.*tenant-b" accounts.md | head -1)
if [ -z "$ADMIN_TENANT_B" ]; then
  # Try to create an admin account in Tenant B via the invite flow
  USER_TENANT_B_TOKEN=$(grep "user.*tenant-b" accounts.md | head -1 | awk '{print $6}')
  
  # Try inviting yourself as admin
  curl -s -X POST "https://TARGET.COM/api/organizations/2/invite" \
    -H "Authorization: Bearer $USER_TENANT_B_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"email": "admin@tenant-b.com", "role": "admin"}'
  
  # Try modifying your own role
  curl -s -X PUT "https://TARGET.COM/api/users/me" \
    -H "Authorization: Bearer $USER_TENANT_B_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"role": "admin"}'
  
  # Try adding yourself to admin group
  curl -s -X POST "https://TARGET.COM/api/groups/1/members" \
    -H "Authorization: Bearer $USER_TENANT_B_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"user_id": "me"}'
fi
```

## 5. Role Testing Matrix

### 5.1 Complete Role Matrix

```python
ROLE_MATRIX = {
    "anonymous": {
        "level": 0,
        "can": ["/login", "/signup", "/password-reset", "/api/public"],
        "cannot": ["/api/me", "/api/admin", "/api/dashboard"],
    },
    "user": {
        "level": 10,
        "can": ["/api/me", "/api/profile", "/api/search"],
        "cannot": ["/api/admin", "/api/users", "/api/billing/all"],
    },
    "premium": {
        "level": 20,
        "can": ["/api/premium", "/api/advanced-search"],
        "cannot": ["/api/admin", "/api/billing/all"],
    },
    "moderator": {
        "level": 50,
        "can": ["/api/reports", "/api/content/moderate"],
        "cannot": ["/api/admin/settings", "/api/billing"],
    },
    "admin": {
        "level": 99,
        "can": ["/api/admin", "/api/users", "/api/settings"],
        "cannot": ["/api/superadmin", "/api/billing/global"],
    },
    "superadmin": {
        "level": 100,
        "can": ["/api/superadmin", "/api/billing/all", "/api/config"],
        "cannot": [],
    },
}

# For every endpoint, for each role, test:
# 1. With correct role -> must succeed
# 2. With lower role -> must fail (403/401)
# 3. With no auth -> must fail if authenticated endpoint
```

### 5.2 Automated Role Testing

```python
#!/usr/bin/env python3
"""
Test every endpoint against every role automatically.
"""
import requests

TARGET = "https://target.com"
ENDPOINTS = [
    ("GET", "/api/users/me"),
    ("POST", "/api/users"),
    ("GET", "/api/admin/users"),
    ("GET", "/api/admin/settings"),
    ("GET", "/api/billing/invoices"),
    ("GET", "/api/dashboard"),
]

ROLES = {
    "anonymous": {"token": None},
    "user": {"token": "USER_TOKEN"},
    "admin": {"token": "ADMIN_TOKEN"},
}

results = []
for method, path in ENDPOINTS:
    for role, data in ROLES.items():
        headers = {}
        if data["token"]:
            headers["Authorization"] = f"Bearer {data['token']}"
        
        r = requests.request(method, f"{TARGET}{path}", headers=headers)
        results.append({
            "role": role,
            "method": method,
            "path": path,
            "status": r.status_code,
            "length": len(r.content),
        })
        print(f"  {role:12s} {method:6s} {path:40s} -> {r.status_code}")

# Find anomalies
for r in results:
    if r["role"] == "user" and r["status"] in [200, 201]:
        print(f"[!] User accessed: {r['method']} {r['path']}")
```

## 6. Account Lifecycle Testing

### 6.1 Lifecycle States

Every account goes through: Created → Verified → Active → (Suspended/Banned) → (Expired) → (Deleted)

Test access at EVERY state:

```bash
# 1. Pre-verification (registered but not verified)
# Use token from registration response before clicking verification link
curl -s "https://TARGET.COM/api/premium/features" \
  -H "Authorization: Bearer $REGISTRATION_TOKEN"

# 2. Active (verified, good standing)
curl -s "https://TARGET.COM/api/premium/features" \
  -H "Authorization: Bearer $ACTIVE_TOKEN"

# 3. Expired trial / subscription
curl -s "https://TARGET.COM/api/premium/features" \
  -H "Authorization: Bearer $EXPIRED_TOKEN"

# 4. Suspended / Banned
curl -s "https://TARGET.COM/api/me" \
  -H "Authorization: Bearer $BANNED_TOKEN"

# 5. Deleted / Deactivated
curl -s "https://TARGET.COM/api/me" \
  -H "Authorization: Bearer $DELETED_TOKEN"

# 6. Password changed (old session still valid?)
curl -s "https://TARGET.COM/api/me" \
  -H "Authorization: Bearer $OLD_TOKEN"
# ^ If this returns 200, password change didn't invalidate sessions
```

### 6.2 Session Invalidation Testing

```bash
# Test what actions invalidate sessions:
# - Password change
curl -X PUT "https://TARGET.COM/api/users/me/password" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"old_password": "old!", "new_password": "new!"}'
# Then re-test old token
curl -s "https://TARGET.COM/api/me" \
  -H "Authorization: Bearer $TOKEN"
# Should be 401 if properly invalidated

# - Email change
curl -X PUT "https://TARGET.COM/api/users/me/email" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"email": "new@test.com"}'
# Old token should be invalidated

# - 2FA enrollment/enable
curl -X POST "https://TARGET.COM/api/2fa/enable" \
  -H "Authorization: Bearer $TOKEN"
# All sessions should be invalidated

# - Logout
curl -X POST "https://TARGET.COM/api/logout" \
  -H "Authorization: Bearer $TOKEN"
# Token should be invalidated
```

## 7. Identity Rotation & Rate Limit Evasion

### 7.1 Multi-Account Rotation

```python
# When rate limited, rotate through identity pool
IDENTITIES = [
    {"email": "user-a@a.com", "token": "TOKEN_A", "role": "user"},
    {"email": "admin-a@a.com", "token": "TOKEN_ADMIN", "role": "admin"},
    {"email": "user-b@b.com", "token": "TOKEN_B", "role": "user"},
    {"email": "user-c@a.com", "token": "TOKEN_C", "role": "user"},
]

import itertools
rotator = itertools.cycle(IDENTITIES)

for i in range(100):
    identity = next(rotator)
    r = requests.get("https://TARGET.COM/api/users",
                     headers={"Authorization": f"Bearer {identity['token']}"})
    if r.status_code == 429:
        print(f"[!] Rate limited on {identity['email']}, skipping")
        next(rotator)  # Skip to next identity
        time.sleep(5)
    else:
        print(f"[{identity['role']}] Request {i}: {r.status_code}")
```

## 8. Identity Ledger Maintenance

```bash
# Check all tokens for expiry
while IFS='|' read -r id email role tenant status token rest; do
  if [ "$status" = "active" ]; then
    TOKEN=$(echo "$token" | xargs)
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
      "https://TARGET.COM/api/me" \
      -H "Authorization: Bearer $TOKEN")
    
    if [ "$STATUS" = "401" ]; then
      echo "[!] Token expired for $id ($email) - needs refresh"
    fi
  fi
done < accounts.md
```

## 9. Provisioning Checklist

- [ ] User accounts on primary tenant (minimum 2)
- [ ] Admin accounts on primary tenant
- [ ] User account on secondary tenant
- [ ] OAuth/SSO linked account (if applicable)
- [ ] Expired/cancelled account
- [ ] Suspended/banned account
- [ ] Deleted account
- [ ] Accounts for each API key type
- [ ] Tokens extracted and logged in accounts.md
- [ ] Cross-tenant access tested
- [ ] Role escalation tested
- [ ] Lifecycle state access tested
- [ ] Token rotation/refresh tested
