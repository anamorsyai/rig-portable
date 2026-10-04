---
name: claude-bughunter-bundle
description: Namespaced deep-dive layer ported from elementalsouls/Claude-BugHunter. Supplementary technique skills and meta-gates for classes not covered by the core rig catalog. NOT the primary router; these load as claude-bughunter: secondaries. Use alongside core skills when the core coverage is absent or needs a report-grounded deep-dive. Usage: skill-router routes here when the core index has no cell for the (surface × technique) pair.
category: meta-orchestration
---

# Claude-BugHunter Bundle (namespaced)

Curated port of the best technique-depth + meta-gate skills from
`elementalsouls/Claude-BugHunter` (Sachin Sharma), installed **without touching**
the core rig's 153 skills, `skill-router`, `SKILLS.INDEX`, or the hawk-level
(`hunt`/`recon`/`report`/`chain`) commands.

## Why namespaced, not flattened

- The core rig already covers ~40 of the same vuln classes (xss, ssrf, sqli,
  oauth, cors, csrf, graphql, race…). Flattening their `hunt-*` twins would
  **double-load** every class and let two different payload tables fire on the
  same test — muddying `hypotheses/` coverage ledgers.
- The core `skill-router` is the single selection authority. This bundle is a
  **secondary deep-dive layer**: routed to *only* when the core index has no
  cell for the (surface × technique) pair, or when a finding needs the
  report-grounded bypass tables these skills carry.

## What this bundle adds (new-only)

| Skill | Class | Why ported |
|---|---|---|
| `hunt-llm-ai` | LLM/AI (prompt injection, agentic, exfil) | Core `ai-llm-security` is thin; this is 2026-current OWASP GenAI/agentic depth |
| `hunt-saml` | SAML/SSO (XSW, key confusion, replay) | Core `saml-sso` lacks the assertion-level attack tables |
| `hunt-cache-poison` | Web cache poisoning / WCD | Core `web-cache-poisoning` lacks 2024 path-normalization WCD |
| `hunt-source-leak` | JS maps, .env/.git, swagger discovery | Core exposes git-svn; this is the broader leak playbook |
| `hunt-mfa-bypass` | 7-pattern MFA/2FA bypass | Core `mfa-2fa-bypass` lacks race+recovery-dump patterns |
| `hunt-dom` | DOM clobbering / postMessage (client-side) | Core `dom-clobbering`/`postmessage` are narrower |
| `hunt-nosqli` | MongoDB/NoSQL injection | Core `nosql-injection` lacks Mongo operator suites |
| `hunt-shadow-api` | API version-inventory + behavioral diff | Core `improper-inventory` overlaps; this is deeper |
| `hunt-spa-api` | SPA JS-bundle → backend API discovery | New play, high yield |
| `hunt-lfi` | LFI path traversal depth | Core `path-traversal` is broad; this is the probe table |
| `hunt-deserialization` | Deser chains (Java/Py/JS) | Core `deserialization-attacks` is thin |
| `hunt-misc` | 225-report misc class | Broad catch-all from their corpus |
| `offensive-osint` | 15-module recon arsenal | Far deeper than core recon feeds |
| `triage-validation` | 7-Question Gate | **Cross-check** against core `mandatory-triage-gate` |
| `evidence-hygiene` | PoC/HAR/PII redaction | Core lacks a redaction protocol |
| `recon-scope-triage` | ASM ownership noise triage | Core lacks it |
| `mid-engagement-ir-detection` | Client-patch/attacker detection | Core lacks it |

## Router bridge

Core `skill-router` stays authoritative. This bundle registers as a secondary:

- **Entry namespaces:** every `hunt-*` skill here is the **deep-dive variant**
  for its class — prefer core `*expert`/`*-techniques` for a single probe, route
  here when the core cell is empty or the class isn't covered.
- **Meta-gate cross checks:** `triage-validation` (7Q Gate) and `evidence-hygiene`
  run as a *secondary* gate alongside `mandatory-triage-gate` before any finding
  ships — they do not replace the core gate.
- **Commands:** `/token-scan`, `/memory-gc`, `/claude-bughunter-scope` (renames
  of their `token-scan`/`memory-gc`/`scope` to avoid any `/scope` ambiguity).
  The colliding `hunt`/`recon`/`report`/`chain` commands are deliberately **not**
  installed — the core owns those.

## Do-not-touch

- Core `~/.claude/skills/*` (all 153)
- Core `~/.claude/commands/*` including `hunt`, `recon`, `report`, `chain`
- Core `skill-router`, `SKILLS.INDEX`, `skillforge`, `agent-dispatch`
- Core memory/system state, daemon, memory hooks

## Upstream

`github.com/elementalsouls/Claude-BugHunter` — skill bundle by Sachin Sharma.
This is a curated subset (17 skills + 3 commands), ported 2026-08-29.
