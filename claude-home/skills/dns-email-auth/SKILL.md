---
name: dns-email-auth
description: DNS & email authentication audit — SPF/DKIM/DMARC misconfig, subdomain takeover vectors, open mail relays, SPF bypass (email spoofing). Use during recon on the target's domains and MX records.
category: misconfig-exposure
---

# DNS & Email Authentication

## Detection
```powershell
Resolve-DnsName -Name "$D" -Type TXT | Where-Object {$_.Strings -match 'spf|dkim|dmarc'}
Resolve-DnsName -Name "_dmarc.$D" -Type TXT
Resolve-DnsName -Name "default._domainkey.$D" -Type TXT
# nmap equivalent for open relay check (install nmap if needed)
```
- Flag: missing/weak SPF (`+all`, `~all` with missing include), no DMARC or `p=none`, no DKIM selector.

## Exploitation
1. **Spoofing**: DMARC absent/`p=none` → craft spoofed mail from `$D` (prove with a test to a mailtrap in-scope mailbox).
2. **SPF `+all`/`?all`** → full spoof scope.
3. **Subdomain takeover**: CNAME/NS to a dangling external host (see subdomain-takeover-detection) — combine for spoofing + phishing surface.
4. **Open relay**: MX accepts mail to arbitrary domains → spam source + reputation damage.
5. **DKIM selector enumeration**: weak selectors leak via GitHub/automated tools.

## Tool Commands (Windows)
```powershell
# SPF record
Resolve-DnsName -Name "$D" -Type TXT | Where-Object {$_.Strings -match 'v=spf1'}
# DMARC record
Resolve-DnsName -Name "_dmarc.$D" -Type TXT
# DKIM selector
Resolve-DnsName -Name "default._domainkey.$D" -Type TXT
# Note: swaks not available by default on Windows, use PowerShell or install swaks
# For authorized spoof test, use PowerShell Net.Mail.SmtpClient or external tool
```

## Verification & Evidence
- Proof = spoofed mail delivered to an authorized test inbox, or takeover vector, or open relay SMTP session log.
- Record SPF/DKIM/DMARC records verbatim + delivery proof.