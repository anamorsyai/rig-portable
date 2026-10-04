---
name: github-secrets-scanner
description: "GitHub leaks finder: search for leaked secrets, hardcoded credentials, API keys, config files on GitHub."
---

# GitHub Secrets Scanner

## Search Queries
- "password" language:json
- "api_key" filename:.env
- "secret" filename:config
- "AWS_ACCESS_KEY" filename:.env
- "PRIVATE KEY" path:.ssh
- org:TARGET path:config

## Target Patterns
- Hardcoded API keys in source
- Database credentials in config files
- AWS/GCP/Azure keys in environment
- JWT secrets, signing keys
- Internal URLs and endpoints
- .env files committed to repo

## Detection Tools
- trufflehog (entropy-based)
- gitleaks (pattern-based)
- git-secrets (pre-commit)
- GitHub secret scanning alerts

## Impact
- Direct credential access
- Internal endpoint discovery
- Technology stack confirmation
- Source code exposure
