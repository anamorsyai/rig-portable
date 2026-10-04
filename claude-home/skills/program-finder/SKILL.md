---
name: program-finder
description: Discover external self-hosted bug bounty programs NOT on major platforms (HackerOne, Bugcrowd, Intigriti). Deep dorking for obscure programs with rewards, sign-up access, free trials, APIs/SaaS, complex features, multi-tenant roles. Finds the hidden gems most hunters miss. Use when hunting for new targets, finding programs to test, or building a target pipeline.
---

# Program Finder — Deep Discovery of Self-Hosted Bug Bounty Programs

This skill finds bug bounty programs that most hunters never see: self-hosted, obscure, international, and platform-independent programs with real rewards and testable applications.

---

## Core Philosophy

**The best programs are NOT on HackerOne/Bugcrowd.** Companies that run their own programs:
- Don't pay platform fees (more budget for bounties)
- Are less tested (fewer hunters find them)
- Often have higher bounties (no platform cut)
- Respond faster (direct communication)
- Have simpler scope (less bureaucracy)

**Your job: find these before other hunters do.**

---

## Phase 1: Security Infrastructure Dorking

Most self-hosted programs announce themselves through standard security files.

### Google Dorks — security.txt & well-known
```
"inurl:security.txt" "bounty" OR "reward" OR "disclosure"
"inurl:/.well-known/security.txt" "reward"
"site:*.com inurl:security.txt "bounty"
"intitle:security.txt "responsible disclosure""
"inurl:/security "bug bounty" OR "vulnerability disclosure""
"filetype:txt "we offer bounties""
"filetype:txt "responsible disclosure policy""
"filetype:txt "vulnerability reward""
"filetype:txt "security@"" "contact"
"ext:txt "bug bounty" "terms""
```

### Program Pages (Direct Discovery)
```
"inurl:bug-bounty" OR "inurl:bugbounty" OR "inurl:bug_bounty"
"inurl:vulnerability-disclosure" OR "inurl:vdp"
"inurl:responsible-disclosure"
"inurl:security-policy" "reward" OR "bounty"
"inurl:security page" "report" OR "submit"
"intitle:bug bounty program"
"intitle:vulnerability disclosure"
"intitle:responsible disclosure policy"
"inurl:/security#tab" OR "inurl:/security/programs"
```

### Exclusion Dorks (Platform Filtering)
```
"bug bounty" -site:hackerone.com -site:bugcrowd.com -site:intigriti.com -site:yeswehack.com
"vulnerability disclosure" -site:hackerone.com -site:bugcrowd.com -site:intigriti.com
"responsible disclosure" -site:hackerone.com -site:bugcrowd.com -site:gitlab.com
"reward" "security" "report" -site:hackerone.com -site:bugcrowd.com -site:discord.com
```

---

## Phase 2: Terms & Legal Page Mining

Programs often mention rewards in their terms, legal pages, or policies.

### Dorks for Legal/Reward Mentions
```
"terms of service" "bounty" OR "reward" OR "disclosure"
"legal" "bug bounty" OR "vulnerability reward"
"privacy policy" "security" "report" OR "disclosure"
"acceptable use" "vulnerability" "report"
"security" "we reward" OR "we offer" OR "bounty program"
"inurl:terms "security" "reward""
"inurl:legal "bug bounty""
```

---

## Phase 3: GitHub/GitLab Source Discovery

Open-source projects and organizations often have security.md or bug bounty mentions.

### GitHub Dorks
```
"security.md" "bug bounty" OR "reward" OR "disclosure" org:
"security.md" "responsible disclosure" org:
"bug-bounty" filename:README
"bugbounty" filename:README
"vulnerability disclosure" filename:security
"reward" filename:security.md
"bounty" filename:security.md
```

### GitLab Dorks
```
"bug bounty" site:gitlab.com -site:hackerone.com
"security policy" site:gitlab.com "reward"
"responsible disclosure" site:gitlab.com
```

