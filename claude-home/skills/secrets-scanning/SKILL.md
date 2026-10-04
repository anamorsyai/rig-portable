---
name: secrets-scanning
description: Scan JS files, configs, responses for leaked API keys, tokens, passwords, internal URLs.
---

# Secrets Scanning

## JS File Patterns
- API keys: /api[_-]?key["'=:][ ]*["'][A-Za-z0-9]{20,}["']
- Tokens: /token["'=:][ ]*["'][A-Za-z0-9\-._]{20,}["']
- Secrets: /secret["'=:][ ]*["'][A-Za-z0-9\-._]{10,}["']
- Passwords: /password["'=:][ ]*["'][^"]{6,}["']
- AWS keys: /AKIA[0-9A-Z]{16}/

## Response Headers
- X-Api-Key, Authorization, X-Auth-Token, Set-Cookie

## HTML Source
- Meta tags with secrets
- Hidden form fields
- Commented credentials

## Tools
- trufflehog, gitleaks for repo scanning
- grep for patterns in JS bundles
- LinkFinder for endpoint discovery
