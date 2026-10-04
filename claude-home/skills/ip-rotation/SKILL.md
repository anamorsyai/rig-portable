---
name: ip-rotation
description: Windows IP rotation via Tor for rate-limit and WAF evasion. Drives tor-rotate.ps1 (SIGNAL NEWNYM), session proxy env, and the system-level Windows proxy (PAC). Auto-detect 429/403 and rotate with run-rotate. Invoke when a target starts 429/403/timeout-ing, or when IP-based identity matters.
---

# IP Rotation & Rate-Limit / WAF Evasion (Windows rig)

The Alpine `oc-rotate-ip` script does NOT exist on this rig. The Windows equivalent is
`tor-rotate.ps1` — a headless Tor daemon with on-demand exit-IP rotation. Every command below is
real and smoke-tested on this machine.

## Primary Command

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File C:/Users/S3ck1llr/bughunting-rig/hunting/tools/tor-rotate.ps1 <sub>
```

Also reachable via the dashboard/CLI:
- `claude-tools tor <sub>` (passthrough, default `status`)
- `/claude-tools` slash-command menu -> IP ROTATE section
- interactive `claude-tools` dashboard -> "Tor / IP rotate" (option 3)

Subcommands:

| Sub | Effect |
|-----|--------|
| `start` | Launch headless Tor daemon (idempotent; socks 127.0.0.1:9050, ctrl 9051) |
| `status` | Daemon state, circuit, uptime, current exit IP |
| `rotate` | ONE-CLICK: SIGNAL NEWNYM, verify exit IP changed (8 x 5s poll) |
| `run-rotate <cmd>` | Run a command through Tor; on 429/403/timeout auto-rotate + retry (up to `$env:TOR_ROTATE_MAX`, default 3) |
| `proxy-on` / `proxy-off` | Set/clear session env `ALL_PROXY/HTTPS_PROXY/HTTP_PROXY=socks5h://127.0.0.1:9050` (this shell only) |
| `sysproxy-on` | SYSTEM-LEVEL: Windows system proxy -> Tor via PAC file (all apps: browsers, everything that honors WinINET) |
| `sysproxy-off` | Clear system proxy (direct) |
| `sysproxy-status` | System proxy state, PAC URL, Tor state, exit IP |
| `stop` | Stop the rig Tor daemon |

## Auto-Detect + Auto-Rotate (429 AND 403)

Wrap any HTTP tool to auto-detect rate-limit/WAF signals, rotate the circuit, and retry:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File C:/Users/S3ck1llr/bughunting-rig/hunting/tools/tor-rotate.ps1 run-rotate curl.exe -sk https://target.com/api/users
powershell -NoProfile -ExecutionPolicy Bypass -File C:/Users/S3ck1llr/bughunting-rig/hunting/tools/tor-rotate.ps1 run-rotate ffuf -u https://target.com/FUZZ -w dirs.txt
```

`run-rotate` injects `ALL_PROXY/HTTPS_PROXY/HTTP_PROXY=socks5h://127.0.0.1:9050` into the child
(cmd.exe /c), runs it, scans output for the WAF pattern
`429|403|too many requests|rate.?limit|forbidden|timeout|connection reset|maximum redirects`, and on a
match runs `rotate` and retries. `NO_PROXY=localhost,127.0.0.1` is always set so the rig/zen-proxy/Burp
stay reachable.

Increase retries for stubborn targets:
`$env:TOR_ROTATE_MAX = 5` before invoking, or in the launching session.

## Tool Proxy Flags (when you drive a tool directly)

| Tool | Flag |
|------|------|
| curl.exe | `--proxy socks5h://127.0.0.1:9050` (socks5h = remote DNS, hides DNS from the local resolver) |
| ffuf | `-x socks5h://127.0.0.1:9050` |
| nuclei | `-proxy socks5h://127.0.0.1:9050` |
| httpx | `-proxy socks5h://127.0.0.1:9050` |
| sqlmap | `--proxy=socks5h://127.0.0.1:9050` |
| dalfox | `--proxy socks5h://127.0.0.1:9050` |
| gobuster | `--proxy socks5h://127.0.0.1:9050` |
| python/requests | `proxies={'http':'socks5h://127.0.0.1:9050','https':'socks5h://127.0.0.1:9050'}` (needs `pysocks`) |

Always prefer `socks5h://` (remote DNS) over `socks5://` so DNS lookups also exit through Tor.

## System-Level Rotation (Windows system proxy)

When you want the WHOLE desktop (browser, any app honoring the system proxy) to ride Tor:

1. `sysproxy-on` — writes `etc/tor/tor-proxy.pac` and sets
   `HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings` `AutoConfigURL` +
   `ProxyEnable=1`, then force-refreshes WinINet via `InternetSetOption` so running apps pick it up.
2. PAC logic: private ranges (10/8, 172.16/12, 192.168/16, 127/8, 169.254/16, `*.local`, single-label
   hosts) return DIRECT; everything else goes `SOCKS 127.0.0.1:9050` (with DIRECT fallback).
3. Verify with `sysproxy-status` (shows exit IP). Rotate with `rotate` while it is on.
4. `sysproxy-off` when done — never leave it on: it routes the whole desktop through Tor.

## Rotation Strategy

- **Continuous rotation**: if the new IP is still blocked, `run-rotate` rotates again (up to the
  max). `rotate` alone returns exit 2 if Tor reuses the same exit — rotate again.
- **Never stop testing**: if every rotation is blocked, fall back to passive recon, a different
  technique, or a different route (Burp proxy for localhost-bound tools) — a block is a bypass
  problem, not a stop.
- **Session proxy vs system proxy**: `proxy-on` for a single command shell (tools you launch get the
  env vars); `sysproxy-on` for GUI apps/browsers that only honor the Windows system proxy.

## Logging

Log rotation events to the rig event log:
`python C:/Users/S3ck1llr/bughunting-rig/hunting/tools/rig.py event TOOL_RUN "rotate <target> — reason: 429/403/manual"`.
