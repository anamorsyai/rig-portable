---
description: Analyze JavaScript files for secrets, endpoints, and hidden attack surface.
---

JavaScript file analysis for: $ARGUMENTS

## Download and analyze

```bash
# Fetch JS files
curl -s "$ARGUMENTS" > /tmp/js_target.js

# Find all JS files in page
curl -s "$ARGUMENTS" | grep -oP 'src="[^"]*\.js[^"]*"' | sed 's/src="//;s/"//' | sort -u > /tmp/js_files.txt
```

## Check for

### Hardcoded Secrets
- API keys: `sk_live_`, `ak_`, `AKIA`, `AIza`
- Tokens: `token`, `apikey`, `api_key`, `secret`
- Passwords: `password`, `passwd`, `pwd`
- Private keys: `BEGIN RSA PRIVATE KEY`, `BEGIN PRIVATE KEY`
- Connection strings: `mongodb://`, `postgres://`, `mysql://`

### Hidden Endpoints
- API base URLs: `/api/v1/`, `/api/v2/`
- Internal endpoints: `/admin`, `/internal`, `/debug`
- GraphQL schemas: `query`, `mutation`, `__schema`
- WebSocket URLs: `wss://`, `ws://`

### Sensitive Functionality
- Authentication bypass logic
- Admin-only functions
- File upload handlers
- Payment/testing bypasses (e.g., `testMode`, `debug`)

### Security Misconfigurations
- Verbose error handling
- Commented-out code with sensitive info
- Environment-specific URLs
- Feature flags and toggles

Use `grep`, `jq`, and regex to extract findings. Provide a categorized list with file references.
