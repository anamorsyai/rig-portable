---
name: wayback-machine-mining
description: "Wayback Machine mining: extract old endpoints, deleted pages, hidden parameters from web archives."
---

# Wayback Machine Mining

## URL Extraction
- waybackurls tool for bulk extraction
- Filter unique paths
- Remove static assets (JS, CSS, images)
- Focus on dynamic endpoints

## Old Endpoint Discovery
- Deleted pages still accessible
- Deprecated API endpoints
- Old admin panels
- Removed features

## Parameter Discovery
- GET parameters from archived URLs
- POST parameters from forms
- Hidden parameters in old versions

## File Discovery
- Old config files
- Backup files
- Test/debug pages
- Internal documentation

## Integration
- Feed into ffuf for fuzzing
- Feed into arjun for parameter testing
- Compare current vs archived endpoints
- Identify removed security controls
