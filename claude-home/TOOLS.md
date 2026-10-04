# Hunting Rig — Tool Inventory

Install date: 2026-08-19. Alpine Linux 3.24, Go 1.26.3, Python 3.14.

## Recon / Asset Discovery
| Tool | Where | Purpose |
|------|-------|---------|
| subfinder | /usr/local/bin | subdomain enumeration |
| assetfinder | /usr/local/bin | subdomain enumeration (passive) |
| dnsx | /usr/local/bin | DNS probing/validation |
| naabu | /usr/local/bin | port scanning |
| puredns | /usr/local/bin | DNS bruteforce |
| alterx | /usr/local/bin | subdomain permutation/generation |
| nmap | /usr/bin | network/port scanning |
| masscan | /usr/bin | fast port scanning |
| mapcidr | /usr/local/bin | CIDR expansion |
| httpx | /usr/local/bin | HTTP probing, tech detection |
| whatweb | /usr/local/bin | web fingerprinting |
| wafw00f | /usr/bin | WAF fingerprinting |

## Content / Param Discovery
| Tool | Where | Purpose |
|------|-------|---------|
| ffuf | /usr/local/bin | fuzzing (dirs, vhosts, params) |
| feroxbuster | /usr/local/bin | recursive content discovery |
| gobuster | /usr/local/bin | dirs/vhosts/DNS bruteforce |
| dirsearch | /usr/bin | directory bruteforce |
| arjun | /usr/bin | hidden parameter discovery |

## URL / Endpoint Collection
| Tool | Where | Purpose |
|------|-------|---------|
| katana | /usr/local/bin | crawler (projectdiscovery) |
| gospider | /usr/local/bin | crawler/spider |
| hakrawler | /usr/local/bin | crawler/spider |
| getJS | /usr/local/bin | JS file discovery |
| gau | /usr/local/bin | historical URLs (getallurls) |
| waybackurls | /usr/local/bin | wayback URLs |
| waymore | /usr/bin | wayback/commoncrawl/otx URLs |
| uro | /usr/bin | URL dedup/normalization |

## Vulnerability Scanning
| Tool | Where | Purpose |
|------|-------|---------|
| nuclei | /usr/local/bin | template-based scanner (templates in ~/nuclei-templates) |
| sqlmap | /usr/bin | SQLi exploitation |
| dalfox | /usr/local/bin | XSS scanner |
| xsstrike | /usr/bin | XSS scanner |
| nikto | /usr/bin/nikto.pl | web server scanner |
| kxss | /usr/local/bin | reflected XSS param filter |
| trufflehog | /usr/local/bin | leaked credential scanning |

## Post-Processing / Piping
| Tool | Where | Purpose |
|------|-------|---------|
| gf | /usr/local/bin | grep patterns (~/.gf, 14 patterns) |
| unfurl | /usr/local/bin | URL parsing |
| qsreplace | /usr/local/bin | query string replacement |
| anew | /usr/local/bin | append-newlines dedup |
| jq | /usr/bin | JSON processing |
| interactsh-client | /usr/local/bin | OAST (out-of-band) callbacks |

## Auth / Token / Other
| Tool | Where | Purpose |
|------|-------|---------|
| jwt_tool | /usr/local/bin -> /opt/jwt_tool | JWT attacks |
| curl-impersonate | /usr/local/bin/curl_chrome* | TLS-fingerprint rotation for WAF bypass |
| proxychains | /usr/bin | route via socks (config: /etc/proxychains/proxychains.conf) |
| playwright MCP | ~/.claude.json mcpServers | headless Chromium automation (crawl, JS, DOM XSS) |
| browser-control MCP | ~/.claude.json mcpServers | drives real user browser via extension (:8089) for anti-bot-gated flows |

## External MCP
- Burp MCP: http://127.0.0.1:9876 (SSH-tunneled; raw HTTP replay/history/Collaborator)
- Caido MCP: http://127.0.0.1:3333/mcp (SSH-tunneled; captured-traffic analysis/replay)

## Recon pipeline
- Methodology: `~/.claude/RECON-METHODOLOGY.md`
- Executable pipeline: `~/tools/recon/` (`./recon.sh <target> all`)
- Wordlists: `/opt/SecLists` (Discovery/Web-Content, Discovery/DNS, Fuzzing)
- Resolvers: `~/.config/resolvers.txt`; optional API keys in `~/.config/recon/config.sh`
- httpx ML model cached at `~/.dit/model.json` (avoids first-run HuggingFace stall)

## Reinstall notes
- Go tools: `GOBIN=/usr/local/bin go install <pkg>@latest` (see /etc/profile.d/hunt-env.sh)
- pip tools: `pip3 install --break-system-packages --no-cache-dir <pkg>`
- trufflehog: clone https://github.com/trufflesecurity/trufflehog then `go build` (go install blocked by replace directives)