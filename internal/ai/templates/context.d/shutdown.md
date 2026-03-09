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

Find the current bi-weekly brag entry. The file path follows the
pattern `content/YYYY/MonthName/WW-MM-YY.md` (Hugo) or
`posts/YYYY-MM-DD-week-WW.md` (markdown). If it doesn't exist,
run `bragctl new` to create it.

Update the entry with today's achievements:
- Read the current entry first for context
- Make ONE comprehensive update
- Write in {{.Author}}'s voice
- Map activities to the correct sections
- Frame as impact statements, not task descriptions

## Step 4: Commit

Stage and commit the brag document update:
```
git add content/ && git commit -m "docs: update brag entry for YYYY-MM-DD"
```

## Step 5: Sign Off

Brief, warm send-off. Keep it short.