---

## Phase 4: Job Listing Intelligence

Companies hiring security engineers often have or are building bug bounty programs.

### Job Board Dorks
```
"bug bounty" "security engineer" hiring
"responsible disclosure" "security team" jobs
"vulnerability disclosure program" "we're hiring"
"security" "penetration testing" "bug bounty" career
"bug bounty program" "manage" "security" job
```

### LinkedIn/Job Dorks
```
"bug bounty" site:linkedin.com/jobs
"security program manager" "bug bounty" hiring
"responsible disclosure" site:indeed.com
```

---

## Phase 5: SaaS/API Discovery

Target SaaS platforms with APIs, trials, and complex features.

### SaaS-Specific Dorks
```
"inurl:pricing" "API" "documentation" "security"
"intitle:API documentation" "authentication" "OAuth"
"inurl:signup" "free trial" "API" "security"
"inurl:docs" "API reference" "authentication"
"swagger" OR "openapi" "security" "policy"
"inurl:api/v1" OR "inurl:api/v2" "authentication"
"GraphQL" "introspection" "security"
```

### Feature Complexity Indicators
```
"roles" "permissions" "admin" "organization" "team"
"multi-tenant" OR "workspace" OR "organization" "API"
"webhook" OR "integration" "API" "authentication"
"SSO" OR "SAML" OR "OAuth" "enterprise"
"role-based access" OR "RBAC" "documentation"
```

---

## Phase 6: Community & Social Discovery

Programs often get mentioned in forums, Reddit, Twitter, Discord.

### Forum/Community Dorks
```
"bug bounty" site:reddit.com -site:reddit.com/r/netsec "self-hosted"
"bug bounty" site:reddit.com "program" "reward"
"vulnerability disclosure" site:news.ycombinator.com
"bug bounty" site:forum "reward" "program"
"security" "disclosure" site:discord.gg OR site:discord.com
```

### Twitter/X Dorks
```
"bug bounty" site:twitter.com OR site:x.com "program" "reward"
"responsible disclosure" site:twitter.com OR site:x.com
"security" "bounty" site:twitter.com "new program"
```

---

## Phase 7: Alternative Platforms

These platforms host programs NOT on the Big 3.

### Platform Discovery
```
"open bug bounty" "program" site:openbugbounty.org
"immunefi" "bug bounty" site:immunefi.io
"hackenproof" "bug bounty" site:hackenproof.com
"antidote" "bug bounty" site:antidote.fi
"patchstack" "bug bounty" site:patchstack.com
"yeswehack" "bug bounty" site:yeswehack.com
"secfaults" "bug bounty" site:secfaults.com
"bugbountyhunter" "program" site:bugbountyhunter.com
```

---

## Phase 8: International & Non-English Discovery

Many programs exist in non-English markets and are rarely tested.

### International Dorks
```
"bug bounty" "belohnung" (German)
"bug bounty" "récompense" (French)
"bug bounty" "recompensa" (Spanish/Portuguese)
"bug bounty" "報酬" (Japanese)
"bug bounty" "보상" (Korean)
"bug bounty" "奖励" (Chinese)
"漏洞" "奖励" OR "赏金" (Chinese)
"安全" "漏洞" "报告" (Chinese)
"responsible disclosure" site:.de OR site:.fr OR site:.jp
"bug bounty" site:.co.uk OR site:.com.au OR site:.ca
```

### Regional Platform Dorks
```
"bug bounty" site:.de -site:hackerone.com -site:bugcrowd.com
"bug bounty" site:.jp -site:hackerone.com
"bug bounty" site:.cn -site:hackerone.com
"bug bounty" site:.ru -site:hackerone.com
"vulnerability" "reward" site:.br OR site:.mx OR site:.ar
```

---

## Phase 9: Certificate Transparency & Asset Discovery

Find related domains and subdomains for discovered programs.

