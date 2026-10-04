---
name: cloud-bucket-enumeration
description: S3/GCS/Azure bucket enumeration, public access testing, credential discovery.
---

# Cloud Bucket Enumeration

## S3 Bucket Enumeration
- List: aws s3 ls s3://BUCKET/
- Public access: curl https://BUCKET.s3.amazonaws.com/
- Enumerate with: bucket_finder, S3Scanner
- Common names: {company}-{env}-{region}

## GCS Bucket
- curl https://storage.googleapis.com/BUCKET/
- Public listing: gsutil ls gs://BUCKET/

## Azure Blob
- curl https://BUCKET.blob.core.windows.net/?restype=container comp=list
- Public: ?comp=list restype=container

## Credential Discovery
- Check JS for AWS keys (AKIA...)
- Check configs for access_key/secret_key
- Check metadata endpoint for IAM role

## Access Levels
- Private (403) -> Try path traversal
- Authenticated listed -> Try ACL manipulation
- Public read -> Download all
- Public write -> Upload webshell
