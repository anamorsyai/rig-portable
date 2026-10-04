---
description: Enumerate hidden directories, files, and parameters on a target.
---

Directory and parameter enumeration on: $ARGUMENTS

```bash
# Directory brute-force (common wordlists)
ffuf -u https://$ARGUMENTS/FUZZ -w /usr/share/wordlists/dirb/common.txt -mc 200,301,302,403 -s -o /tmp/dir_ffuf.json -of json

# Sensitive file discovery
ffuf -u https://$ARGUMENTS/FUZZ -w /usr/share/wordlists/dirbuster/directory-list-2.3-medium.txt -mc 200,301,403 -fs 0 -s -o /tmp/dir_medium.json -of json

# Parameter discovery
ffuf -u https://$ARGUMENTS/FUZZ -w /usr/share/seclists/Discovery/Web-Content/burp-parameter-names.txt -mc 200,301 -fs 0 -s -o /tmp/params.json -of json

# Subdomain takeover check
subfinder -d $ARGUMENTS -silent | httpx -silent -status-code -cname | grep -E "(NXDOMAIN|CNAME.*\.cloudfront\.net|CNAME.*\.amazonaws\.com|CNAME.*\.azurewebsites\.net|CNAME.*\.herokuapp\.com|CNAME.*\.github\.io|CNAME.*\.surge\.sh)"
```

Analyze results and provide:
1. Discovered directories with status codes
2. Interesting files (configs, backups, source code)
3. Discoverable parameters
4. Potential subdomain takeover candidates
5. All 403 endpoints worth bypassing
