---
name: attack-forensics
description: "Attack forensics: log analysis, attack path reconstruction, evidence chain, impact quantification."
---

# Attack Forensics

## Log Analysis
- Web server logs: find attack requests
- Application logs: find error patterns
- Auth logs: find unauthorized access attempts
- Database logs: find injection attempts

## Attack Path Reconstruction
- Timeline: first access -> privilege escalation -> data access
- Map all endpoints accessed during attack
- Identify all data modified/exfiltrated
- Document credential usage

## Evidence Chain
- Preserve raw logs (never modify)
- Hash files for integrity
- Timestamp all evidence
- Chain of custody documentation

## Impact Quantification
- Count affected records
- Estimate data sensitivity
- Calculate financial exposure
- Assess regulatory implications

## PoC Documentation
- Step-by-step reproduction
- Raw HTTP requests/responses
- Screenshots with timestamps
- Before/after state comparison
