---
name: skill-router
description: Routes task intent and surface type to the right technique skills from SKILLS.INDEX — the primary selection mechanism before any test is executed.
category: meta-orchestration
---

# Skill Router

> Maps `(surface × technique-intent)` → concrete skills. Load BEFORE dispatching a test.
> Registry: `SKILLS.INDEX.md` (human) / `SKILLS.INDEX.json` (machine).

## Surface → skill-class map
| Surface | Skills to route to |
|---|---|
| unauthenticated web | api-fuzzing, parameter-fuzzing, directory-fuzzing, wayback-machine-mining, exposed-git-svn, backup-exposure |
| authenticated web | broken-access-control, idor-expert, mass-assignment, session-attacks, auth-bypass-session, multi-tenant-accounts |
| API (REST/JSON) | graphql-attacks, grpc, api-fuzzing, api-testing, verb-tampering, batching-dos, bfla |
| API (GraphQL) | graphql-attacks, introspection-based tests (field-level authz) |
| injection sinks | sqli-expert, sqli-techniques, nosql-injection, ldap-injection, xpath-injection, command-injection, ssti, template-injection, el-injection |
| XSS / client-side | xss-exploitation, xss-expert, dom-clobbering, postmessage, csp-bypass, prototype-pollution |
| SSRF / file fetch | ssrf-exploitation, dns-rebinding, xxe-expert, xxe-injection |
| auth flows | oauth-abuse, saml-sso, jwt-attacks, mfa-2fa-bypass, password-reset, password-reset-poisoning |
| business logic | business-logic, business-logic-deep, workflow-state-mapping, coupon-promo-abuse, price-payment-tampering, automated-flow-abuse |
| race / concurrency | race-condition-testing, race-conditions |
| cloud / infra | cloud-iam, cloud-bucket-enumeration, serverless-lambda, k8s-docker, ci-cd-pipeline, dependency-confusion, supply-chain-attacks |
| config / misconfig | security-misconfiguration, headers-audit, cors-misconfig, host-header, actuator-endpoints, tls-ssl-audit, dns-email-auth, subdomain-takeover-detection |
| transport / smuggling | smuggling, web-cache-poisoning, crlf, host-header |
| recon feeds | recon-workflow, vhost-enum, osint-research, shodan-censys-search, hackerone-intelligence, urlscan-analysis, google-dorking, github-secrets-scanner, email-user-enumeration, javascript-deep-analysis, improper-inventory |
| chains / impact | exploit-chains, aco-chains, escalator, chain-analysis |
| verification | adversarial-verification, false-positive-filter, mandatory-triage-gate |

## Routing algorithm (before every test)
1. Identify surface(s) of the target (unauth/auth/API/cloud/bizlogic/client-side/config).
2. For each candidate technique class, look up the class row above.
3. Confirm the skill exists in `SKILLS.INDEX.json` (`rig.py skill dispatch <class>`).
4. If the class row is empty for a surface, consult `ai-coverage-intelligence` or generate via `skillforge`.
5. Record the chosen `(surface × technique × skill)` cell in `hypotheses/` — coverage ledger expects it.

## Dispatch rules
- Prefer the narrowest skill: use `sqli-techniques` for a single injection probe, `sqli-expert` for a full engagement-class effort.
- Meta skills (this file, agent-dispatch, hunting-methodology) are routing aids, not test techniques — never listed as the executed technique.
- If a technique is absent from the index, do NOT improvise silently: `skillforge` generate + re-run index build.

## Output
- A concrete `hypotheses/<id>.md` entry: surface, technique, skill(s), payloads, gate.