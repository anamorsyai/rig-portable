# Ultimate Web Recon Methodology & Pipeline

Authoritative recon playbook for the hunting rig. Pairs with the executable pipeline in
`~/tools/recon/`. Principle: **produce a prioritized, organized attack surface —
never a wall of URLs — and never silently drop a lead.**

## Core Principles

1. **Layered, never single-source.** Every class of data is harvested from ≥3 independent
   sources. A surface gathered one way is UNTESTED.
2. **Evidence-tracked.** Every phase writes outputs into a dated workspace and appends to a
   running `state.txt`. Nothing is thrown away. Finished phases are skipped on re-run unless
   `--force`.
3. **Scope is report-time, not hunt-time.** Scope never gates or thins testing (rig rule).
   It is only consulted after a finding is proven, to decide reporting.
4. **Blocked = bypass problem.** If a source/technique fails (WAF, rate limit, fingerprint
   block), rotate: method, encoding, header, `curl-impersonate`, param pollution. Ruled-out
   requires ≥3 distinct techniques with reasons.
5. **Every discovery is a resource, not a conclusion.** Hosts, paths, params, secrets and
   panels are collected for manual hunting with vuln-class grouping and priority ranking.

## Workspace Layout

```
recon/<target>/
├── state.txt              # phase completion + counts (append-only log)
├── logs/<phase>.log
├── 01-passive/            # subdomains, ips, cidr, wildcard, asn
├── 02-active/             # ports, http hosts, exposed infra
├── 03-content/            # paths, hidden files, vhosts, params, takeover
├── 04-crawl/              # urls, endpoints, params, js files
├── 05-secrets/            # secrets, leaks, panels, cloud buckets
├── 06-fingerprint/        # tech, waf, misconfig, nuclei results, cve map
├── 07-report/             # map.md + per-host summaries
└── all/                   # merged unified surface (the handoff)
    ├── domains.txt  hosts.txt  urls.txt  endpoints.txt  params.txt
    ├── js_files.txt  secrets.txt  admin_panels.txt  tech.txt
    └── graph.txt / map.md
```

## Phase 0 — Workspace & Intake

- Inputs: root domain(s), optional wildcard/CIDR/ASN, optional API keys
  (`~/.config/recon/config.sh` — shodan, virustotal, github, urlscan, censys).
- Create `recon/<target>/`, seed `input.txt` with roots.
- Wordlists live in `/opt/SecLists`; resolvers in `~/.config/resolvers.txt`.
- Record start in `state.txt`.

## Phase 1 — Passive Asset Discovery (never touch the target)

**1.1 Subdomains (multi-source, dedupe with `anew`):**
- `subfinder -d T -all -silent`
- `assetfinder --subs-only`
- `curl "https://crt.sh/?q=%25.T&output=json" | jq`
- Hackertarget, AlienVault OTX, Rapiddns, ThreatMiner/other via web/API
- With keys: SecurityTrails, VirusTotal, Shodan, Censys

**1.2 Validate + wildcard detect:**
- `dnsx -silent -a -resp` → live subdomains + IPs
- Wildcard: resolve a random host; if it answers, note it (affects all later logic).

**1.3 Permutation + brute:**
- `alterx` wordlist-driven permutation → `puredns bruteforce` against `/opt/SecLists/Discovery/DNS/*`
- Recursively validate; merge into domains.

**1.4 Network footprint (CIDR/ASN):**
- `whois`, Shodan/Censys org search (keys), input CIDRs expanded with `mapcidr`.

**1.5 Related infrastructure:** TLD variants, staging/dev/test naming conventions.

**Outputs:** `01-passive/domains.txt`, `ips.txt`, `wildcard.txt`, `cidr.txt`, `asn.txt`.

## Phase 2 — Active Host Discovery

**2.1 Port scan** (every unique IP):
- `naabu -top-ports 1000`; `masscan` for large ranges. Merge `ports.txt`.

**2.2 HTTP probing:**
- `httpx` on every host and `host:port` with full detail
  (`-status-code -title -tech-detect -server -redirect -websocket -content-type -location -hash -cdn`).
- Save JSON (structured) + split 200/301/302/401/403/500 lists.

