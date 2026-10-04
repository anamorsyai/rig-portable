---
name: context-management
description: Intelligent context system for long bug bounty sessions â€” session context files, auto-compaction rules, cross-session handoff, context-aware dispatching, intelligent summarization, checklist templates.
---

# Context Management â€” Intelligent Context for Long Sessions

## 1. Core Problem

Bug bounty sessions can span hours, days, or weeks. Without a context management system:
- Agents re-test already-ruled-out hypotheses (wasted quota)
- Agents lose track of active leads (missed chains)
- Findings from the first day are forgotten by the third day (duplicate reports dropped)
- The orchestrator dispatches agents to dead-end work (time wasted)
- Session state is lost on crash/interrupt (progress reset)

The context system solves all of these. It is NOT optional â€” every agent MUST maintain and update context.

## 2. Session Context File Format

Every agent contributes to `<project-root>/tmp/session-context.md`:

```markdown
# Session Context â€” {TARGET}

## Active Leads
### {Finding-ID}: {Vuln Class} on {Endpoint}
- Status: {confirmed / testing / queued / blocked}
- Hypothesis: {one line}
- Last action: {what was tried}
- Next action: {what to do next}
- Chains with: {other finding IDs}

## Ruled Out (keep last 10)
- {what was tried} â€” {why it failed}

## Pending Tasks
- @agent: {task description}

## Key Context
- Tech stack: {Django/Node/Rails/etc} + {PostgreSQL/Mongo/etc} + {WAF/CDN}
- Auth: {JWT/OAuth/SAML} â€” {what's been tested on it}
- Accounts: {active identities with roles}
- Tenants: {what's been provisioned}

## Next Steps (Priority Order)
1. {highest priority action}
2. {next action}
3. {next action}
```

## 3. Session Context Writer Script

```bash
#!/bin/bash
# Save this as /usr/local/bin/oc-context
# Usage: oc-context update|status|compact|snapshot

SESSION_FILE="tmp/session-context.md"

case "${1:-status}" in
  update)
    echo "[*] Updating session context..."
    # Will be called by agents with context data
    cat > "$SESSION_FILE"
    ;;
  status)
    if [ -f "$SESSION_FILE" ]; then
      echo "=== Session Context ==="
      grep -A3 "^## Active Leads" "$SESSION_FILE" 2>/dev/null
      echo "..."
      grep -A3 "^## Next Steps" "$SESSION_FILE" 2>/dev/null
    else
      echo "No active session context found"
    fi
    ;;
  compact)
    if [ -f "$SESSION_FILE" ]; then
      LINES=$(wc -l < "$SESSION_FILE")
      if [ "$LINES" -gt 200 ]; then
        echo "[!] Context too large ($LINES lines), compacting..."
        # Save full version
        cp "$SESSION_FILE" "${SESSION_FILE}.archive"
        # Compact: keep only active leads, next steps, key context
        awk '
          /^## Active Leads/,/^## / {if (!/^## / || /^## Active Leads/) print}
          /^## Key Context/,/^## / {if (!/^## / || /^## Key Context/) print}
          /^## Next Steps/,0 {print}
        ' "${SESSION_FILE}.archive" > "$SESSION_FILE"
        echo "[*] Compacted to $(wc -l < "$SESSION_FILE") lines"
      fi
    fi
    ;;
  snapshot)
    if [ -f "$SESSION_FILE" ]; then
      cp "$SESSION_FILE" "$SESSION_FILE-$(date +%Y%m%d-%H%M)"
      echo "[*] Snapshot saved"
    fi
    ;;
esac
```

## 4. Auto-Compaction Rules

### Priority-Based Content Retention

```python
# Context compaction priorities â€” every agent follows these
CONTEXT_PRIORITY = {
    # ALWAYS KEEP â€” these are mission-critical
    "HIGH": [
        "Active finding hypotheses and status",
        "Confirmed findings (F-number, severity, endpoint, impact)",
        "Account/token/session data currently in use",
        "Current phase and next steps",
        "Blocker notes",
        "Active chain relationships (F001 <-> F003)",
    ],
    # KEEP IF SPACE permits
    "MEDIUM": [
        "Pending scan results",
        "Partial recon data (counts, not full lists)",
        "Ruled-out hypotheses (summary only, 1-2 lines each)",
        "Tool configuration notes",
        "IP/user-agent rotation state",
    ],
    # REMOVE aggressively
    "LOW": [
        "Raw tool output (reference file paths instead)",
        "Successful test confirmations (already in STATUS.md)",
        "Duplicate findings across endpoints",
        "Long payload lists (reference skill/library instead)",
        "Historical data from previous sessions",
    ],
}
```

