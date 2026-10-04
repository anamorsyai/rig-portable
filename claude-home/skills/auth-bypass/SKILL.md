---
name: auth-bypass
description: Authentication and authorization bypass testing. Use when probing login, registration, password reset, OAuth/SSO flows, session tokens, JWTs, or any account-level control.
---

# Auth Bypass

Goal: demonstrate unauthorized access — login bypass, session/forgery, token tampering,
privilege escalation, or account takeover. ATO is a top-tier bounty.

## Login / auth logic
- **Rate limit / no limit:** brute-force a weak-known credential only if clearly authorized;
  otherwise test enumeration (different error for existing vs non-existing user).
- **Parameter tampering:** `user=admin&pass=admin` variants, mass-assign `role`, `isAdmin`,
  `emailVerified`, `level` in register/update; HTTP verb confusion (GET for POST action).
- **JWT/Token:** change algorithm to `none`; swap RS256→HS256 and sign with the public key
  (the infamous alg-confusion); check `kid` path injection; test expiry/`nbf`/`jti` fields.
- **Session:** predictable session id (sequential, timestamp, base64 of username); session
  fixation (pre-set cookie, login does not rotate); cookie not invalidated on logout.

## Password reset (high-value ATO vector)
- **Host header poisoning:** `Host: attacker.com` — reset links emailed may include attacker
  domain.
- **Token in response / predictable token** (numeric, time-based, short entropy).
- **User-supplied email vs token mismatch:** reset flow may confirm a token for ANY user if
  `user`/`email` param is attacker-controlled.
- **POST body user switch:** change `username` between step 1 (victim) and step 2 (attacker)
  while reusing the victim's token.

## OAuth / SSO
- **state param missing/not validated** → login CSRF / session swap.
- **redirect_uri not strictly validated** (prefix/suffix match bypass: `https://app.com.attacker.com`)
- **token leakage** in `redirect_uri` fragment/query of attacker page; authorization code
  reuse (no one-time enforcement).

## Privilege escalation
- After login as low user, hit admin-only endpoints directly (no client-side-only checks).
- Change `role`/`org`/`tenant` in request body/headers; test with a low user's cookie on a
  high-privilege endpoint.

## Honest severity
Distinguish: auth bypass (no creds) = critical; user→admin = high; reset-ATO = critical if
proven end-to-end. Prove the exact impact (which victim account, what access).

## Ruled out?
Require: login tampering + JWT/alg check (if JWT) + reset-token check + one OAuth
redirect_uri check before ruling out.

## Evidence
Capture the full multi-step flow with raw requests/responses proving the bypass (e.g.
reset token received and used on victim). No real passwords or user data in notes.