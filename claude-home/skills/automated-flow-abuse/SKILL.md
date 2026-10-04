---
name: automated-flow-abuse
description: "Automated flow abuse — server-side automation of human-gated features: CAPTCHA, email/SMS OTP loops, report spam, notification bombing, rate-limit-free bulk actions. Use when the app has flow gates designed to be human-only."
category: business-logic
---

# Automated Flow Abuse

## Detection
- Features with human gates: CAPTCHA, one-per-IP signups, OTP resend limits, report/contact forms, referral invites, notification triggers, review submissions.
- Test: is the gate actually enforced server-side, or just in the UI (no CAPTCHA token validation, no rate limit on the API)?

## Exploitation
1. **CAPTCHA bypass**: token not validated; reusable token; token for image only; call the API directly without the CAPTCHA step.
2. **OTP/SMS spam**: unlimited resend endpoint → SMS/email bombing (not a finding on its own — needs real impact: quota abuse, cost to provider, mailbox DoS).
3. **Bulk report/notification**: script N submissions in a loop → spam team/abuse systems; report spam = DoS if it floods inboxes.
4. **Referral/invite flood**: generate unlimited invites to exhaust quota or farm rewards.
5. **Rate-limit absence on state-changing flow**: unlimited password-reset emails, unlimited account verification.

## Payloads
```
POST /contact {"msg":"spam", "captcha_token":"REUSED"}
POST /otp/send {"phone":"+1..."}   # loop
POST /invite {"email":"x@y.com"}   # loop
```

## Tool Commands
```powershell
# demonstrate no CAPTCHA enforcement
curl.exe -s -X POST "$U/contact" -d '{"msg":"test"}' -i   # no captcha field needed?
# loop invite (limited to proof of N=5)
for ($i=1; $i -le 5; $i++) { curl.exe -s -X POST "$U/invite" -d "email=u$i@x.com" }
```

## Verification & Evidence
- Proof = gate bypassed (CAPTCHA skippable) or bulk action possible beyond human limits (N consecutive ops).
- Quantify impact (cost/spam/DoS); keep loops small.