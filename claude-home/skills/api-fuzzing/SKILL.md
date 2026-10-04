---
name: api-fuzzing
description: Automated API endpoint discovery, parameter fuzzing, method testing, content-type manipulation.
---

# API Fuzzing

## Endpoint Discovery
- Brute force paths: /api/v1/FUZZ
- Try REST verbs: GET, POST, PUT, PATCH, DELETE
- Common suffixes: /list, /search, /export, /import

## Parameter Discovery
- arjun for hidden params
- Try JSON body params
- Try XML content-type
- GraphQL introspection

## Method Testing
- OPTIONS -> allowed methods
- Try PATCH on GET-only endpoints
- Try DELETE on any resource
- HEAD for existence check

## Content-Type Fuzzing
- application/json
- application/xml
- multipart/form-data
- application/x-www-form-urlencoded

## Auth Testing
- Remove auth header -> 200 = vulnerability
- Try API key in query string
- Try Bearer token manipulation
- Try cookie-based auth bypass
