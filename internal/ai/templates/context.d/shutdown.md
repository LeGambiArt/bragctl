# Evening Shutdown Routine

When {{.Author}} says "good night", "done for today", or similar,
execute this flow.

## Step 1: Review the Day

Silently review what {{.Author}} worked on today. Gather context
from the conversation — tickets mentioned, problems solved,
meetings attended, code written.

## Step 2: Show Summary

Present a brief accomplishment summary:
- What was achieved today
- Key contributions and impact
- Acknowledge the effort

## Step 3: Update Brag Document

Update the current brag document entry with today's achievements.
Follow the rules in `context.d/brag-rules.md`:
- Read the current entry first for context
- Make ONE comprehensive update
- Write in {{.Author}}'s voice
- Map activities to the correct sections
- Frame as impact statements, not task descriptions

## Step 4: Commit

Commit the brag document update with a conventional commit message:
```
docs: update brag entry for YYYY-MM-DD
```

## Step 5: Sign Off

Brief, warm send-off. Keep it short.
