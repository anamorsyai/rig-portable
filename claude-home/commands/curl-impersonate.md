---
description: Make HTTP requests with browser TLS impersonation (curl-impersonate). Use to bypass anti-bot detection or mimic real browsers.
---

Use curl-impersonate for requests that need to look like a real browser at the TLS/HTTP2 level.

Available browser presets (use the wrapper scripts, installed at /usr/local/bin):
- Chrome: curl_chrome99, curl_chrome100, curl_chrome101, curl_chrome104, curl_chrome107, curl_chrome110, curl_chrome116, curl_chrome119, curl_chrome120, curl_chrome123, curl_chrome124, curl_chrome131, curl_chrome133a, curl_chrome136, curl_chrome142, curl_chrome145, curl_chrome146, curl_chrome150
- Firefox: curl_firefox133, curl_firefox135, curl_firefox144, curl_firefox147
- Safari: curl_safari153, curl_safari155, curl_safari170, curl_safari172_ios, curl_safari180, curl_safari180_ios, curl_safari184, curl_safari184_ios, curl_safari260, curl_safari260_ios, curl_safari2601
- Edge: curl_edge99, curl_edge101
- Tor: curl_tor145

Usage: $ARGUMENTS

Example: curl_chrome136 -s https://example.com
Example with JSON output: curl_chrome150 -s -H 'Accept: application/json' https://api.example.com/data