---
name: robust-file-write
description: Robust large file writing without JSON parse errors â€” use base64 encoding to bypass shell special character issues in tool calls. Use whenever write tool fails or content exceeds 2KB.
---

# Robust File Writing â€” No More JSON Parse Errors

## The Problem
- `write` tool fails for content >2KB (JSON truncation)
- `bash 'cat > file << EOF'` fails when content has quotes, backticks, $, or other special characters (JSON parse error in tool call)
- This kills skill rewrites, report writing, and any large file generation

## The Solution: Base64 Workflow

### Pattern A: Direct base64 write (for any content size)
1. Manually base64-encode the content in your response
2. Pass the base64 string (A-Za-z0-9+/= only â€” NO special JSON-breaking chars)
3. Command: `echo '<base64>' | base64 -d > /path/to/file`

```
Instead of: bash 'cat > file << ENDOFFILE ...content with $quotes and `backticks`... ENDOFFILE'
Do: echo 'QWxpY2Ugd2FzIGJlZ2lubm...' | base64 -d > /path/to/file
```

### Pattern B: Python base64 helper
If you need to generate base64 from stdin:
```
python3 -c "import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())" < input.txt
```

### Pattern C: Chunked writes (for extremely large content)
Write in 2KB chunks using the write tool with append:
1. Write chunk 1 with write tool
2. Append chunk 2 with write tool
3. Continue until complete

### Pattern D: Available helper commands
```
oc-b64write /path/to/file '<base64-content>'   # Write base64 to file
oc-file-helper write /path/to/file '<base64>'   # Same
oc-file-helper encode                           # Encode stdin to base64
```

## Quick Reference: Special Characters That Break JSON
| Char | Problem | Fix |
|------|---------|-----|
| " | Breaks JSON string | Use base64 |
| $ | Shell expansion | Use base64 |
| ` | Command substitution | Use base64 |
| ' | Shell quote | Use base64 |
| \ | Escape character | Use base64 |
