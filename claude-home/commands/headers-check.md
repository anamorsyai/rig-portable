---
description: Check for common security misconfigurations and missing headers.
---

Security headers and misconfiguration check on: $ARGUMENTS

## Security Headers Check
```bash
curl -sI "https://$ARGUMENTS" | grep -iE "(strict-transport|content-security|x-frame|x-content-type|x-xss|referrer-policy|permissions-policy|feature-policy|server|x-powered|access-control)"
```

## Checklist

### Missing Headers
- [ ] `Strict-Transport-Security` (HSTS)
- [ ] `Content-Security-Policy` (CSP)
- [ ] `X-Frame-Options` or `frame-ancestors` CSP
- [ ] `X-Content-Type-Options: nosniff`
- [ ] `Referrer-Policy`
- [ ] `Permissions-Policy`

### Misconfigurations
- [ ] Server version disclosure (`Server` header)
- [ ] X-Powered-By disclosure
- [ ] Verbose error pages
- [ ] Directory listing enabled
- [ ] Default credentials on admin panels
- [ ] Debug endpoints exposed (`/debug`, `/trace`, `/actuator`)
- [ ] Backup files (`/.git/`, `.env`, `.bak`)
- [ ] CORS misconfiguration (origin reflection, null origin)

### Cookie Security
- [ ] `HttpOnly` flag
- [ ] `Secure` flag
- [ ] `SameSite` attribute
- [ ] Session token in URL

### TLS/SSL
```bash
# Quick TLS check
echo | openssl s_client -connect $ARGUMENTS:443 2>/dev/null | openssl x509 -noout -dates -subject -issuer
```

## CORS Testing
```bash
# Test CORS with different origins
curl -sI -H "Origin: https://evil.com" "https://$ARGUMENTS" | grep -i "access-control"
curl -sI -H "Origin: null" "https://$ARGUMENTS" | grep -i "access-control"
curl -sI -H "Origin: https://$ARGUMENTS.evil.com" "https://$ARGUMENTS" | grep -i "access-control"
```

Provide a severity-rated list of all findings with exploitation potential.
