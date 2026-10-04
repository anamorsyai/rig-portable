---
name: serverless-lambda
description: Serverless (AWS Lambda/API Gateway, Cloud Functions) security — misconfigured permissions, exposed invoke endpoints, env-secret leakage, SSRF from Lambda roles, and function URL exposure. Use when targets run serverless backends (api gateway subdomains, `execute-api`, `.cloudfront.net`, `function.app`).
category: cloud-infra
---

# Serverless / Lambda Security

## Detection
- Fingerprint: `execute-api.<region>.amazonaws.com`, `*.cloudfront.net` + `x-amz-apigw-*`, `x-aws-exec-arn`, `x-amzn-RequestId`, function URLs `*.lambda-url.<region>.on.aws`, Google `cloudfunctions.net`, Azure `azurewebsites.net` with function name.
- Test unauthenticated invoke: `POST /Prod/yourfunction` with arbitrary JSON.

## Exploitation
1. **Unauthenticated invoke**: public function URL / API Gateway route with no authz → execute business logic / leak data.
2. **Env-secret leak**: function returns error/debug body containing `process.env` / `os.environ` values.
3. **SSRF via IAM role**: function fetches user URL (image proxy, webhook) → `GET http://169.254.169.254/latest/meta-data/iam/security-credentials/` → **creds → AWS takeover**.
4. **Injection → env dump**: command/SQL injection inside function → `env` reveals `AWS_ACCESS_KEY_ID`, tokens.
5. **API Gateway authorizer bypass**: invoke backend route directly (function ARN) skipping the authorizer; or `?code=` `?grant=` tricks on the gateway.
6. **Overly permissive role**: function can list S3/dynamo (assumed role) — read other resources once you have execution context.

## Tool Commands (Windows)
```powershell
# unauthenticated invoke
curl.exe -s -X POST "$U/Prod/fn" -d '{}' -H 'Content-Type: application/json' -i
# SSRF to metadata
curl.exe -s -X POST "$U/Prod/img" -d '{"url":"http://169.254.169.254/latest/meta-data/iam/security-credentials/"}' -i
# header check
curl.exe -s -I "$U/" | Select-String -Pattern 'amzn|lambda' -CaseSensitive:$false
```

## Verification & Evidence
- Impact = data access or (best) assumed-role creds or SSRF to metadata.
- Redact any recovered keys; capture invoke request + response.