---
name: live-teaching
description: Explains what a specific test/check/finding means in plain language for someone learning bug bounty hunting â€” what was checked, why it's safe as observed, and the exact condition that would flip it into a real vulnerability. Used ONLY by @teacher, never by the actual hunting agents â€” this is a read-only companion, not a hunting participant.
---

# Live Teaching â€” Explain What Just Happened, Not What To Do

## Fixed explanation format (use this exact structure every time)

For any check, test, or finding pulled from the project's logs/files, answer
these four things, in this order, in plain language:

1. **What was checked** â€” the concrete action, in one sentence a beginner
   understands (not "tested for IDOR" â€” "tried loading another user's
   invoice by changing the number in the URL from 8841 to 8842").
2. **Why it's safe here** (if it is) â€” the specific mechanism that blocked
   it. Not "the server validated it" â€” name the actual check: "the server
   compared the invoice's owner_id against the logged-in session's user_id
   and returned 403 because they didn't match."
3. **What would flip it to a real vulnerability** â€” the exact missing
   condition, concretely: "if that owner_id check didn't happen â€” for
   example if the endpoint just trusted the ID in the URL â€” you'd see the
   other user's real invoice data instead of a 403. That's the difference
   between 'safe' and an IDOR."
4. **Where this fits in the bigger picture** â€” one sentence connecting it
   to the broader methodology step it belongs to (e.g. "this is Step 2 of
   the T1-T3 checklist for this endpoint â€” access-control testing on
   every ID-based endpoint, not just this one").

## Worked example

Pulled from `$PROJECT/vuln/open-leads.md`: `idor-expert` tested
`/api/invoices/{id}` and got a 403 on a cross-account request.

> **What was checked:** whether logging in as one account and requesting
> another account's invoice by guessing its ID would return that other
> account's data.
> **Why it's safe here:** the response was `403 Forbidden`, and the raw
> request/response log shows the server explicitly checked the invoice's
> owner field against the requester's session before responding â€” access
> control is enforced server-side, not just hidden by the UI.
> **What would flip it:** if that same request had returned `200` with the
> other account's real invoice JSON instead of `403`, that's a live IDOR â€”
> exactly the kind of finding in `mandatory-triage-gate`'s worked example.
> The only difference between this safe result and a paid bounty is which
> HTTP status code came back.
> **Bigger picture:** this is routine IDOR coverage from the T1-T3
> checklist â€” every ID-bearing endpoint gets this same test, this is just
> one of many.

## What NOT to do
- Don't narrate every single HTTP request â€” only checks/findings that
  actually resolved to something (a pass, a fail, a confirmed finding, a
  rejected one). Pulling from `oc-checkpoint`'s finding/error log entries
  is the right granularity, not raw traffic.
- Don't speculate about findings that haven't been verified by `@exploit`
  yet â€” explain confirmed results and clearly-ruled-out results, and say
  "still being tested" for anything mid-flight rather than guessing at its
  outcome.
- Don't slow anything down. This skill is read-only against files the
  hunting agents already wrote â€” it never sends a request to the target,
  never calls `@exploit`/`@vuln` for more detail, and never blocks on
  anything still in progress.
