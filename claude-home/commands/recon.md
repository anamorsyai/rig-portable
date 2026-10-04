---
description: Full recon on a target domain. Enumerates subdomains, endpoints, technologies, and maps the attack surface.
---

Run a full reconnaissance workflow on: $ARGUMENTS

## Phase 1: Passive
```bash
# Subdomain enumeration
subfinder -d $ARGUMENTS -silent -o /tmp/subs.txt

# Certificate transparency
curl -s "https://crt.sh/?q=%25.$ARGUMENTS&output=json" | jq -r '.[].name_value' | sort -u >> /tmp/subs.txt
```

## Phase 2: Active
```bash
# HTTP probing with tech detection
cat /tmp/subs.txt | httpx -silent -status-code -title -tech-detect -follow-redirects -o /tmp/probes.txt

# Crawl for endpoints
echo $ARGUMENTS | katana -d 3 -jc -ef css,svg,png,jpg -o /tmp/katana.txt

# Wayback URLs
echo $ARGUMENTS | waybackurls | sort -u > /tmp/wayback.txt
```

## Phase 3: Analysis
```bash
# Directory fuzzing (common wordlist)
ffuf -u https://$ARGUMENTS/FUZZ -w /usr/share/wordlists/dirb/common.txt -mc 200,301,302,403 -s -o /tmp/ffuf.json -of json

# Nuclei template scan
cat /tmp/probes.txt | nuclei -severity medium,high,critical -silent -o /tmp/nuclei.txt
```

Analyze all results and provide:
1. Subdomain count and status code breakdown
2. Interesting endpoints grouped by vuln class
3. Technology stack summary
4. Recommended attack vectors to test next
