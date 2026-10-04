---
name: situational-decision
description: Moment-aware selection of tools/skills/features — fire exactly the right instrument when the current situation needs it, keep the rest idle. The "automatic but intelligent" decision layer for the whole hunt.
category: meta-orchestration
---

# Situational Decision — Fire What This Moment Needs

Do NOT run everything all the time. Do NOT blindly follow the stage checklist. Read the
current state, pick the ONE highest-value action, fire the exact tool/skill/feature it
calls for, then re-evaluate. Other instruments stay off until their moment arrives.

## The decision loop

1. **Read state** — `STATE.md` + event-log tail: what's open, blocked, confirmed.
2. **Score candidate moves** — expected_impact × likelihood × uniqueness − cost.
3. **Fire the one tool this moment needs** (router for skills, dispatch for agents).
4. **Re-read state, repeat.**

## Situational trigger map — what fires when

| When the MOMENT is... | Fire... |
|---|---|
| New target / surface unknown | recon-agents + recon-methodology, subdomain/endpoint mining |
| HTTP request needed (ad-hoc, any bytes) | Burp MCP raw send |
| Captured traffic / history mining | Caido MCP (list_requests, sitemap, replay) |
| Endpoint surfaced → test class | skill-router → exact skill (ideor/ssrf/sqli/xss...) |
| Anomaly / odd response in a flow | Jump the queue: investigate NOW (chase it live) |
| ID-bearing endpoint appears | idor-bola immediately |
| Auth / password reset / MFA in view | auth-bypass, password-reset, mfa-2fa-bypass |
| Money / state / workflow / race surface | bizlogic, race-condition-testing |
| JSON/GraphQL/API discovered | api-testing, graphql-attacks |
| CORS / headers / CSP seen in response | cors-misconfig, headers-audit |
| Finding confirmed | mandatory-triage-gate + adversarial-verification NOW |
| Stalled/blocked a technique (3+ methods fail) | rotate technique / bypass — not a stop |
| Recon/endpoint data stale (> minutes) | refresh that feed only, don't rerun all |
| Many independent surfaces, all low-dependency | spawn PARALLEL agents (agent-dispatch) — independent work runs concurrently |
| One deep interdependent chain / shared state | stay SINGLE-THREADED (sequential) — parallel agents would race/conflict |
| Time-sensitive intensive task, small scope | parallel fan-out (intruder/fuzz bursts, multi-endpoint sweep) |
| WAF / TLS-fingerprint / rate-based block | rotate IP / rotate technique (curl-impersonate, proxy) — not a stop |
| Rate-limit or IP-throttle observed | polite pacing / IP rotation + spread requests |
| Target behind Cloudflare / anti-bot | browser-control (real-browser extension, pass fingerprinting), lightweight is not enough |
| Auth-gated surface reached | mint/load the test account(s) NOW (disposable-email if OTP needed) |
| Suspected blind vuln (blind SQLi/SSRF/XXE) | OOB channel (Collaborator/burp_generate_collaborator_payload) |
| Timing/differential needed (boolean-blind, race) | timing tooling / repeated differential requests |
| JWT / session / OAuth token in traffic | jwt-attacks / session-attacks / oauth-abuse now |
| Auth token expired / 401 mid-flow | refresh token / re-auth the session, don't retry blindly |
| Same finding pattern at many endpoints | automate the sweep (param-fuzz × list) then verify hits |
| User watching / wants to learn | live-hunt-mentor narration on top of everything |
| Budget / termination / scope call | pause ONLY then (the 3 gates) |

## Cost-aware selection

- Prefer the lightest instrument that answers the immediate question (a curl beats a
  browser fleet; a single Repeater send beats Intruder).
- Escalate to heavy instruments (browser fleet, parallel agents, interactive auth, wide
  fuzzing) ONLY when the moment genuinely demands it (auth-gated surface, CF-anti-bot,
  deep unique scenario).
- Idle = correct when the moment doesn't need it. Do not pad activity.

## Guardrails
- Selection is driven by live state, not by "coverage for coverage's sake."
- Never fire a tool without a reason tied to the current state; never withhold one the
  current state demands.
- Keep proof-of-impact + evidence discipline: what you fire must still produce/confirm
  real, verifiable results.
