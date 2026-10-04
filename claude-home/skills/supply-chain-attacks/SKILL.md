---
name: supply-chain-attacks
description: "Supply chain attacks: dependency confusion, typosquatting, CI/CD abuse, package registry poisoning."
---

# Supply Chain Attacks

## Dependency Confusion
- Discover internal package names (GitHub, error messages, source)
- Publish same-named package to public registry with higher version
- npm: malicious postinstall script runs on npm install
- PyPI: malicious setup.py executes on pip install

## Typosquatting
- Register similar domain names to popular packages
- Common typos: lodash vs lodas, express vs experss
- Use package manager typosquatting databases

## CI/CD Pipeline
- Compromise CI/CD config files (.github/workflows, .gitlab-ci.yml)
- Inject malicious build steps
- Poison build artifacts

## Lock File Poisoning
- Modify package-lock.json / yarn.lock to point to malicious versions
- Commit poisoned lock files

## Detection
- Check for unexpected network calls during install
- Audit package dependencies for known malicious packages
- Use npm audit, pip-audit, safety
- Check for typosquatting: npm-name-checker
