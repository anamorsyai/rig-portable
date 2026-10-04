---
name: subdomain-takeover-detection
description: Detect dangling CNAMEs, unclaimed services, and takeover opportunities.
---

# Subdomain Takeover Detection

## Detection Flow
1. Find all CNAME records: dig CNAME sub.domain.com
2. Check if CNAME target resolves
3. If NXDOMAIN -> potential takeover
4. Check target on fingerprinting databases

## Services to Check
- GitHub Pages (*.github.io)
- Heroku (*.herokuapp.com)
- AWS S3 (*.s3.amazonaws.com)
- Azure (*.azurewebsites.net)
- Fastly (*.fastly.net)
- Cloudfront (*.cloudfront.net)
- Shopify (*.myshopify.com)

## Fingerprinting
- Check response for service-specific patterns
- Subjack, CanITakeOverXYZ templates
- Nuclei templates for takeover verification

## Takeover
- Register claimed service
- Upload content to prove impact
- Report with proof of control