**2.3 Exposed infra / DB candidates** from ports:
- 27017 Mongo, 9200/9300 ES, 6379 Redis, 5432 PG, 3306 MySQL, 1433 MSSQL, 8086 Influx,
  2375 Docker, 8443, 9090 Prometheus, 5601 Kibana, 3000 Grafana. → `db_exposed.txt` (manual auth/unauth test later).

**Outputs:** `02-active/ports.txt`, `http_hosts.txt`, `hosts_alive.txt`, `db_exposed.txt`.

## Phase 3 — Content & Path Discovery

**3.1 Directory brute** per live host: `ffuf`/`feroxbuster` with
`/opt/SecLists/Discovery/Web-Content/raft-medium-words.txt` + common extensions
(.php .asp .jsp .aspx .json .txt), recursive on interesting dirs.

**3.2 High-value hidden files** (dedicated small list): `.git/config`, `.env`, `.svn`,
`config.*`, `*.bak`, `*.zip`, `*.sql`, `phpinfo.php`, `robots.txt`, `sitemap.xml`,
`.well-known/*`, `swagger.json`, `openapi.yaml`, `actuator/*`, `graphql`, `crossdomain.xml`,
`.DS_Store`, source maps (`*.js.map`), `/server-status`.

**3.3 Vhost fuzzing + takeover:**
- `ffuf -H "Host: FUZZ.T"` against each host.
- `nuclei -t takeover` + manual CNAME dangling check (no A/AAAA + dangling CNAME).
- Header-based (Host header) takeover probes (Heroku/GH Pages/S3/CNAME wildcard).

**3.4 Parameter discovery:** `arjun` on high-value endpoints (also post-crawl in Phase 4).

**3.5 Tech-aware paths:** based on fingerprint — `wp-content/uploads`, `/_next`, `/api-docs`,
`/uploads/`, `/backup/`, `/api/v1/`, `/graphql`, `/v1/graphql`.

**Outputs:** `03-content/discovered_paths.txt`, `hidden_files.txt`, `vhosts.txt`,
`takeover_candidates.txt`, `params.txt`.

## Phase 4 — Crawl & URL Harvest

**4.1 Historical URLs:** `gau --subs`, `waybackurls`, `waymore` (wayback+commoncrawl+otx),
`urlscan.io` API (key). Merge → `raw_urls.txt`.

**4.2 Live crawl:** `katana -d 5 -jc -ef` , `gospider`, `hakrawler`.

**4.3 JS mining (the goldmine):**
- Collect JS via `getJS` + katana `-jc`.
- Download every JS; grep for: API routes, `/v[0-9]/`, `/internal/`, `/admin/`, `/api/`,
  GraphQL ops, Firebase config, AWS keys, tokens, hardcoded secrets, internal IPs.
- Extract endpoint skeleton (`/api/v1/users/{id}` style) for param fuzzing later.

**4.4 Normalize + dedupe:** `uro` for URL dedup/normalization, `unfurl --unique` for path
skeletons, `qsreplace` to strip params → `endpoints.txt`, `params.txt`.

**Outputs:** `04-crawl/urls.txt`, `endpoints.txt`, `params.txt`, `js_files.txt`, `js_findings.txt`.

## Phase 5 — Secrets, Leaks & Exposure Hunting

**5.1 Secret scan of everything harvested:** `trufflehog filesystem --only-verified`
and regex grep across all fetched content (AWS `AKIA`, JWT, GitHub tokens, Slack webhooks,
private keys, Firebase, Google API, `password=`, DB creds, internal IPs/`10.`/`172.16`/`192.168`).

**5.2 Source leakage (if org known):** GitHub code search for org + `trufflehog git` on
public repos; hunt committed `.env`, `.git` exposure on hosts (`curl /.git/config`).

**5.3 Cloud exposure:** S3/GCS/Azure buckets and Firebase projects found in URLs/JS — test
public listing/permissions (non-destructive: `ListObjects`).

**5.4 Admin / debug / panel hunting:** crawl a high-value path list against every live host —
`/admin`, `/administrator`, `/login`, `/dashboard`, `/panel`, `/console`, `/kibana`,
`/grafana`, `/jenkins`, `/phpmyadmin`, `/pgadmin`, `/api/swagger`, `/_next`, `/actuator`,
`/debug`, `/test`. Keep HTTP 200/302 with distinct bodies → `admin_panels.txt` (manual auth
hunting targets).

