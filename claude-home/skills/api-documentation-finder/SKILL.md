---
name: api-documentation-finder
description: "API doc finder: find Swagger/OpenAPI, GraphQL schemas, WSDL files, Postman collections, API changelogs."
---

# API Documentation Finder

## Common Paths
- /swagger.json, /swagger-ui/
- /openapi.json, /openapi.yaml
- /graphql (introspection enabled)
- /wsdl, /wsdl?wsdl
- /api/docs, /api-docs
- /postman.json

## Swagger/OpenAPI
- Full endpoint listing
- Parameter schemas
- Authentication methods
- Data models

## GraphQL
- Introspection query: {__schema{types{name,fields{name}}}}
- Type definitions
- Available queries/mutations
- Directive definitions

## Postman Collections
- Full request library
- Environment variables
- Pre-request scripts
- Test scripts

## Impact
- Complete API surface mapping
- Hidden endpoint discovery
- Authentication bypass vectors
- Data model understanding
