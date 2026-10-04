---
name: xss
description: Cross-site scripting testing (reflected, stored, DOM, blind). Use when user input is reflected in responses, rendered in client-side JS, persisted, or when testing name/comment/URL/profile fields.
---

# XSS

Goal: prove script execution (or an XSS-capable primitive) end-to-end. Stored/blind XSS in
a privileged context is high value; reflected behind CSP may be noise — prove execution.

## Detect reflection
1. Send a unique marker `xssPROBE123abc` in each parameter/body field.
2. Look for it echoed: where (HTML, attribute, script block, JS string, JSON), sanitized or raw.
3. Inspect the exact reflection context — this determines the payload.

## Context-based payloads
- **HTML body:** `<img src=x onerror=alert(1)>`
- **Attribute:** `" onmouseover="alert(1)` or `"><svg/onload=alert(1)>`
- **JS string:** `';alert(1);//`
- **script block:** `</script><script>alert(1)</script>`
- **JSON context:** `\u003cscript\u003e` / `{"a":"\u003cimg src=x onerror=alert(1)\u003e"}`

## Filter bypass rotation (test until one works or all fail)
- Case, tag substitution (svg/body/img/video/details), attribute events (onerror/onload/onfocus)
- Encodings: `&#x3C;`, `%3C`, `\u003c`, HTML entities, `javascript:` variants
- Breakouts: newlines/tabs inside tags, backticks, nested `<svg><script>`
- CSP note: if CSP blocks inline, look for DOM sinks (see below) or allowed gadgets

## DOM XSS (no server reflection — use the browser)
- Identify sinks in JS: `innerHTML`, `document.write`, `eval`, `location`, `setTimeout`,
  `insertAdjacentHTML`, `outerHTML`, `URL`/`location.search` sources.
- Chrome DevTools / playwright: set URL param, observe DOM execution.
- Payload for hash/search based: `#<img src=x onerror=alert(1)>`

## Blind XSS
- Inject a Collaborator-bearing payload into stored fields (feedback, name, ticket, admin
  notes) and watch `get_collaborator_interactions` for a hit.

## Ruled out?
Require: HTML-context + attribute + JS-string + one encoded variant + DOM sink check.

## Evidence
Capture raw request + response showing reflection (or the DOM sink for DOM XSS). For
stored/blind, capture the storage request + the poisoned render. Prove impact (alert, cookie
read, admin-context hit) rather than just reflection.