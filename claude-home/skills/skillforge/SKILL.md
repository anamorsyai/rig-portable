---
name: skillforge
description: Auto-generate and maintain opencode skills and commands from session discoveries. Detects novel techniques, repetitive workflows, validated tool configs, and bypass patterns â€” then creates SKILL.md files and command .md files automatically. Use when any agent discovers something reusable, when the same workflow is done manually 3+ times, or when a new pattern should be captured for future engagements.
---

# SkillForge â€” Auto Skill & Command Generator

## When to Trigger (Auto-Capture)

Any agent dispatches @skillforge when ONE of these conditions is true:

| Trigger | What to create | Example |
|---------|---------------|---------|
| Novel technique confirmed working on real target | SKILL.md | "Found Unicode WAF bypass for Cloudflare â€” works on 3 targets" |
| Same workflow done 3+ times manually | Command .md | "I keep manually polling Burp history then extracting tokens then saving to file" |
| Tool config produces significantly better results | SKILL.md update | "ffuf with -mc 200,301,302,403 + rate-limit 50rps finds 40% more" |
| Bypass pattern validated (WAF, auth, rate-limit) | SKILL.md | "Double URL encoding bypasses ModSecurity for SQLi" |
| Chain pattern confirmed | SKILL.md update to exploit-chains | "Open redirect + OAuth callback = full ATO" |
| Dead-end methodology worth preserving | SKILL.md (negative results) | "Tried 15 SQLi payloads against X framework â€” all blocked, here's what was tried" |
| Agent dispatch pattern that works well | Command .md | "@intake + @recon parallel start saves 2 minutes every engagement" |
| Target-specific insight applicable beyond current engagement | SKILL.md | "Jenkins 2.300+ has /script endpoint exposed by default" |

## Detection Patterns (What Agents Notice)

### Repetitive Workflow Detection
If an agent does these steps manually 3+ times in a session, it should dispatch @skillforge:
```
"I've now done: pull history â†’ extract tokens â†’ save to file â†’ hand to @accounts"
  â†’ This is a command: /burp-mine (already exists, or needs update)
```

### Novel Technique Detection
If an agent tries something that:
1. Isn't covered by existing skills
2. Actually works against the target
3. Would apply to other targets

â†’ Dispatch @skillforge to create a skill

### Config Optimization Detection
If an agent discovers a tool flag combination that:
1. Produces significantly better results than defaults
2. Isn't documented in tool-configs skill

â†’ Dispatch @skillforge to update tool-configs or create new skill

## Skill Generation Workflow

### Step 1: Dedup Check
```bash
# Read existing skill index
cat C:/Users/S3ck1llr/.claude/skills/INDEX.md

# Check if skill already exists
ls C:/Users/S3ck1llr/.claude/skills/<potential-name>/SKILL.md 2>/dev/null

# If exists â†’ UPDATE existing skill (append new variation, don't duplicate)
# If not exists â†’ CREATE new skill
```

### Step 2: Generate SKILL.md

```markdown
name: <lowercase-hyphenated-name>
description: <one sentence â€” front-loaded with trigger keywords, covers what AND when>

# <Title>

## When to Use
<Trigger conditions â€” when should an agent load this?>

## <Core Technique/Workflow>
<Concrete commands, payloads, steps â€” copy-paste ready>

## Variations
<Different approaches for different contexts>

## Common Failures
<What goes wrong and how to fix>

## Integration
<Which agents use this, what they hand off>
```

### Step 3: Verify
```bash
# File exists and non-empty
test -s C:/Users/S3ck1llr/.claude/skills/<name>/SKILL.md && echo "OK"

# Frontmatter valid
head -1 C:/Users/S3ck1llr/.claude/skills/<name>/SKILL.md | grep -q "^name:" && echo "FRONTMATTER OK"

# Name matches folder
FOLDER=$(basename C:/Users/S3ck1llr/.claude/skills/<name>)
SKILL=$(head -1 C:/Users/S3ck1llr/.claude/skills/<name>/SKILL.md | sed 's/name: //')
[ "$FOLDER" = "$SKILL" ] && echo "NAME MATCH"
```

### Step 4: Update INDEX.md
Append new row to `C:/Users/S3ck1llr/.claude/skills/INDEX.md`:
```
| <name> | <what it covers> | <date> |
```

