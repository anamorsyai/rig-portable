---
name: vhost-enum
description: Virtual host enumeration — discover hidden vhosts (dev/staging/admin) by fuzzing the Host header against a known IP. Use when the same IP hosts multiple apps or you need hidden admin surfaces.
category: recon
---

# Virtual Host Enumeration

## Detection
- Multiple apps on one IP: `Host` header decides routing.
- Public vhosts may not be in DNS (only resolvable internally) but still route if the Host header matches.

## Exploitation
1. **Fuzz vhosts**: send each candidate Host against the IP and compare response sizes; the "default" host response (unknown Host) is the baseline — different size/status = real vhost.
2. **Known prefixes**: `dev`, `staging`, `admin`, `api`, `internal`, `test`, `jenkins`, `gitlab`, `grafana`, `prometheus`.
3. **Wordlist**: combine subdomain wordlist with `$D` and `$D` suffixes (`admin.$D`, `$D-admin`).
4. **Triple-auth bypass**: admin vhost found → test weak auth, no auth, default creds (authorized scope only).
5. **Cache poisoning on vhost**: if CDN keys on URL only, vhost content can be poisoned for the wrong host.

## Tool Commands (Windows)
```powershell
# enumerate IP A records then fuzz Host header
$IP = (Resolve-DnsName -Name "$D" -Type A | Select-Object -First 1).IPAddress
# baseline size (requires ffuf or manual)
$defaultSize = (curl.exe -s -H "Host: nonexistent.$D" "https://$IP" | Measure-Object -Character).Characters

# fuzz manually with wordlist
@vhosts.txt | ForEach-Object {
    $vhost = "$_.$D"
    $code = curl.exe -s -o NUL -w "%{http_code}" -H "Host: $vhost" "https://$IP"
    $len = (curl.exe -s -H "Host: $vhost" "https://$IP" | Measure-Object -Character).Characters
    if ($len -ne $defaultSize) { Write-Host "VHOST FOUND: $vhost (size: $len, code: $code)" }
}
```

## Verification & Evidence
- Real vhost = distinct response (size/status/content) not matching the default host.
- Follow up on admin/internal vhosts for auth findings; document Host + IP + evidence.