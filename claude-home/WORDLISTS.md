# Hunting Rig — Wordlists Index

All wordlists under `/opt/`. Pipelines default to these via `~/tools/recon/lib.sh`.

## SecLists — `/opt/SecLists` (2.1GB)
Sparse clone: Discovery (Web-Content, DNS), Fuzzing, Passwords, Payloads, Usernames.

### Content discovery (FFUF / Feroxbuster / Dirsearch)
- `Discovery/Web-Content/common.txt` — quick first pass (~4.7k)
- `Discovery/Web-Content/raft-medium-words.txt` — default medium fuzz (~38k)
- `Discovery/Web-Content/raft-medium-directories.txt` — directories
- `Discovery/Web-Content/raft-large-words.txt` / `raft-large-directories.txt` — deep pass
- `Discovery/Web-Content/backup.txt` / `config.txt` — backup/config files
- `Discovery/Web-Content/Common-PHP-Filenames.txt` — PHP-specific

### DNS brute
- `Discovery/DNS/subdomains-top1million-5000.txt` — quick
- `Discovery/DNS/deepmagic.com-prefixes-top50000.txt` — deeper

### Auth/password testing
- `Passwords/Leaked-Databases/rockyou-*.txt` (10..70 length variants, 1M+ entries)
- `Passwords/Common-Credentials/10-million-password-list-top-100000.txt`
- `Passwords/Default-Credentials/` — vendor default creds
- `Usernames/xato-net-10-million-usernames.txt` — username bruteforce

### Payloads
- `Payloads/` — XSS/SQLi/command-injection payload collections
- `Fuzzing/` — generic fuzz lists

## Assetnote — `/opt/wordlists/assetnote` (from wordlists-cdn.assetnote.io)
- `best-dns-wordlist.txt` — 9.8MB, the best DNS brute list (pipeline default)
- `2m-subdomains.txt` — 2M+ subdomains from GitHub BigQuery dataset
- `phpmillion.txt` / `php.txt` / `jsp.txt` / `asp_lowercase.txt` / `aspx_lowercase.txt` — tech-specific path lists
- `bak.txt` — backup file names
- `dot_filenames.txt` / `xml_filenames.txt` — dotfile/xml file names
- `html.txt` / `cfm.txt` / `do.txt` / `pl.txt` — extension-specific
- More: `https://wordlists-cdn.assetnote.io/data/` (automated httparchive lists for deep fuzzing)

## fuzzdb — `/opt/fuzzdb` (23MB)
- Attack payloads: SQLi, XSS, LDAP, command injection, traversal
- `attack/` — payload dictionaries per vector
- `discovery/` — web paths

## PayloadsAllTheThings — `/opt/PayloadsAllTheThings` (21MB)
- Per-vuln-class payload cheat-sheets + payloads: SSTI, SSRF, XXE, IDOR, GraphQL, etc.
- `Directory Traversal/`, `SQL Injection/`, `XSS Injection/`, `Server Side Injection/`

## Pipeline integration
- `lib.sh` defaults: `DNS_WORDS=/opt/wordlists/assetnote/best-dns-wordlist.txt`,
  `WEB_WORDS=raft-medium-words`, `PASS_WORDS`, `USER_WORDS`
- Override per-run: `DNS_WORDS=/path ./recon.sh <target> all`