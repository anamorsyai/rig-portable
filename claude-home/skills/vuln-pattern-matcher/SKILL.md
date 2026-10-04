---
name: vuln-pattern-matcher
description: "Vulnerability pattern matching: match known CVE patterns, vulnerability signatures, and attack patterns to target."
---

# Vulnerability Pattern Matcher

## CVE Pattern Matching
- Version-specific vulnerabilities
- Configuration weaknesses
- Default credential patterns
- Known vulnerable components

## Attack Pattern Matching
- SQL injection signatures (error messages, timing)
- XSS reflection points (input in output)
- SSRF entry points (URL parameters, webhooks)
- File inclusion patterns (include, require, template)

## Tech Stack Patterns
- PHP: file inclusion, deserialization, type juggling
- Java: deserialization, SSRF, XXE
- Node.js: prototype pollution, NoSQL injection
- Python: pickle, SSTI, format string
- Ruby: deserialization, SSTI

## Detection Signatures
- Error message patterns (DB errors, stack traces)
- Response header patterns (server versions, debug flags)
- Timing patterns (sleep in response = injection)
- Content patterns (reflected input, error pages)
