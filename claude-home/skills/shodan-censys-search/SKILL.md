---
name: shodan-censys-search
description: "Shodan/Censys hunting: find exposed services, open ports, default creds, certificates, IoT devices."
---

# Shodan/Censys Search

## Shodan Queries
- hostname:target.com
- org:"Company Name"
- ssl.cert.subject.cn:"target.com"
- http.title:"Dashboard"
- port:22,80,443,8080 hostname:target.com

## Censys Queries
- services.tls.certificates.leaf.names: target.com
- services.http.response.html_title: Dashboard
- autonomous_system.organization: "Company Name"

## Default Credentials
- Check all management interfaces
- Try admin:admin, admin:password, root:root
- Try vendor-specific defaults

## Certificate Intelligence
- Find all subdomains via cert transparency
- Track certificate changes
- Identify infrastructure providers

## Service Discovery
- Open ports and services
- Software versions
- Default pages (admin panels, dashboards)
- Exposed APIs
