---
name: tls-ssl-audit
description: TLS/SSL configuration audit — weak protocols, cipher issues, certificate problems, heartbleed-class configs. Use as a low-severity/hygiene check on every host; rarely a standalone bounty finding unless paired with MITM impact.
category: misconfig-exposure
---

# TLS/SSL Audit

## Detection
```powershell
# requires openssl in PATH
openssl s_client -connect "$H:443" -servername "$H" < NUL 2>$null | openssl x509 -noout -dates -subject -issuer
# nmap alternative
nmap --script ssl-enum-ciphers -p 443 "$H"
```
- Flags: TLS 1.0/1.1 enabled, export ciphers, RC4/3DES, weak DH (≤1024), cert expiry/self-signed/wrong-hostname, missing SAN.

## Exploitation
1. **Downgrade**: TLS1.0/1.1 accepted → BEAST/POODLE-era attacks; report only if modern clients get downgraded.
2. **Weak cipher**: 3DES/RC4 → practical MITM decryption (rarely exploitable in modern stacks).
3. **Cert issues**: expired/self-signed → trust-chain MITM; wrong hostname cert → phishing surface.
4. **CRIME/BREACH**: compression enabled + attacker-influenced secrets in body (rare; requires XSS-reflected secret near other reflected content).
5. **Chain**: no HSTS + TLS1.0 → full downgrade + cookie theft on shared network.

## Tool Commands (Windows)
```powershell
nmap -sV --script ssl-enum-ciphers,ssl-cert,ssl-heartbleed -p 443 "$H" -oN tls.log
openssl s_client -tls1 -connect "$H:443" < NUL | Select-Object -First 5   # TLS1.0 check
echo | openssl s_client -connect "$H:443" -cipher 'EXPORT:ALL' 2>$null | Select-String -Pattern cipher -CaseSensitive:$false
```

## Verification & Evidence
- Hygiene findings usually informational; attach to a chained MITM/cert-theft scenario for severity.
- Capture openssl/nmap output in evidence.