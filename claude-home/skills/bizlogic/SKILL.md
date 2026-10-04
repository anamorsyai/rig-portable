---
name: bizlogic
description: Business logic and race condition testing. Use when probing workflow flaws, pricing/quantity/currency manipulation, coupon/gift abuse, transfer/balance endpoints, or any stateful multi-step flow.
---

# Business Logic + Race Conditions

Goal: prove a state-machine / monetary flaw that a triager will pay for. These are
program-specific, so spend time understanding the app's rules first.

## Understand the rules
Map the real business flow: register → cart → checkout → payment → fulfillment; or
transfer → balance → withdrawal. Read the JS/API to find client-side-only validations
(those are the ones worth bypassing).

## Classic bizlogic flaws
- **Pricing/quantity:** negative quantity, `qty=-1`, decimal `0.01`, change `price`/
  `total`/`currency` fields the client sends; integer overflow (send huge value).
- **Coupons/gifts:** reuse a used coupon (race or replay), apply coupon to wrong item,
  stack unlimited coupons, self-referral (gift code to own second account).
- **Currency:** change `currency` between steps → exchange-rate arbitrage; pay in cheap
  currency.
- **State machine:** skip steps (checkout without payment, confirm order twice), replay a
  response, tamper `step`/`status` params.
- **Verification bypass:** change `emailVerified`/`phoneVerified` flags; complete "verify"
  without actually verifying; use a resend endpoint to bypass the check.
- **Limits:** withdraw more than balance by parallel requests; bypass per-IP/per-user rate
  or daily limits by header (X-Forwarded-For) or by multiple accounts.
- **IDOR-in-bizlogic:** apply a discount/promo code that belongs to another tenant.

## Race conditions (the money-maker)
Target state-changing endpoints with a "check-then-act" pattern (balance check, coupon
validation, one-time token):
```bash
# Fire N concurrent identical requests at the same moment
for i in $(seq 1 20); do curl -s -X POST 'https://TARGET/api/apply-coupon' \
  -H 'Cookie: SESSION=...' -d 'code=WELCOME10' & done; wait
```
- **Coupon reuse / double spend:** N requests all return success → used once but granted N
  times → money or free goods.
- **Balance/token race:** withdraw/settle twice with the same request → double credit.
- **Rate-limit race:** one-time verify/OTP accepted by racing submissions.
Confirm by checking the resulting balances/orders in the account.

## Honest severity
Monetary loss to the company or free goods = real. "This looks odd" without demonstrated
loss is a lead, not a finding.

## Evidence
Capture the N racing requests and the resulting state (balance before/after, coupon usage
count). Timing evidence matters — include the concurrency proof (identical timestamps or
interleaved logs).