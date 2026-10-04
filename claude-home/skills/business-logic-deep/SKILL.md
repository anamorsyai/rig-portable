---
name: business-logic-deep
description: "Deep business logic abuse: multi-step workflow exploitation, state machine attacks, conditional logic bypass."
---

# Deep Business Logic Abuse

## Multi-Step Workflow Abuse
- Identify all state transitions (signup, payment, approval, etc.)
- Call steps out of order (step 3 without steps 1-2)
- Skip mandatory steps (skip verification, skip approval)
- Replay completed steps for duplicate effect

## State Machine Attacks
- Map the state machine (e.g., draft->pending->approved->active)
- Try to jump states (draft->active directly)
- Try to reverse states (active->draft)
- Try to corrupt state (partial update leaves inconsistent state)

## Conditional Logic Bypass
- Negative quantities (buy -1 items = refund)
- Zero-price items (bypass payment)
- Extreme values (overflow, integer underflow)
- Invalid state combinations (cancel already-shipped order)

## Race on State Transitions
- Parallel requests to change state simultaneously
- TOCTOU: check state, then act based on stale state
- Double-submit: send same state change twice
- Concurrent coupon redemption

## Financial Logic Abuse
- Discount stacking (combine multiple discounts)
- Coupon reuse (single-use coupon used multiple times)
- Price manipulation via currency/region
- Tax calculation bypass
- Shipping cost manipulation
- Refund amount exceeds original payment

## Business Logic Testing Checklist
- [ ] Map complete user journey/workflow
- [ ] Test every state transition
- [ ] Test out-of-order step execution
- [ ] Test with invalid/missing steps
- [ ] Test financial calculations with edge values
- [ ] Test concurrent operations
- [ ] Test role transitions (user->admin)
- [ ] Test boundary conditions (min/max/zero/negative)
- [ ] Test with expired/invalid tokens mid-flow
- [ ] Test interruption/resumption of multi-step flow

## Impact Assessment
- Financial: direct monetary loss
- Data: unauthorized data access
- Availability: service disruption
- Integrity: data corruption/modification
- Compliance: regulatory violations
