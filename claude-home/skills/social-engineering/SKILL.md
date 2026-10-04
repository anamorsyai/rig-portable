---
name: social-engineering
description: "Social engineering: OAuth consent abuse, phishing payloads, OAuth state manipulation."
---

# Social Engineering (In-Scope Only)

## OAuth Consent Abuse
- Register malicious app with broad scopes
- Trick user into authorizing
- Use authorization code to access victim account
- Refresh token for persistent access

## OAuth State Manipulation
- Remove state parameter = CSRF on OAuth
- Predictable state values
- State parameter fixation

## Phishing Payloads
- Craft convincing login pages
- OAuth redirect URI manipulation
- Password reset link interception
- Session cookie theft via phishing

## Trust Exploitation
- Impersonate legitimate service emails
- Use legitimate domains for credibility
- Leverage existing trust relationships
- Abuse email forwarding rules

## Account Recovery Abuse
- Password reset flow manipulation
- Security question bypass
- Backup code enumeration
- Recovery email/phone interception