## Command Generation Workflow

### Step 1: Detect Repetitive Pattern
Look for:
- Same 3+ step sequence done manually multiple times
- Same agent dispatch pattern triggered by same condition
- Same tool invocation with same flags across sessions

### Step 2: Generate Command File

```markdown
---
description: "<what it does â€” shown in TUI>"
agent: <which agent should execute>
subtask: true
---

<The prompt template with $ARGUMENTS support>

## Steps
1. <step 1>
2. <step 2>
...

## Rules
- <constraints>
- <quality bar>

Return: <what the agent should return>
```

### Step 3: Save to Commands Dir
```bash
# Global commands
~/.claude/commands/<name>.md

# Verify
test -s ~/.claude/commands/<name>.md && echo "COMMAND OK"
```

### Step 4: Test Invocability
The command should:
- Have a clear name (verb-noun pattern: `burp-mine`, `chain-check`, `vuln-test`)
- Accept `$ARGUMENTS` for target/scope
- Dispatch to the right agent via `agent:` frontmatter
- Return structured output

## Skill Update Workflow (Not Duplicate)

When a new variation of an existing skill is discovered:

1. Read the existing SKILL.md
2. Find the section where the new variation fits
3. Append the variation with a `### New Variation (date)` header
4. Update the description if the skill's scope expanded
5. Update INDEX.md "Covers" column
6. Do NOT create a new skill file

Example â€” updating `waf-bypass`:
```markdown
### Cloudflare Unicode Bypass (2026-07-22)
When standard bypass fails, try Unicode normalization:
- `%u0027` â†’ `'`
- `%u003B` â†’ `;`
- Full-width characters: `ï¼…` â†’ `%`
Confirmed working on: Cloudflare WAF + Apache 2.4
```

## Quality Bar (Enforce on Every Creation)

Every skill MUST have:
1. **Actionable** â€” specific commands, not "run sqlmap"
2. **Reusable** â€” applies to more than one engagement
3. **Verified** â€” based on confirmed technique, not theory
4. **Discoverable** â€” description front-loads search keywords

Every command MUST have:
1. **Clear trigger** â€” `/name` is obvious when to use it
2. **Agent assignment** â€” `agent:` frontmatter points to specialist
3. **Argument support** â€” `$ARGUMENTS` for target/scope
4. **Structured return** â€” agent knows what to output

## What NOT to Create

| Don't create | Why |
|-------------|-----|
| Skill for one-off command | No reusability |
| Skill for generic tool usage | Already covered by tool-configs |
| Skill for theoretical technique | Must be confirmed working |
| Command for single-step action | Overhead not worth it |
| Command that duplicates existing | Check ~/.claude/commands/ first |

## Auto-Capture Dispatch Rules

| Agent | Discovers | Dispatches @skillforge with |
|-------|-----------|---------------------------|
| @vuln | Novel payload/bypass | "Novel technique: <what>, confirmed on <target>, create skill for <class>" |
| @bizlogic | New workflow abuse | "New chain pattern: <what>, create/update exploit-chains skill" |
| oc-burp tooling | Token extraction pattern | "New token pattern: <regex>, add to oc-burp-tokens" |
| @scan | Optimal tool config | "Tool config: <tool> <flags> produces <improvement>, update tool-configs" |
| @exploit | Confirmed finding chain | "Chain confirmed: <components>, update exploit-chains skill" |
| @bac | Session management flaw | "Session pattern: <what>, add to auth-bypass-session skill" |
| @intel | Target-specific CVE exploit | "CVE exploit: <CVE> works via <vector>, create skill <name>" |
| @fixer | Tool repair solution | "Tool fix: <tool> <problem> solved by <solution>, update tool-configs" |
| Any agent | Same workflow 3rd time | "Repetitive workflow: <steps>, create command <name>" |

## Negative Results Log

Also capture what DIDN'T work â€” equally valuable:
```markdown
## Dead Ends (what was tried)
- SQLi on /api/search with UNION: blocked by WAF (Cloudflare)
- XSS in profile name with <script>: HTML-encoded
- IDOR on /api/users/{id}: returns same data for all IDs
```

This prevents other agents from re-testing dead hypotheses.
