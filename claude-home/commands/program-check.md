---
description: Quick scope and program check. Lists targets, scope, and rules of engagement.
---

Analyze the bug bounty program for: $ARGUMENTS

## Steps

1. **Find the program page**
```bash
# Search for the bug bounty program
curl -s "https://$ARGUMENTS/.well-known/security.txt" 2>/dev/null | grep -i "contact\|policy\|scope"
```

2. **Check scope**
- What domains/subdomains are in scope?
- What URL patterns are explicitly excluded?
- Are APIs in scope?
- Any specific vulnerability types excluded?

3. **Rules of Engagement**
- Rate limiting rules?
- Authentication requirements?
- Data handling policies?
- Safe harbor provisions?

4. **Reward Structure**
- Severity ratings and payouts
- Any bonus scopes or focus areas?
- Recent valid findings for hints

Provide a structured summary of:
- In-scope assets
- Out-of-scope assets
- Key rules to follow
- Recommended attack vectors based on scope
- Any recent interesting findings worth investigating
