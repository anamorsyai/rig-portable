---
name: skill-builder
description: Create new opencode skills during a session when a novel technique, tool config, bypass pattern, or refined workflow is discovered and should be reusable.
---
---

# Skill Builder

Create reusable skills on-the-fly when the engagement produces something worth preserving.

## When to Create a Skill

Trigger conditions:
- A novel technique or payload chain is confirmed working against a real target
- A tool configuration produces significantly better results than defaults
- A bypass pattern (WAF evasion, auth bypass, rate-limit dodge) is validated
- A workflow refinement saves time over the existing approach
- @intel surfaces a target-specific insight applicable beyond the current engagement

Do NOT create skills for:
- One-off commands with no reusability
- Generic tool usage already covered by existing skills
- Theoretical techniques not yet confirmed working

## SKILL.md Format (Exact)

```markdown
name: <skill-name>
description: <one sentence covering what AND when to trigger it, front-loaded with keywords>
---

# <Title>

<Body in markdown — actionable, not documentary>
```

### Frontmatter Rules
- `name`: lowercase, hyphen-separated, max 64 chars, MUST match folder name
- `description`: one sentence, front-loaded with trigger keywords so the matcher fires correctly
- Second line is the description, third line is blank, fourth line starts the body

## Where to Save

| Scope | Path |
|-------|------|
| Global (all projects) | `C:/Users/S3ck1llr/.claude/skills/<name>/SKILL.md` |
| Project-local | `<project-root>/.claude/skills/<name>/SKILL.md` |

Project-local skills override global ones when names collide.

## Naming Conventions

- Lowercase letters and hyphens only: `sqlmap-tamper-guide`, not `SQLMap Tamper Guide`
- Descriptive but concise: `waf-evasion-xss` not `bypass`
- Verb-noun when it's an action: `chain-idor-priv-escal`
- Noun when it's a reference: `cors-misconfiguration`

## Quality Bar

Every skill MUST include:
1. **Trigger conditions** — when should the system load this?
2. **Concrete commands** — copy-paste ready, not "run sqlmap"
3. **Flags and options explained** — why those specific flags
4. **Expected output** — what success looks like
5. **Common failures** — what goes wrong and how to fix it

## Verification After Creation

After writing the file, verify:
```bash
# File exists and is non-empty
test -s C:/Users/S3ck1llr/.claude/skills/<name>/SKILL.md && echo "OK" || echo "MISSING"

# Frontmatter is valid (first line starts with 'name:')
head -1 C:/Users/S3ck1llr/.claude/skills/<name>/SKILL.md | grep -q "^name:" && echo "FRONTMATTER OK" || echo "BAD FRONTMATTER"

# Name matches folder name
FOLDER_NAME=$(basename C:/Users/S3ck1llr/.claude/skills/<name>)
SKILL_NAME=$(head -1 C:/Users/S3ck1llr/.claude/skills/<name>/SKILL.md | sed 's/name: //')
[ "$FOLDER_NAME" = "$SKILL_NAME" ] && echo "NAME MATCH" || echo "NAME MISMATCH"
```

## Integration with Workflow

Skill creation is a natural checkpoint activity. After a confirmed finding or refined technique:
1. Decide if it's reusable (see "When to Create")
2. Write the SKILL.md with the exact format
3. Verify it (commands above)
4. Log the creation in the phase's STATUS.md
5. Continue the engagement — don't stop flow to over-engineer the skill
