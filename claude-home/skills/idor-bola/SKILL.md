---
name: idor-bola
description: Insecure Direct Object Reference / Broken Object Level Authorization testing. Use when endpoints reference object IDs (user, order, invoice, file, ticket, document), when numeric/UUID params look enumerable, or when testing multi-tenant data access.
---

# IDOR / BOLA

Goal: demonstrate that account A can read/modify account B's object. This is one of the
highest-paying bug classes — but ONLY when proven with real cross-object access.

## Detect
1. Map endpoints that carry an object identifier in URL path or body:
   `/api/users/123`, `/api/orders/{id}`, `?file_id=`, `?doc=`, `?invoice=`, `/download?id=`.
2. Identify the two accounts you control (or one account + a victim object id you can
   legitimately see in your own response, e.g. your own order id then change it).

## Test pattern (the core loop)
1. Request YOUR object: `/api/orders/1001` → 200 with your data. Save response.
2. Change id to a DIFFERENT value: `/api/orders/1002` → 
   - **BOLA:** 200 returning another user's data → PROVEN (capture it).
   - 403/404: test sequential/nearby ids, UUIDs you can observe elsewhere, alternate
     representation (decimal↔hex, base64 of id, UUIDv1 timestamps).
3. Cross-verify: confirm the returned record does not belong to your account.

## Variations
- **Path:** `/api/v1/orders/1001` vs `/api/orders/1001` vs `/orders/1001/items/`
- **Method:** GET vs POST vs PUT vs DELETE on the same id (write BOLA).
- **Mass assignment:** add fields like `ownerId`, `userId` to create/update requests.
- **Encoded/alternate id:** base64, URL-encoded, UUID with dashes stripped, v1/v2 UUID
  timestamp guess, numeric in hex.
- **Multi-tenant:** switch `tenant`/`org`/`company` headers or claim in body.
- **List endpoints:** `/api/orders` may leak all objects without an id at all (BOLA on list).

## Preconditions & honest severity
- Unauthenticated BOLA on sensitive data = critical.
- Authenticated BOLA (user→user) = high, but note the auth level required.
- Read vs write (modify/delete) changes severity — demonstrate the worst real one.

## Ruled out?
Require: numeric + UUID/opaque-id + path variation + one method change + list-endpoint check.

## Evidence
Capture the request for YOUR id (200) and the request for ANOTHER id returning their data
(200) — the pair proves the flaw. Sanitize PII before storing.