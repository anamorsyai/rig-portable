---
name: disposable-email
description: Readable disposable email inboxes (emailnator) for OTP-based auth testing. Use when a test needs to receive verification codes, confirm signup/reset emails, or control an attacker-side inbox during account flows.
---

# Disposable Email (emailnator)

Goal: a disposable address you can BOTH submit into a target flow AND read the
OTP/link it receives. emailnator gives real-Gmail-backed addresses readable via API.

## Hard constraints (why naive attempts fail)

1. **Cloudflare protects emailnator.** Raw `curl` gets challenged (403/challenge HTML).
   ALL calls must originate from inside a Playwright browser page already past CF,
   via same-origin `fetch` (`credentials: 'include'`) on `https://www.emailnator.com`.
2. **Laravel CSRF header must be LOWERCASE**: `x-xsrf-token`. Uppercase/mixed-case
   `X-XSRF-TOKEN` → HTTP 419 Page Expired. This cost real time once — never vary it.
3. Token source: `XSRF-TOKEN` cookie, URL-decoded:
   ```js
   const c = document.cookie.split(';').find(x => x.trim().startsWith('XSRF-TOKEN='));
   const tok = decodeURIComponent(c.split('=').slice(1).join('=')); // value contains '=' padding
   ```
   Note `.slice(1).join('=')` — the base64 token itself contains `=`.

## Canonical call sequence (all verified working)

Run inside `page.evaluate` on an emailnator-origin tab. Common headers every call:

```js
headers: {
  'content-type': 'application/json',
  'x-xsrf-token': tok,
  'x-requested-with': 'XMLHttpRequest',
  'accept': 'application/json, text/plain, */*'
}
```

### 1. Mint an address
```js
const r = await fetch('/generate-email', { method:'POST', credentials:'include',
  headers, body: '{}' });
// read the shown value from the page input afterwards (input[readonly])
```
Addresses come out as dot-trick variants of Gmail inboxes emailnator controls
(e.g. `jame.sck.e.l.ley9.2.3@gmail.com`). Arbitrary `+alias` forms on their base
Gmail are also deliverable+readable (e.g. `sisatmp+ldquc@gmail.com`) — useful when
you need N distinct identities fast.

### 2. List inbox messages — body is a BARE STRING, full address
```js
const r = await fetch('/message-list', { method:'POST', credentials:'include',
  headers, body: JSON.stringify({ email: '<full-addr>@gmail.com' }) });
// → { messageData: [ { messageID: '<b64>', from, subject, time, ... } ] }
```
**Body format is asymmetric and matters (verified live 2026-08-25):**
- LIST works with `{"email":"full@addr"}` — bare string.
- `{"email":["full@addr"]}` array → HTTP 500 Server Error.
- Local-part-only string → HTTP 500 (their own /mailbox UI sends this; their bug,
  don't imitate it).
- `/generate-email` is the opposite: needs the ARRAY form
  (`{"email":["plusGmail","dotGmail","googleMail","domain"]}`); `{}` mints nothing.

### 3. Read a message body (SAME endpoint, add messageID)
```js
const r = await fetch('/message-list', { method:'POST', credentials:'include',
  headers, body: JSON.stringify({ email: '<full-addr>@gmail.com', messageID: mid }) });
// → HTML body; extract OTP with /\b\d{6}\b/g style match (take LAST = freshest)
```

## Session-loss recovery (seen live 2026-08-25)

emailnator rotates sessions; an old address can start returning 500 even though the
mailbox itself still receives mail. What WORKED to revive it:
1. Keep/drive ONE emailnator tab at any emailnator-origin page.
2. Retry `message-list` with the bare-string full-address form — old inboxes come
   back alive without re-minting (the ATK inbox revived after ~30 min of 500s).
3. Only if that truly fails, mint fresh (array form above) and read the new value
   from `input[readonly]`.

Never fire the API cross-origin from another site's tab (e.g. the B2C page) —
CORS/CF kills it silently. Keep ONE stable emailnator tab; drive it by switching
tabs, not by embedding calls in other pages' evaluate contexts.

## Hunt integration

- Mint TWO addresses up front: ATK (attacker-controlled) and VIC-proxy (readable
  stand-in when you must not touch real users). Record them in the engagement's
  evidence dir immediately (`inbox.md`) — addresses are session-bound and cheap to lose.
- OTP extraction pattern for Transavia-style mail:
  ```js
  const codes = html.match(/\b\d{6}\b/g); // take the LAST match = freshest code
  ```
- Poll list→body on a 5–8s loop; OTP mails land within ~15s of the trigger.
- Prefer reading via the API over clicking through the UI reader — the UI marks
  messages read and some senders only render once.
