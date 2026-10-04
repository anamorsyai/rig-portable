---
name: exposed-git-svn
description: Exposed .git/.svn source control disclosure — dump repo history, extract secrets and uncommitted changes. Use when you find /.git/ or /.svn/ reachable on the web root.
category: misconfig-exposure
---

# Exposed .git / .svn

## Detection
```powershell
# quick checks
$paths = @(".git/HEAD", ".git/config", ".git/index", ".svn/entries", ".svn/text-base/config.svn-base")
foreach ($p in $paths) { $code = curl.exe -s -o NUL -w "%{http_code}" "$U/$p"; Write-Host "$code $p" }
```
- 200 on `HEAD`/`config`/`index` = confirmed.

## Exploitation
1. **Full dump**: use git-dumper / githack tools (or manual object fetch) to reconstruct the repo.
2. **Extract from dump**: `.env`, `config.*`, `*secret*`, `.npmrc`, API keys in commit history (`git log -p`), hardcoded creds, internal hostnames, DB DSNs, signing keys.
3. **Uncommitted/deleted files**: `.git/lost-found`, stale objects → sensitive files removed from disk but present in history.
4. **.svn**: `.svn/entries` lists file paths; `.svn/text-base/<file>.svn-base` serves content → pull configs/DB backups.
5. **Chain**: leaked creds → internal systems; leaked API keys → external services.

## Tool Commands (Windows)
```powershell
# full dump (if git-dumper available)
# git clone "$U/.git/" dump 2>$null || python -m git_dumper "$U/.git/" dump
# extract secrets from dump
# git -C dump log --all -p | Select-String -Pattern 'api[_-]?key|password|token|secret|BEGIN.*PRIVATE' -CaseSensitive:$false
```

## Verification & Evidence
- Confirm ≥1 sensitive artifact recovered from the dump (secret/config/source) — that's the finding, not just the .git exposure.
- Save recovered artifact (redacted) + repro command.