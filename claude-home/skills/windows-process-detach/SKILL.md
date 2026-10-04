---
name: windows-process-detach
description: Windows PowerShell 5.1 hang traps when launching long-lived daemons (Start-Process redirect switches blocking on pythonw children) and when calling curl.exe with inline JSON/pipes — use the cmd start /b detach pattern and file-based bodies. Trigger when a launch script prints output but never returns to the prompt, an infinite spinner hangs a bash tool call, or curl from PowerShell corrupts/hangs on -d payloads.
---

# Windows Process Detach & CLI Hang Traps

Two Windows-specific traps cost real debugging time on this rig. Both have a single reliable
pattern that avoids them. Applies to any PowerShell 5.1 + curl.exe + pythonw daemon combo.

## Trap 1: Start-Process redirect switches hang on long-lived children

### Symptom
- A launcher script prints its success line ("PROXY UP (PID ...)") then **never returns** — the
  caller's prompt / spinner hangs forever even though the output was delivered.
- Wrapping the launcher in `Start-Job { ... }` + `Wait-Job -Timeout 60` reports `STILL HUNG`.
- In the bash tool, the call "responds then never stops" — the tool's pipe is held open by a
  descendant process.

### Root cause
PowerShell 5.1's `Start-Process` with `-RedirectStandardOutput` / `-RedirectStandardError`
(and especially `-RedirectStandardInput`) creates pipe/file handles and the parent waits on them
**until the spawned process exits**. When the child is a long-lived daemon (the zen-proxy, any
server), it never exits, so the call blocks forever. Compound factor: venv `Scripts\pythonw.exe`
is a *shim* that spawns a second generation (base interpreter pythonw) — two live processes, so
the hang persists even if the wrapper's process tree is inspected.

### The fix: `cmd /c start /b` detached launcher (battle-tested pattern)
Do NOT use `Start-Process` redirect switches for daemons. Put the real spawn in a small `.cmd`
and `start /b` it with file redirections, then `Start-Process` that `.cmd` with NO redirect
switches. The `.cmd` returns immediately; `start /b` breaks console/pipe handle inheritance.

`launch-<svc>.cmd` (the detached spawner):
```bat
@echo off
rem Fully-detached daemon launcher. `start /b` returns immediately and detaches;
rem redirections point the daemon's std handles at log files only.
start "" /b "C:\path\to\venv\Scripts\pythonw.exe" -m <pkg>.proxy.<cli> --config "C:\path\to\config.yaml" --port 4000 --request_timeout 60 > "C:\path\to\opt\<svc>\<svc>.log" 2> "C:\path\to\opt\<svc>\<svc>.err.log"
```

`start-<svc>.ps1` (the idempotent wrapper — this is what users call):
```powershell
$ErrorActionPreference = "Stop"
$base = "C:\path\to\rig"
# Already answering? Nothing to do.
try {
    $alive = Invoke-WebRequest -Uri "http://localhost:4000/health" -UseBasicParsing -TimeoutSec 3
    if ($alive.StatusCode -eq 200) { "PROXY ALREADY RUNNING (HTTP $($alive.StatusCode))"; exit }
} catch { }

# Detached spawn - NO redirect switches here (PS5.1 blocks forever on long-lived children).
Start-Process -FilePath "$base\launch-<svc>.cmd" -WindowStyle Hidden

# Bounded readiness poll.
$up = $false
for ($i = 0; $i -lt 60; $i++) {
    Start-Sleep -Seconds 1
    try {
        $r = Invoke-WebRequest -Uri "http://localhost:4000/health" -UseBasicParsing -TimeoutSec 2
        if ($r.StatusCode -eq 200) { $up = $true; break }
    } catch { }
}
if ($up) {
    $m = Invoke-WebRequest -Uri "http://localhost:4000/v1/models" -UseBasicParsing -TimeoutSec 3
    $pid4000 = (Get-NetTCPConnection -LocalPort 4000 -State Listen -ErrorAction SilentlyContinue).OwningProcess
    "PROXY UP (PID $pid4000, HTTP $($r.StatusCode), $($m.Content.Length) bytes models)"
} else { "PROXY FAILED TO START. Check logs:"; exit 1 }
```
Entry point `.cmd` wrapper (ExecutionPolicy cannot block `.cmd`):
```bat
@echo off
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0start-<svc>.ps1" %*
```

### Expected output
`PROXY UP (PID <real listener PID>, HTTP 200, <N> bytes models)` and the call **returns to the
prompt** within ~startup time (zen-proxy ~3-5s). PID must come from `Get-NetTCPConnection`, not a
`Start-Process -PassThru` result (the shim PID is the wrong generation).

### Verification it no longer hangs
```powershell
$job = Start-Job { & "C:\path\to\start-<svc>.cmd" }
if (Wait-Job $job -Timeout 60) { Receive-Job $job } else { Write-Output "STILL HUNG - fix incomplete" }
Remove-Job $job -Force
```

### Common failures
- Re-adding `-RedirectStandardInput` to the ps1 spawn: reintroduces the hang. Keep the ps1 spawn
  free of ALL redirect switches; file redirects live only in the `.cmd`.
- Two pythonw PIDs (venv shim + base) is NORMAL — don't double-kill and don't report the shim PID.
- Killing the listener PID with `Stop-Process` leaves the venv shim behind; kill by matching both,
  or just `Get-Process pythonw | Stop-Process` before a clean restart.

## Trap 2: curl.exe JSON bodies and pipes from PowerShell

### Symptom A — JSON mangled
`curl.exe -d '{"model":"x",...}'` from PowerShell → `{"error":"failed to unmarshal JSON: invalid
character 'm' looking for beginning of object key string"}`. PowerShell 5.1 mangles single-quoted
strings passed to native exes (quotes stripped before curl sees them).

### Fix A — file-based body
```powershell
$t = "C:\Users\S3ck1llr\AppData\Local\Temp\opencode"
Set-Content -Path "$t\body.json" -Value '{"model":"m","messages":[{"role":"user","content":"hi"}]}' -Encoding Ascii
curl.exe -s -m 60 -H "Content-Type: application/json" -H "Authorization: Bearer <KEY>" --data-binary "@$t\body.json" -o "$t\out.json"
Remove-Item "$t\body.json","$t\out.json" -Force -ErrorAction SilentlyContinue
```

### Symptom B — pipe hangs the tool
`curl.exe ... | python -c "..."` can hang the bash tool forever (native pipeline buffering /
handle inheritance). 

### Fix B — never pipe curl into python. Write to a file, then parse separately.
```powershell
curl.exe -s -m 40 -o "$t\out.json" -w "HTTP %{http_code} in %{time_total}s`n" "<URL>" ...
$d = Get-Content "$t\out.json" -Raw | ConvertFrom-Json   # PowerShell native, no pipe
# or: python -c ... reading "$t\out.json" (file path, not stdin)
```

### Expected output
Clean HTTP code line, then parseable output; the tool call returns promptly.

### Common failures
- Omitting `-o` and instead capturing stdout for big/streaming responses.
- `-m` (max time) too small for thinking models: gemini/groq free tiers can take 30-90s per turn.
  Start at `-m 90` for LLM endpoints, `-m 10` for local proxy health checks.

## Trigger conditions
- Any PowerShell launcher that must start a long-lived service/proxy/daemon.
- A bash tool call that prints output but never terminates (spinner forever).
- A curl call from PowerShell that returns malformed-JSON errors or hangs.
