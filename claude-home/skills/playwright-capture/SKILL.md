---
name: playwright-capture
description: Headless browser capture via Playwright â€” all HTTPS traffic saved to requests.jsonl for agent analysis. Use when testing web UIs, login flows, JS-heavy pages, or any HTTPS endpoint that needs browser rendering.
---

# Playwright Capture Pipeline

All headless browser traffic flows: **Playwright â†’ requests.jsonl â†’ agents analyze**.

## Quick capture

```bash
# Browse a URL â€” captures all requests/responses
oc-playwright launch https://target.com

# Run custom script with capture
oc-playwright run script.js
```

## What gets captured

Every HTTP request and response is saved to `$HOME/hunting-rig-data/default/requests.jsonl`:
- Method, URL, host, path
- Request headers and body
- Response status, headers, body (base64)
- Protocol (https-playwright)
- Source (playwright)

## Read captured traffic

```bash
# Latest 10 requests
tail -10 ~/hunting-rig-data/default/requests.jsonl | python3 -c "
import sys, json
for line in sys.stdin:
    r = json.loads(line)
    print(f\"{r['method']} {r['url']} â†’ {r.get('responseStatus')} ({r.get('protocol','?')})\")"

# Filter by host
grep '"host":"target.com"' ~/hunting-rig-data/default/requests.jsonl | tail -5

# Get full request/response
tail -1 ~/hunting-rig-data/default/requests.jsonl | python3 -m json.tool
```

## Agent workflow

### @map â€” Browser-based endpoint discovery
```bash
# 1. Capture JS-heavy page
oc-playwright launch https://target.com/dashboard

# 2. Read captured traffic for endpoints
grep '"url"' ~/hunting-rig-data/default/requests.jsonl | \
  python3 -c "import sys,json; [print(json.loads(l)['url']) for l in sys.stdin]" | \
  sort -u > endpoints-from-browser.txt

# 3. Feed to @vuln for testing
```

### @vuln â€” Browser-based vulnerability testing
```bash
# 1. Test login flow
oc-playwright launch https://target.com/login

# 2. Check captured requests for:
# - Auth tokens in headers/cookies
# - Session management patterns
# - API endpoints called by JS
# - Interesting parameters

# 3. Replay interesting requests via Burp MCP
```

### @bac â€” Live manual testing
```bash
# 1. Browse sensitive areas
oc-playwright launch https://target.com/admin
oc-playwright launch https://target.com/api/v1/users

# 2. Analyze captured traffic for:
# - IDOR patterns (sequential IDs in URLs)
# - Missing auth checks
# - Verbose error messages
# - Data exposure in responses
```

### @intake â€” Policy page extraction
```bash
# 1. Capture program policy page
oc-playwright launch https://target.com/responsible-disclosure

# 2. Extract scope from captured HTML
grep 'responseBody' ~/hunting-rig-data/default/requests.jsonl | tail -1 | \
  python3 -c "import sys,json; print(json.loads(sys.stdin.readline()).get('responseBody','')[:5000])"
```

## Playwright script template

```javascript
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({
    headless: true,
    args: [
      '--no-proxy-server',
      '--ignore-certificate-errors',
      '--dns-server=8.8.8.8'
    ]
  });

  const context = await browser.newContext({
    ignoreHTTPSErrors: true,
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
  });

  const page = await context.newPage();

  // Capture all traffic (auto-saved by oc-playwright wrapper)
  page.on('request', req => {
    console.log(`â†’ ${req.method()} ${req.url()}`);
  });

  page.on('response', res => {
    console.log(`â† ${res.status()} ${res.url()}`);
  });

  await page.goto('https://target.com', { waitUntil: 'networkidle', timeout: 30000 });

  // Interact with page
  await page.fill('input[name="username"]', 'testuser');
  await page.fill('input[name="password"]', 'testpass');
  await page.click('button[type="submit"]');
  await page.waitForLoadState('networkidle');

  console.log('Title:', await page.title());
  console.log('URL:', page.url());

  await browser.close();
})();
```
