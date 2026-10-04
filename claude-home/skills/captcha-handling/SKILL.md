---
name: captcha-handling
description: CAPTCHA detection, classification, and bypass — when to fight and when to move on. Covers reCAPTCHA, hCaptcha, Turnstile, FunCaptcha, text/math/audio CAPTCHAs, and honeypots.
---

# CAPTCHA Handling Protocol

## Golden rule
**Don't fight CAPTCHAs.** Identify, attempt quick bypass, if it fails — move on. Max 2 minutes per CAPTCHA encounter.

## Classification table

| Type | Signs | Bypass? | Action |
|------|-------|---------|--------|
| reCAPTCHA v2 | "I'm not a robot" checkbox | No | Don't fight — switch endpoint/flow |
| reCAPTCHA v3 | Invisible, score-based | Maybe | Try realistic behavior delays |
| hCaptcha | "I'm a human" checkbox | No | Don't fight — switch endpoint/flow |
| Turnstile | "Verify you are human", Cloudflare | Maybe | Try proper browser headers first |
| FunCaptcha | Visual puzzle (rotate/match) | No | Don't fight — switch endpoint/flow |
| Text CAPTCHA | Distorted text image | Yes | OCR: `tesseract captcha.png stdout` |
| Math CAPTCHA | Simple math question | Yes | Parse and answer |
| Audio CAPTCHA | Audio challenge | Maybe | Speech-to-text: `whisper` or API |
| Honeypot | Hidden input field | Yes | Just don't fill the hidden field |

## Bypass attempts (max 2 minutes)

### Text-based CAPTCHA
```bash
# Install tesseract if missing
command -v tesseract || apk add tesseract-ocr 2>/dev/null || \
  apt-get install -y tesseract-ocr 2>/dev/null || \
  pacman -S --noconfirm tesseract 2>/dev/null || \
  dnf install -y tesseract 2>/dev/null

# OCR the captcha image
tesseract captcha.png stdout 2>/dev/null
```

### Honeypot detection
```bash
# Check for hidden fields that bots fill
curl -s URL | grep -i 'type="hidden"'
# If found — just don't fill those fields, submit normally
```

### Turnstile / Cloudflare
```bash
# Sometimes auto-passes with proper browser headers
curl -s -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" \
     -H "Accept: text/html,application/xhtml+xml" \
     -H "Accept-Language: en-US,en;q=0.9" \
     URL
```

### reCAPTCHA v3 (low score)
```bash
# Add realistic delays between requests
sleep $((RANDOM % 5 + 3))
# Use realistic User-Agent
# Interact with the page before submitting
```

## When CAPTCHA blocks different flows

| Agent | What it's doing | CAPTCHA response |
|-------|----------------|-----------------|
| @accounts | Signup/login | Stop provisioning. Try different signup path. Log blocked in accounts.md |
| @bac | Live testing | Switch to different endpoint. Don't disrupt session |
| @vuln | Fuzzing/testing | WAF detected. Switch to passive techniques |
| @intake | Policy extraction | Hand to @intel for browser-based approach |
| @map | Endpoint mapping | Skip this endpoint, map others |

## What NOT to do
- Don't retry same CAPTCHA more than once
- Don't ask the human to solve it (unless it's the ONLY path to a critical finding)
- Don't use CAPTCHA-solving services (cost, unreliability)
- Don't spend > 2 minutes on any CAPTCHA bypass attempt
- Don't let one CAPTCHA block the entire engagement — always have an alternative path

## Logging format
```
CAPTCHA encountered: [type] at [URL]
Flow blocked: [what you were trying to do]
Attempted: [what you tried]
Result: [unsolvable / bypassed / switched to alternative]
Alternative: [what you did instead]
```