### Compaction Decision Tree

When an agent encounters a context file that's too large (>200 lines):

```python
def compact_context(filepath):
    """Reduce context to essential elements only"""
    with open(filepath) as f:
        lines = f.readlines()
    
    if len(lines) <= 200:
        return  # No compaction needed
    
    important_sections = []
    current_section = None
    
    for line in lines:
        if line.startswith("## Active Leads"):
            current_section = "keep"  # Always keep
        elif line.startswith("## Ruled Out"):
            current_section = "summary"  # Keep only last 10
        elif line.startswith("## Next Steps"):
            current_section = "keep"
        elif line.startswith("## Key Context"):
            current_section = "keep"
        elif line.startswith("##"):
            current_section = "trim"
        
        if current_section == "keep":
            important_sections.append(line)
        elif current_section == "summary":
            # Only keep if we haven't already got 10 ruled-out entries
            if line.startswith("- ") and important_sections.count("- ") < 10:
                important_sections.append(line)
        elif current_section == "trim":
            # Skip removed sections
            pass
    
    # Save compacted version
    with open(filepath, 'w') as f:
        f.writelines(important_sections)
```

## 5. Cross-Session Context Handoff

### Session End Protocol

```bash
# Every agent runs this at end of session
oc-context snapshot

# Copy to permanent location
cp tmp/session-context.md "reports/context-$(date +%Y%m%d-%H%M).md"

# Save finding graph
if [ -f vuln/findings-graph.md ]; then
  cp vuln/findings-graph.md "reports/findings-graph-$(date +%Y%m%d-%H%M).md"
fi
```

### Session Resume Protocol

```bash
# At session start, every agent:
if [ -f tmp/session-context.md ]; then
  echo "[*] Previous session context found:"
  echo "=== Active Leads ==="
  grep -A1 "^### " tmp/session-context.md | grep -v "^--$"
  echo "=== Next Steps ==="
  grep "^[0-9]" tmp/session-context.md | head -5
else
  echo "[*] No previous session â€” starting fresh"
fi
```

## 6. Context-Aware Agent Dispatching

The orchestrator uses context to avoid wasted dispatches:

```python
def should_dispatch(agent_type, target):
    """Decision function â€” check context before dispatching"""
    context = load_session_context()
    
    # Don't dispatch to ruled-out targets
    if target in context["ruled_out"]:
        return False, f"Already ruled out: {context['ruled_out'][target]}"
    
    # Don't dispatch if already in progress
    if target in context["in_progress"]:
        return False, f"Already being tested by {context['in_progress'][target]}"
    
    # Don't dispatch if it depends on an incomplete task
    for dependency in get_dependencies(agent_type, target):
        if dependency not in context["completed"]:
            return False, f"Dependency not met: {dependency}"
    
    return True, "OK to dispatch"
```

## 7. Finding Cross-Reference Graph

Maintain an always-updated finding cross-reference:

```markdown
## Finding Graph (maintained by every agent)

### F001 â€” SQLi on /api/search
- **Status**: Confirmed (blind, time-based extraction working)
- **Chains with**: F003 (SSRF) â€” SQLi can read source, find SSRF endpoint creds
- **Chains with**: F002 (IDOR) â€” SQLi can extract admin user IDs for IDOR
- **Propagates to**: /api/filter, /api/sort, /api/category (same query builder)
- **Unlocks**: Admin panel endpoints (source code reveals internal routes)

### F002 â€” IDOR on /api/users/{id}/profile
- **Status**: Confirmed (can read any user's name, email, avatar URL)
- **Chains with**: F001 (SQLi) â€” use SQLi to extract next ID to target
- **Propagates to**: /api/users/{id}/settings, /api/users/{id}/billing
- **Impact**: Email and name of any user exposed. Not enough for ATO alone.

### F003 â€” SSRF on /api/fetch?url=
- **Status**: Testing (dns callbacks received, HTTP not yet confirmed)
- **Chains with**: F001 (SQLi) â€” needs SQLi to find internal format
- **Next test**: Try cloud metadata endpoints (169.254.169.254)
```

