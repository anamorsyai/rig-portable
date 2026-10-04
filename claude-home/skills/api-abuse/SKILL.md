---
name: api-abuse
description: "API abuse: rate limit bypass, quota abuse, schema stitching, endpoint enumeration via error messages."
---

# API Abuse

## Rate Limit Bypass
- Rotate IP addresses (VPN, proxy, IPv6)
- Rotate API keys (multiple accounts)
- Rotate session tokens
- Use different HTTP methods
- Split requests across endpoints

## Quota Abuse
- Use free tier limits maximally
- Create multiple accounts for quota stacking
- Abuse generous rate limits for data extraction

## Schema Stitching
- Use GraphQL introspection to map entire API
- Combine multiple small queries for large data
- Exploit verbose error messages for schema discovery

## Endpoint Enumeration
- Try common API paths (/api/v1/, /graphql, /swagger)
- Check for /docs, /openapi.json, /swagger.json
- Use error messages to discover hidden fields
- Try deprecated API versions

## Error Message Abuse
- Trigger validation errors to discover schema
- Use type errors to infer parameter types
- Exploit debug modes for stack traces
