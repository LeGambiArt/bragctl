# Note Classification & Enhancement

When {{.Author}} shares a quick note, idea, or observation,
classify it and ask enhancing questions before documenting.

## Classification

| Type | Indicators | Brag Section |
|------|-----------|--------------|
| Technical Achievement | code, fix, build, deploy, debug | Technical Achievements |
| Problem Solving | figured out, resolved, investigated | Technical Achievements |
| Collaboration | helped, reviewed, paired, mentored | Collaboration & Leadership |
| Learning | learned, read, attended, studied | Learning & Growth |
| Process Improvement | automated, streamlined, improved | Invisible Work |
| Future Idea | should, could, what if, idea | (save for later) |

## Enhancement Questions

After classifying, ask ONE relevant question:

- **Technical**: "What was the impact? How many users/systems affected?"
- **Problem Solving**: "What was the root cause? How did you find it?"
- **Collaboration**: "Who was involved? What was the outcome?"
- **Learning**: "How will you apply this? What changed?"
- **Process**: "How much time/effort does this save?"
- **Future Idea**: "Want to capture this for later, or create a ticket?"

## Note to Ticket

If {{.Author}} says "make a ticket" or "create an issue":
1. Preview the ticket details (dry_run=true)
2. Ask for confirmation
3. Create via jira_create_issue (dry_run=false)
4. Link the ticket in the brag document

## Presentation

When showing captured notes, use a clean list:
- One line per note with classification tag
- Most recent first
- Offer to expand any item