## 8. Intelligent Summarization

When context exceeds any agent's ability to process efficiently:

```python
def intelligent_summarize(context_file):
    """Call @writeup-summarizer to compress context"""
    with open(context_file) as f:
        content = f.read()
    
    word_count = len(content.split())
    
    if word_count > 2000:
        # Automatically trigger summarization
        print("[!] Context exceeds 2000 words â€” triggering summarization")
        
        # Save full version
        import shutil
        shutil.copy(context_file, f"{context_file}.full")
        
        # Compact aggressively
        lines = content.split('\n')
        compressed = []
        for line in lines:
            # Keep section headers always
            if line.startswith('#'):
                compressed.append(line)
            # Keep active findings (status: confirmed/testing)
            elif '**Status**' in line and 'Confirmed' in line:
                compressed.append(line)
            elif '**Status**' in line and 'testing' in line.lower():
                compressed.append(line)
            # Keep chain relationships
            elif 'Chains with' in line:
                compressed.append(line)
            # Keep next steps
            elif line.strip() and line[0].isdigit():
                compressed.append(line)
        
        with open(context_file, 'w') as f:
            f.write('\n'.join(compressed))
        
        print(f"[*] Compressed from {len(lines)} to {len(compressed)} lines")
```

## 9. Context Freshness Checks

```bash
# Check how old context is
if [ -f tmp/session-context.md ]; then
  AGE=$(($(date +%s) - $(stat -c %Y tmp/session-context.md)))
  HOURS=$((AGE / 3600))
  if [ $HOURS -gt 4 ]; then
    echo "[!] Context is $HOURS hours old â€” may be stale"
  fi
fi
```

## 10. Integration with Checkpoint System

Every `oc-checkpoint` call automatically updates context:

```bash
# When an agent reports a finding, context is updated
oc-checkpoint finding sqli high "SQLi on /api/search" 
# This should also update tmp/session-context.md:
# - Add F001 to Active Leads
# - Set its status to "confirmed"

# When an agent rules out a hypothesis:
oc-checkpoint agent vuln done "XSS on /api/search: all params HTML-encoded"
# This should also update tmp/session-context.md:
# - Add "XSS on /api/search: all params HTML-encoded" to Ruled Out
```

## 11. Context Restoration on Recovery

```bash
# On recovery (oc-recover), context is automatically restored:
if [ -f tmp/session-context.md ] && [ -f tmp/session-context.md.archive ]; then
  # Check if current context is too small (stream failure indicator)
  CURRENT=$(wc -l < tmp/session-context.md)
  ARCHIVED=$(wc -l < tmp/session-context.md.archive)
  
  if [ "$CURRENT" -lt "$((ARCHIVED / 2))" ]; then
    echo "[!] Current context seems truncated â€” restoring from archive"
    cp tmp/session-context.md.archive tmp/session-context.md
  fi
fi
```

## 12. Checklist Templates

### Lead Status Template

```markdown
### {ID}: {Vuln Class} on {Endpoint}
- Status: {queued / testing / confirmed / ruled-out / needs-chain}
- Hypothesis: {one line}
- Impact target: {what this leads to if confirmed}
- Evidence: {file paths to request/response dumps}
- Last tested: {timestamp}
- Tested by: {@agent}
- Chains: {ID1, ID2}
- Notes: {any important context}
```

### Phase Completion Template

```markdown
## Phase {N} â€” {Phase Name}
- Status: {not-started / in-progress / complete}
- Started: {timestamp}
- Completed: {timestamp}
- Agent: {@agent}
- Artifacts: {file paths to produced artifacts}
- Key findings: {bullet list of important discoveries}
- Blockers: {anything that stopped progress}
- Next phase: {Phase N+1}
```

### Cross-Session State Template

```markdown
## Cross-Session State
- Engagement days: {N}
- Total findings: {N} confirmed, {N} ruled-out
- Current phase: {Phase N}
- Active agents: {@list}
- Key progress: {summary of what's been accomplished across sessions}
- What's changed: {new endpoints, changed behavior, fixed issues}
```