**5.5 Exposed infra:** combine Phase 2 `db_exposed.txt` with unauthenticated checks
(Redis `INFO`, Mongo no-auth, ES index listing, etc. — read-only).

**Outputs:** `05-secrets/secrets.txt`, `leaks.txt`, `admin_panels.txt`, `cloud_buckets.txt`.

## Phase 6 — Fingerprint, Misconfig & Passive Vuln Sweep

**6.1 Tech + versions:** `httpx -tech-detect`, `whatweb`, headers/cookie names; extract
versions from `package.json`/`composer.json`/changelog/`readme` in exposed static files.

**6.2 WAF/CDN:** `wafw00f` per host → `waf.txt` (drives bypass strategy later).

**6.3 Misconfig:** missing `Content-Security-Policy`, `HSTS`, `X-Frame-Options`,
`X-Content-Type-Options`; TLS config; open redirect patterns; CORS misconfig; directory
listing; verbose errors; `Server` banner leakage.

**6.4 Nuclei sweep:** `nuclei -t exposures -t misconfig -t default-logins -t cves -t tech
-t exposures -t fuzz` on all hosts → JSON + MD. This is the passive bug-net (finds CVEs,
default creds, exposure misconfigs automatically).

**6.5 Version → CVE map:** cross-reference versions against CVEs; add to `cve_map.txt`.

**Outputs:** `06-fingerprint/tech.txt`, `waf.txt`, `misconfig.txt`, `nuclei_results.*`,
`cve_map.txt`.

## Phase 7 — Organize Attack Surface (manual-hunt handoff)

**7.1 Merge** every phase's outputs into `all/` (single source of truth, deduped with `anew`).

**7.2 Build `all/graph.txt`:** host → (tech, paths, params, endpoints, secrets).

**7.3 Priority ranking (bounty-ordered), per host and globally:**

1. **Auth/ATO surfaces:** login, register, reset, OAuth/SSO, JWT, session endpoints.
2. **Object-ID endpoints (BOLA/IDOR):** params like `id`, `user`, `order`, `doc`, `file`,
   numeric/UUID keys → cross-tenant tests.
3. **SSRF surfaces:** URL fetch, image proxy, webhook, PDF generator, redirect followers.
4. **Upload / file handling:** multipart, `file=`, path traversal.
5. **Injection surfaces:** params reaching DB/search/sort (SQLi), command/SSI.
6. **XSS/reflection surfaces:** any input reflected; DOM sinks.
7. **Admin/debug/unauthenticated panels** (from Phase 5).
8. **Actionable secrets/leaks** found now (already proven or near-proven impact).
9. **Misconfigs / info disclosure / takeover candidates.**

**7.4 `map.md`:** per target — live hosts + status, tech stack + versions, endpoints
grouped by vuln class, admin panels, secrets, WAF notes, next-actions queue, and the
evidence trail pointer (`state.txt`). This file is the entry point for manual hunting.

## Never-Miss Invariants (audit checklist)

Run this audit after the pipeline; anything unsatisfied means the phase is INCOMPLETE:

- [ ] ≥3 subdomain sources used; output non-empty
- [ ] every domain DNS-validated; wildcard detected
- [ ] every unique IP port-scanned; http probed
- [ ] every live host directory-bruted + hidden-file checked
- [ ] historical AND live crawl both done; JS mined
- [ ] secret scan run over all harvested content (not just URLs)
- [ ] admin/panel list generated
- [ ] tech/WAF fingerprint recorded; nuclei sweep done
- [ ] `all/` merged; `map.md` + `state.txt` written

If a phase produced nothing, record WHY (≥3 attempts each) in `state.txt` — an empty result
is a documented result, never a silence.

## Operational Rules

- Non-destructive at all times during recon (no deletes, no writes to target DBs, no
  exploit beyond PoC reads). Destructive/irreversible actions require user confirmation.
- Respect rate limits; throttle large scans; use `curl-impersonate` profiles when a CDN/WAF
  blocks default TLS fingerprints.
- Do NOT bulk-copy real user data. PoCs use self-owned/benign data only.
- PII: never publish; anonymize in any report.
- Log everything to `logs/`; append every decision to `state.txt`.