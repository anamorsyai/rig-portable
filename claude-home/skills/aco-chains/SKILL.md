---
name: aco-chains
description: Account creation and ownership chains (ACO) — mass account creation to abuse limitations, email verification bypass, disposable-email signups, account-merging confusion, and ownership confusion across user pools. Use when the app has signup quotas, free-tier limits, referral rewards, or multi-tenant account switching.
category: business-logic
---

# Account Creation & Ownership Chains

## Detection
- Signup flows with quotas: free-tier limits, referral bonuses, trial credit, per-account data caps.
- Account switch/merge features: can attacker link accounts, rename, transfer ownership?
- Email verification: is it enforced server-side (rate limit on resend, unique address, disposable blocklist)?

## Exploitation
1. **Mass account creation**: automated signup with rotating emails (authorized only) → bypass free-tier / trial limits.
2. **Email verification bypass**: sign up with a victim's email, request resend (`account_verify?email=victim`), or `verified=true` in signup JSON.
3. **Disposable email bypass**: register with `+` aliases / new domains not blocked → unlimited referrals.
4. **Ownership confusion**: transfer resource to another account via `ownerId` tamper; merge accounts to steal data; switch tenant with `X-Tenant-Id` header.
5. **Invite-chain abuse**: invite token predictable (`invite?code=<seq>`) → invite victims/self repeatedly for rewards.

## Payloads
```
POST /signup {"email":"a@x.com","verified":true,"referrer":"self"}
POST /account/switch {"tenantId":"victim-tenant"}
POST /resource/transfer {"ownerId":"<another-user>"}
```

## Tool Commands
```powershell
# sequential-signup loop (authorized only, rate-limited)
for ($i=1; $i -le 20; $i++) { curl.exe -s -X POST "$U/signup" -d "email=user$i@x.com&ref=ME" }
# tenant switch probe
curl.exe -s -X POST "$U/account/switch" -H 'X-Tenant-Id: victim' -b sess
```

## Verification & Evidence
- Show limit bypass / ownership takeover / verification skip with concrete steps.
- Do not actually harm accounts; use throwaway data where possible.