### CT Log Dorks
```
# After finding a program domain, discover assets:
site:crt.sh "target.com"
site:crt.sh "%.target.com"

# Subdomain enumeration for discovered domains
subfinder -d target.com -silent
amass enum -passive -d target.com
```

---

## Phase 10: Email & Contact Pattern Discovery

Extract security contact emails for direct outreach.

### Email Pattern Dorks
```
"security@" "target.com"
"bugbounty@" "target.com"
"vulnerability@" "target.com"
"disclosure@" "target.com"
"security-team@" "target.com"
"infosec@" "target.com"
```

---

## Phase 11: Evaluation Matrix

For every discovered program, score it:

### Program Quality Score (1-10)

| Factor | Points | How to Check |
|--------|--------|-------------|
| **Has reward/bounty** | +3 | Explicit mention of $ amount or "reward" |
| **Has sign-up** | +2 | Registration page exists |
| **Has free trial** | +2 | Trial or freemium tier available |
| **Has API** | +2 | REST/GraphQL API documented |
| **Has app** | +1 | Web app with authentication |
| **Complex features** | +2 | Roles, orgs, teams, webhooks, integrations |
| **Multi-tenant** | +1 | Organizations, workspaces, teams |
| **Not platform-hosted** | +2 | Self-hosted program page |
| **Less tested** | +3 | No recent public writeups on the program |
| **Active development** | +1 | Recent commits, changelog, job postings |
| **International** | +1 | Non-English or non-US program |

### Minimum Threshold
Score ≥ 6/15 = worth testing immediately
Score ≥ 10/15 = high-priority target

### Auto-Evaluation Script
```bash
# After discovering a program, evaluate it:
# 1. Check for reward mention: grep -i "reward\|bounty\|compensation" <page>
# 2. Check for sign-up: curl -sI <url>/signup | head -1
# 3. Check for API docs: curl -sI <url>/docs OR <url>/api-docs OR <url>/swagger
# 4. Check for free trial: grep -i "free trial\|freemium\|free tier" <page>
# 5. Check for roles: grep -i "role\|permission\|admin\|organization" <page>
# 6. Score the program
```

---

## Phase 12: Pipeline Output Format

Every discovered program goes into a structured file:

```markdown
# Discovered Programs

## Program Name — https://example.com
- **URL**: https://example.com/security or /bug-bounty
- **Rewards**: Yes/No — $X-$Y range if known
- **Sign-up**: Yes/No — URL if available
- **Free trial**: Yes/No
- **API**: Yes/No — docs URL
- **App**: Yes/No — description
- **Features**: [list key features]
- **Roles**: [list roles/orgs if known]
- **Tech stack**: [if known]
- **Country**: [country]
- **Score**: X/15
- **Priority**: HIGH/MEDIUM/LOW
- **Status**: discovered | evaluated | testing | reported
- **Notes**: [any additional intel]
```

---

## Execution: Agent Dispatch Pattern

When `/find-programs` is triggered:

1. **@osint** runs Phase 1-8 dorks (batch by category)
2. **@intel** evaluates discovered programs (Phase 10-11)
3. **@recon** discovers assets for top-scored programs (Phase 9)
4. **@map** maps attack surface of priority programs
5. Results saved to `$PROJECT/programs/discovered.md`

### Parallel Execution
- All dorking phases run in parallel (independent search categories)
- Evaluation runs after dorking completes
- Asset discovery runs in parallel with evaluation
- Priority programs go straight to testing pipeline

---

## Key Principles

1. **Volume matters** — find 100 programs, test 10, report 1-2 Critical/High
2. **Speed matters** — find programs before other hunters do
3. **Obscurity is your friend** — less-tested = more vulnerabilities
4. **International = goldmine** — non-English programs are rarely hunted
5. **Self-hosted = higher bounties** — no platform fees = more budget
6. **SaaS/API = complex features** = more attack surface
7. **Free trial = instant access** — no waiting for approval
8. **Multi-tenant = IDOR goldmine** — role/tenant testing opportunities
