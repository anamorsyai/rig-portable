---
name: cloud-iam
description: "Cloud IAM abuse: policy audit, role chaining, cross-account access, credential harvesting."
---

# Cloud IAM Abuse

## AWS IAM
- Enumerate roles via STS AssumeRole
- Role chaining: assume role that can assume another
- Cross-account access via trust policies
- Check for overly permissive policies (* actions)

## Metadata Endpoint
- http://169.254.169.254/latest/meta-data/iam/security-credentials/
- IMDSv1 (no token required) = vulnerable
- IMDSv2 (token required) = harder but still possible via SSRF

## Credential Harvesting
- Environment variables: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
- Config files: ~/.aws/credentials, ~/.aws/config
- Hardcoded in source code

## Privilege Escalation
- iam:PassRole + lambda:CreateFunction = code execution
- iam:PassRole + ec2:RunInstance = code execution via user data
- iam:AttachUserPolicy = full admin
- iam:CreateAccessKey for another user

## GCP
- metadata.google.internal/computeMetadata/v1/
- service-account/token endpoint
- Default credentials chain
