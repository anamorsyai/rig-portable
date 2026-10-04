---
name: recon-methodology
description: Attack surface reconnaissance. Use when mapping a new target, enumerating subdomains/endpoints/tech/JS, or before starting vulnerability testing on an unfamiliar host.
---

# Recon Methodology

Goal: produce a prioritized target list, not a wall of URLs.

## Subdomain enumeration
```bash
subfinder -d TARGET -silent
curl -s "https://crt.sh/?q=%25.TARGET&output=json" | jq -r '.[].name_value' | sort -u
```
Probe with httpx: `cat subs.txt | httpx -silent -status-code -title -tech-detect -follow-redirects`

## Historical URLs (hunt hidden endpoints + old params)
```bash
echo TARGET | waybackurls | sort -u > wayback.txt
echo TARGET | gau --subs | sort -u >> wayback.txt
# Filter the interesting:
grep -E "\.(json|xml|conf|env|bak|sql|php|asp|aspx|action|api)" wayback.txt
grep -E "\?(id|file|url|redirect|next|path|page|download|token|q)=?" wayback.txt
```

## Tech + version fingerprinting
- `httpx -tech-detect` (headers, body markers)
- Look at `X-Powered-By`, `Server`, `Set-Cookie` names → framework signals.
- Match versions against CVEs (feed to intel-agent).

## JS mining
```bash
echo TARGET | katana -d 3 -jc -ef css,svg,png,jpg | sort -u > js-urls.txt
# then grep the fetched JS for:
grep -E "(api|/v[0-9]/|/internal/|/admin/|/graphql|api[_-]?key|token|secret|aws|firebase|\.onion|10\.|192\.168|172\.16)" 
```

## Directory fuzzing
```bash
ffuf -u https://TARGET/FUZZ -w /usr/share/wordlists/dirb/common.txt -mc 200,301,302,403 -o fuzz.json
ffuf -u https://TARGET/FUZZ -w /usr/share/wordlists/dirbuster/directory-list-2.3-medium.txt -mc 200 -recursion
```

## Priority model (highest bounty first)
1. Admin/panel endpoints, debug pages, legacy tech with CVEs
2. APIs with object IDs (BOLA candidates), GraphQL endpoints
3. Upload endpoints, webhooks, URL-fetch/image-proxy (SSRF candidates)
4. Auth flows: login, reset, register, OAuth (ATO candidates)
5. Any endpoint echoing user input (XSS/reflection), params hitting DB/file (injection)

## Output
`recon/map.md` with: live hosts + status, tech stack + versions, target list prioritized,
endpoints grouped by vuln class, WAF notes.