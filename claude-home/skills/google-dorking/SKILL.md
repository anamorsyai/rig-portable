---
name: google-dorking
description: "Google dorking: advanced operators for sensitive files, login panels, exposed configs, backup files."
---

# Google Dorking

## Sensitive Files
- site:target.com filetype:sql
- site:target.com filetype:log
- site:target.com filetype:bak
- site:target.com filetype:env
- site:target.com filetype:yml
- site:target.com filetype:xml

## Login Panels
- site:target.com inurl:login
- site:target.com inurl:admin
- site:target.com intitle:"index of"
- site:target.com intitle:"login"

## Configuration Files
- site:target.com filetype:json
- site:target.com filetype:conf
- site:target.com filetype:config
- site:target.com filetype:ini

## Directory Listings
- site:target.com intitle:"index of" "parent directory"
- site:target.com intitle:"index of" .git
- site:target.com intitle:"index of" .env

## Backup Files
- site:target.com filetype:zip
- site:target.com filetype:tar.gz
- site:target.com filetype:sql.gz
- site:target.com filetype:bak

## Exposed Info
- site:target.com "password"
- site:target.com "api_key"
- site:target.com "secret"
- site:target.com "internal"
- site:target.com "confidential"

## Code Leaks
- site:github.com "target.com" password
- site:github.com "target.com" api_key
- site:pastebin.com "target.com" 
