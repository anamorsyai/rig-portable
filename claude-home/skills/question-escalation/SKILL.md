---
name: question-escalation
description: Question escalation chain â€” when a subagent needs user input, questions bubble UP through the agent hierarchy to the orchestrator, which presents to user, then re-delegates back down with the answer.
---

# Question Escalation Chain

## The Problem

Subagents sometimes need user input to proceed:
- "Which of these 3 endpoints should I test first?"
- "Should I continue or switch targets?"
- "I found 2 possible paths â€” which one?"

If a subagent stops and waits for user input, it burns context and blocks
the pipeline. The user may not see the question for minutes/hours.

## The Solution

Questions bubble UP through the hierarchy. The orchestrator (build) is the
ONLY agent that talks to the user. Subagents never talk to the user directly.

## Flow

```
Subagent has question
    â†“
Subagent returns to parent with QUESTION flag
    â†“
Parent (orchestrator) receives question
    â†“
Orchestrator presents to user via question tool
    â†“
User answers
    â†“
Orchestrator re-delegates to original subagent with answer
    â†“
Subagent continues with the answer
```

## Protocol

### Step 1: Subagent returns question

When a subagent needs user input, it returns a structured question:

```
## QUESTION FOR USER
Priority: high|medium|low
Context: <1-2 sentence context for the question>
Question: <the actual question>
Options:
1. <option A>
2. <option B>
3. <option C>
Recommendation: <which option the agent recommends and why>
```

### Step 2: Orchestrator receives and presents

The orchestrator:
1. Reads the question
2. Presents it to the user via the `question` tool
3. Waits for the user's answer

### Step 3: Orchestrator re-delegates

After getting the answer, the orchestrator:
1. Calls the original subagent again with the answer in the prompt
2. The subagent continues from where it left off

```
@subagent: "User answered your question: [ANSWER]. Continue your task
using this decision."
```

## Rules

### For subagents
- **NEVER use the `question` tool directly** â€” you don't have access
- **NEVER stop and wait for user input** â€” return the question to your parent
- **NEVER output questions as plain text** â€” use the structured format above
- **Only ask questions that ACTUALLY need user input** â€” don't ask what you can decide yourself
- **Recommend an option** â€” always include your recommendation so the user can quick-approve

### For orchestrator (build)
- **ALWAYS present subagent questions to the user** â€” never answer for them
- **ALWAYS include the recommendation** â€” make it easy for the user
- **ALWAYS re-delegate after getting the answer** â€” don't lose the thread
- **Track which subagent asked** â€” so you re-delegate to the right one

### What qualifies as a question
- Choosing between multiple valid approaches
- Confirming scope boundary (is this in scope?)
- Priority decision (which target first?)
- Resource allocation (which agent next?)
- Go/no-go decision (should we continue or stop?)
- Preference (aggressive or conservative approach?)

### What does NOT qualify
- Tool errors (fix it yourself or call @fixer)
- Missing tools (install it yourself)
- Routine decisions (decide based on methodology)
- Scope questions (check scope.md yourself)

## Example

### Subagent returns:
```
## QUESTION FOR USER
Priority: high
Context: I found 3 potential IDOR endpoints on api.target.com.
Question: Which should I test first?
Options:
1. /api/v1/users/{id}/profile (likely PII)
2. /api/v1/users/{id}/billing (financial data)
3. /api/v1/users/{id}/documents (file access)
Recommendation: Option 2 (billing = highest bounty potential)
```

### Orchestrator presents:
```
@vuln found 3 potential IDOR endpoints. Which should it test first?

1. /api/v1/users/{id}/profile (likely PII)
2. /api/v1/users/{id}/billing (financial data) â€” RECOMMENDED
3. /api/v1/users/{id}/documents (file access)
```

### User answers:
"2"

### Orchestrator re-delegates:
```
@vuln: "User chose option 2 â€” test /api/v1/users/{id}/billing first.
Continue your IDOR testing starting with the billing endpoint."
```

## Integration with existing systems

### With /continue
When resuming a session, check if any subagent had a pending question.
If yes, re-present it to the user before continuing.

### With oc-checkpoint
Log questions in checkpoints:
```
oc-checkpoint question @vuln "Which endpoint to test first?"
```

### With STATE.md
Track pending questions in STATE.md:
```
## Pending Questions
| Agent | Question | Status |
|-------|----------|--------|
| @vuln | Which endpoint first? | ANSWERED: billing |
```
