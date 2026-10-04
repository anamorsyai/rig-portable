---
name: data-exfiltration
description: "Data exfiltration: DNS exfil, HTTP tunneling, S3 exfil, blind data extraction techniques."
---

# Data Exfiltration

## DNS Exfiltration
- Encode data in DNS queries: echo data | base64 | sed 's/./&./g'
- Use DNS TXT records for responses
- Slow exfil to avoid detection

## HTTP Tunneling
- Embed data in HTTP headers (Cookie, Referer, User-Agent)
- Use DNS over HTTPS for covert channel
- Embed in image metadata (steganography)

## S3/Cloud Storage
- Copy data to attacker-controlled S3 bucket
- Use pre-signed URLs for extraction
- Exploit misconfigured bucket policies

## Blind Data Extraction
- SQL injection with LOAD_FILE()
- SSRF to read internal files
- XXE file read
- Error-based data extraction

## Large-Scale Scraping
- Paginate through all records
- Use API bulk export endpoints
- Automate with concurrent requests
- Respect rate limits to avoid detection
