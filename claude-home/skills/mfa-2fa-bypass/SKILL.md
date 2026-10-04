---
name: mfa-2fa-bypass
description: MFA/2FA bypass techniques — OTP brute-force, response manipulation, OTP in response body, race conditions, session/token reuse, backup-code abuse, timing side-channels, and 2FA enforcement gaps. Use when an app requires SMS/TOTP/email 2FA on login or sensitive actions.
category: authn-authz
---

# MFA / 2FA Bypass

## Detection
- Login flow: password → OTP step. Test: is OTP length/entropy small (4-6 digits)? Is there rate limiting on verify? Is OTP returned anywhere in the response chain?
- Sensitive actions re-authenticated with OTP: is the gate server-side enforceable (can it be skipped by calling the post-OTP endpoint directly)?

## Exploitation
1. **Response leak**: OTP/OTP reference embedded in response body, Set-Cookie, or debug headers during the send-OTP request.
2. **Brute force**: 6-digit OTP without rate limit or lockout → iterate 000000–999999 (use ffuf/Intruder).
3. **Race condition**: request send-OTP and verify in parallel with a guessed code; re-issue token race (same as password-reset race).
4. **Token reuse**: OTP codes or MFA tokens not invalidated after use; reuse the same code across sessions.
5. **Enforcement bypass**: directly call `/api/session/confirm-mfa` with session cookie, skip the check entirely; or login with `mfa=false`, `skipMfa=true` params.
6. **Backup codes**: 8-digit codes, many issued, no rate limit, reusable.
7. **Session not invalidated**: MFA verified once, then `/logout` + re-login without MFA re-verification (2FA not re-required).
8. **QR/TOTP secret exposure**: `/api/mfa/secret` leaks the shared secret.

## Payloads
```
POST /api/otp/verify  {"code":"000000"} .. {"code":"999999"}
POST /api/login {"user":"a","pass":"b","mfa":"skip"}
GET  /api/session/confirm-mfa  (unauthenticated gate)
```

## Tool Commands (Windows)
```powershell
# OTP brute force (only if no rate limiting / authorized)
# ffuf -w <(seq -w 0 9999) -X POST -u "$U/api/otp/verify" -d '{"code":"FUZZ"}' -H 'Content-Type: application/json' -mc 200
# race send+verify (requires parallel execution)
# Start-Job { curl.exe -s "$U/api/otp/send" -X POST -d '{"resend":"true"}' }
# curl.exe -s "$U/api/otp/verify" -X POST -d '{"code":"111111"}'
```

## Verification & Evidence
- Prove bypass = completed authentication or sensitive action WITHOUT a valid OTP (or with brute-forced/reused one).
- Record: rate-limit absence, code entropy, exact reproduction.