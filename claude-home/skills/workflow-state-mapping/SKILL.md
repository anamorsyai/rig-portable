---
name: workflow-state-mapping
description: Forces an explicit state-machine artifact of the business workflow BEFORE probing it for logic flaws, instead of ad-hoc poking. Use this as the first step of any @bizlogic engagement — the mapping IS the deliverable that turns "senior-level intuition" into a repeatable procedure a smaller model can execute reliably.
---

# Workflow State Mapping — Map First, Then Break

## Why mapping first matters more than the testing itself

A senior red-teamer's real edge on business logic isn't cleverer payloads —
it's that they draw the actual state machine before touching anything, so
every illegal transition is *visible* rather than something they have to
notice by accident while clicking around. This skill makes that mapping
step mandatory and gives it a fixed output shape, so a smaller model does
the systematic version of what an experienced human does by habit.

## Step 1 — Build the state table (mandatory artifact, write it before testing)

For the workflow under test (checkout, onboarding, approval chain, refund,
whatever the target's core money/permission flow is), produce exactly this:

```json
{
  "workflow": "checkout",
  "states": ["cart", "address_entered", "payment_pending", "paid", "shipped", "refunded", "cancelled"],
  "transitions": [
    {"from": "cart", "to": "address_entered", "endpoint": "POST /checkout/address", "guard": "cart non-empty"},
    {"from": "address_entered", "to": "payment_pending", "endpoint": "POST /checkout/payment-intent", "guard": "address valid"},
    {"from": "payment_pending", "to": "paid", "endpoint": "POST /webhook/payment-confirmed", "guard": "payment processor callback signature valid"},
    {"from": "paid", "to": "shipped", "endpoint": "POST /admin/orders/{id}/ship", "guard": "role=admin/warehouse"},
    {"from": "paid", "to": "refunded", "endpoint": "POST /orders/{id}/refund", "guard": "role=admin, order.status=paid"},
    {"from": "any", "to": "cancelled", "endpoint": "POST /orders/{id}/cancel", "guard": "order.status != shipped"}
  ],
  "server_side_only_transitions": ["payment_pending -> paid"],
  "client_reachable_transitions": ["cart -> address_entered", "address_entered -> payment_pending", "paid -> refunded (admin UI)", "any -> cancelled"]
}
```

The two fields that matter most for finding bugs:
- **`server_side_only_transitions`** — states that SHOULD only ever change
  via a trusted backend event (payment webhook, admin action). If you can
  reach that transition from a client-facing endpoint at all, that's
  already the finding, before you've even tried an attack payload.
- **`guard`** on every transition — write the guard the app SHOULD be
  enforcing. Every guard you wrote down is now a specific thing to test
  removing/bypassing, instead of a vague "check for logic flaws."

## Step 2 — Attack the table, not the app

For every transition in the table, run this fixed checklist — this
replaces open-ended poking with a closed set of concrete tests:

1. **Skip**: can you reach `to` directly from a state that isn't `from`?
   (e.g. `cart -> paid` skipping address/payment entirely)
2. **Reverse**: can you go from `to` back to `from`, or to an earlier state,
   after the guard has already been satisfied once? (e.g. `refunded -> paid`
   to double-claim a refunded order)
3. **Replay**: can the same request that caused `from -> to` be sent again
   to fire a second, unintended effect? (double refund, double credit)
4. **Race**: can two transitions that share a guard on the same resource
   be fired concurrently before either one's guard check completes?
5. **Client-reachable server-only**: for anything tagged
   `server_side_only_transitions`, can you hit that transition's endpoint
   directly, bypassing the trusted trigger (forge/replay the webhook, call
   the admin endpoint with a regular-user role, etc.)?
6. **Guard substitution**: is the guard checked using client-supplied data
   that you control (e.g. `order.status` read from a hidden form field
   instead of re-fetched server-side)?

## Worked example — where mapping catches what poking misses

Un-mapped testing on the example above would likely test the obvious things
(negative quantity, price override) and stop. The mapped version surfaces:

- `payment_pending -> paid` is marked `server_side_only`. Check 5 asks:
  can `POST /webhook/payment-confirmed` be called directly by an
  authenticated user, not just by the payment processor? If the endpoint
  only validates a signature and that signature check can be skipped,
  spoofed, or is missing on a related endpoint (`/webhook/payment-confirmed-v2`
  found during recon) — that's a **direct free-order chain**, worth more
  than any price-manipulation finding, and it was found by reading the
  table, not by guessing to try it.
- `paid -> refunded` guard is `order.status=paid`. Check 6 asks where
  `order.status` is read from. If the refund endpoint reads status from
  the request body instead of re-querying the DB, an order that's already
  `refunded` or `cancelled` can be refunded again by just resending
  `status: "paid"` in the body — **check 6 turned a vague "test edge
  cases" instruction into one specific request to try.**

## Output contract

Write `$PROJECT/bizlogic/<workflow-name>-state-map.json` (Step 1 shape)
before any testing begins, and log each Step-2 check result (pass/fail per
transition) to `$PROJECT/bizlogic/<workflow-name>-checklist.json`. A
bizlogic report should reference which specific table row and which of the
6 checks it came from — this is also what `@skeptic`'s realism gate uses to
confirm the finding wasn't a lucky guess but a systematic result.
