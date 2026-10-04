---
name: claude-bughunter-bridge
description: Router bridge for the claude-bughunter-bundle namespaced secondaries. Maps (surface × technique) pairs with no core-index cell to their namespaced hunt-* deep-dive skill. Loaded by skill-router ONLY when the core index has no assignment — secondaries never override core cells. Also maps meta-gate cross-checks (triage-validation, evidence-hygiene) to run as secondary gates alongside the core mandatory-triage-gate.
category: meta-orchestration
---

# Router Bridge — claude-bughunter-bundle → core

Core `skill-router` is the single selection authority. This bridge is consulted
**only when the core index has no cell** for a (surface × technique) pair, or
when the core coverage lacks the report-grounded deep-dive. Secondaries never
override core cells; if a core cell exists, the core skill wins and this bridge
is skipped.

## (surface × technique) → namespaced deep-dive

| Surface | Technique class | Core cell empty? → route to (namespaced) |
|---|---|---|
| auth flows | SAML/SSO assertion attacks | `claude-bughunter:hunt-saml` |
| auth flows | MFA/2FA bypass (race, recovery-dump) | `claude-bughunter:hunt-mfa-bypass` |
| AI / LLM features | prompt injection, agentic, exfil | `claude-bughunter:hunt-llm-ai` |
| cache layer | web cache poisoning / WCD | `claude-bughunter:hunt-cache-poison` |
| source / build | JS maps, .env/.git, swagger leak | `claude-bughunter:hunt-source-leak` |
| client-side | DOM clobbering / postMessage / SW | `claude-bughunter:hunt-dom` |
| NoSQL | MongoDB operator injection | `claude-bughunter:hunt-nosqli` |
| API (versioned) | shadow/zombie API + behavioral diff | `claude-bughunter:hunt-shadow-api` |
| SPA | JS-bundle → backend API discovery | `claude-bughunter:hunt-spa-api` |
| file read | LFI probe tables | `claude-bughunter:hunt-lfi` |
| deserialization | Java/Py/JS deser chains | `claude-bughunter:hunt-deserialization` |
| any / catch-all | misc from 225-reports corpus | `claude-bughunter:hunt-misc` |
| recon feeds | full offensive-osint arsenal | `claude-bughunter:offensive-osint` |

## Meta-gate cross-checks (secondary, never primary)

- `claude-bughunter:triage-validation` — the 7-Question Gate runs as a **secondary
  cross-check** after the core `mandatory-triage-gate` passes. Disagreement is
  flagged for the operator; it does not veto a core-passed finding by itself.
- `claude-bughunter:evidence-hygiene` — PII/cookie/HAR redaction discipline runs
  before any PoC screenshot or HAR attach; it pairs with core `report-template`.

## Rig-native concept mapping (their bundle name → our canon)

Bundle skills reference methodology/reporting concepts under their own names.
Map to the rig-native equivalents (these are the same concepts, different names):

| Their name | Rig-native equivalent |
|---|---|
| `bb-methodology` | `hunting-methodology` (core) + `hunt-operating-contract` (canon, authoritative) |
| `redteam-mindset` | `hunt-operating-contract` § "DO NOT STOP primary directive" + `aggressive-hunting` (core) |
| `redteam-report-template` | `report-template` (core) |
| `security-arsenal` | `~/hunting-rig/payloads/` payload pack + `waf-bypass`/`waf-bypass-advanced` (core) |

## Unported bundle skills (their engine/reporting layer — deliberately excluded)

These are referenced by installed bundle skills but were **intentionally not
installed** — they belong to BugHunter's orchestration/reporting layer, not the
rig canon's. Treat refs as pointers to the rig-native equivalent:

| Ref | Rig-native equivalent |
|---|---|
| `hunt-dispatch` | `agent-dispatch` (core) + `skillrouter` cell lookup |
| `report-writing` | `report-template` (core) + `writeup-instructor`/`writeup-summarizer` |
| `bugcrowd-reporting` | `report-template` (core) — canonical reporting |
| `security-arsenal` | n/a (payload pack) — referenced as "where payloads live"; rig uses `~/hunting-rig/payloads/` |

## Cross-class chain hooks — resolved to core

Bundle skills reference sibling `hunt-*` skills as **chain primitives** (e.g.
hunt-cache-poison → hunt-xss/hunt-http-smuggling/hunt-auth-bypass). Those
siblings are not installed; resolve them to the core skill that owns the class:

| Bundle chain hook | Core skill (resolve to) |
|---|---|
| `hunt-xss` | `claude-bughunter-bundle` → `xss` / `xss-exploitation` (core) |
| `hunt-http-smuggling` | `smuggling` (core) |
| `hunt-auth-bypass` | `auth-bypass` / `auth-bypass-session` (core) |
| `hunt-host-header` | `host-header` (core) |
| `hunt-file-upload` | `upload-bypass` (core) |
| `hunt-idor` | `idor-bola` / `idor-expert` (core) |
| `hunt-rce` | `command-injection` / `template-injection` / `ssti` (core) |
| `hunt-ssrf` | `ssrf` / `ssrf-exploitation` (core) |
| `hunt-ato` | `session-attacks` / `auth-bypass` / `ido—expert` (core) |
| `hunt-race-condition` | `race-condition-testing` (core) |
| `hunt-api-misconfig` | `api-fuzzing` / `security-misconfiguration` (core) |
| `hunt-business-logic` | `business-logic` / `business-logic-deep` (core) |
| `hunt-subdomain` | `subdomain-takeover-detection` / `recon-workflow` (core) |
| `hunt-oauth` | `oauth-abuse` (core) |
| `hunt-xxe` | `xxe-expert` / `xxe-injection` (core) |
| `hunt-sqli` | `sqli` / `sqli-techniques` / `sqli-expert` (core) |
| `hunt-cloud-misconfig` | `cloud-iam` / `cloud-bucket-enumeration` (core) |
| `hunt-nextjs` | `javascript-deep-analysis` (core) |
| `hunt-brute-force` | `parameter-fuzzing` / `value-fuzzing` (core) |

When a bundle skill names one of these as a chain primitive, read the chain
primitive from the CORE skill (it owns the technique); the bundle skill owns
only its own class's depth.

## Engagement-phase hooks

- `claude-bughunter:recon-scope-triage` — run at the START of any engagement,
  immediately on receiving any ASM/recon/OSINT dataset, BEFORE testing anything.
  Separates the target's real assets from namespace-collision noise.
- `claude-bughunter:mid-engagement-ir-detection` — when a confirmed finding stops
  reproducing, baseline timing shifts, or response patterns change during an
  active engagement.

## Commands (namespaced, no collision)

| Command | Core equivalent | Status |
|---|---|---|
| `/claude-bughunter-scope` | n/a (core has no /scope) | Deterministic deny-wins scope check |
| `/token-scan` | n/a | Meme/token rug-pull audit |
| `/memory-gc` | n/a | Autopilot ledger rotation |

The colliding `/hunt`, `/recon`, `/report`, `/chain` are intentionally NOT
installed — the core owns those.

## Usage

Directory-scoped to `~/.claude/skills/claude-bughunter-bundle/`. Invoked by the
core router or by name (`claude-bughunter:hunt-llm-ai`). If a technique is
absent from both core and bridge, do NOT improvise silently: log the gap in
`hypotheses/` and mark it uncovered.
